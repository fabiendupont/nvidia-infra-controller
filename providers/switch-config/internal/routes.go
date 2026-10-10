// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"net/http"

	echo "github.com/labstack/echo/v4"
	"go.temporal.io/sdk/client"
)

// registerRoutes wires the provider's HTTP routes onto the internal Echo instance.
func registerRoutes(e *echo.Echo, s *Server) {
	g := e.Group("/api/v1/switch-config")
	g.GET("/configs/:switchID", s.handleGetConfig("cli"))
	g.GET("/configs/:switchID/nvue", s.handleGetConfig("nvue-json"))
	g.POST("/configs/:switchID/render", s.handleRenderConfig)
	g.GET("/validation/:siteID", s.handleGetValidation)
	g.POST("/actions/validate-cables", s.handleValidateCables)
}

func (s *Server) handleGetConfig(format string) echo.HandlerFunc {
	return func(c echo.Context) error {
		switchID := c.Param("switchID")
		if s.store == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "store not available"})
		}
		rec, err := s.store.Latest(c.Request().Context(), switchID, format)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "no config found"})
		}
		content, err := Decompress(rec)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "decompress error"})
		}
		c.Response().Header().Set("X-Content-Hash", rec.ContentHash)
		c.Response().Header().Set("Content-Type", "text/plain")
		_, _ = c.Response().Write(content)
		return nil
	}
}

func (s *Server) handleRenderConfig(c echo.Context) error {
	switchID := c.Param("switchID")
	ctx := c.Request().Context()

	sw, err := s.switchClient.GetSwitch(ctx, switchID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "switch not found: " + err.Error()})
	}
	rc, err := s.switchClient.GetRoutingConfig(ctx, switchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "routing config error: " + err.Error()})
	}

	tmplSrc, err := DefaultTemplate("cumulus", "cli")
	if err != nil {
		tmplSrc = "# no default template for this platform\n"
	}

	renderCtx := RenderContext{Switch: sw, RoutingConfig: rc, SiteID: sw.GetSiteId()}
	rendered, err := RenderCLI(tmplSrc, renderCtx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "render error: " + err.Error()})
	}

	if s.store != nil {
		rec, err := s.store.Store(ctx, switchID, sw.GetSiteId(), "cli", "api", "rendered via HTTP", rendered)
		if err == nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"switch_id":    switchID,
				"format":       "cli",
				"content_hash": rec.ContentHash,
			})
		}
	}
	c.Response().Header().Set("Content-Type", "text/plain")
	_, _ = c.Response().Write(rendered)
	return nil
}

func (s *Server) handleGetValidation(c echo.Context) error {
	siteID := c.Param("siteID")
	if s.store == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "store not available"})
	}
	results, err := s.store.LatestValidationResults(c.Request().Context(), siteID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"site_id": siteID, "results": results})
}

func (s *Server) handleValidateCables(c echo.Context) error {
	type req struct {
		SiteID string `json:"site_id"`
	}
	var body req
	if err := c.Bind(&body); err != nil || body.SiteID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "site_id required"})
	}

	// For the HTTP endpoint we run activities directly in a goroutine.
	// Production deployments should dispatch via Temporal for durability.
	go func() {
		ctx := context.Background()
		topology, err := s.activities.FetchExpectedTopologyActivity(ctx, body.SiteID)
		if err != nil {
			return
		}
		var allResults []*ValidationResult
		for _, sw := range topology.Switches {
			observed, err := s.activities.FetchObservedLLDPActivity(ctx, body.SiteID, sw.GetId())
			if err != nil {
				continue
			}
			summary, _ := s.activities.ComputeMismatchesActivity(ctx, sw.GetId(), body.SiteID, topology, observed)
			allResults = append(allResults, summary.Results...)
		}
		_ = s.activities.ReportMismatchesActivity(ctx, body.SiteID, allResults)
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "cable validation started for site " + body.SiteID,
	})
}

// withTemporalClient is used when dispatching cable validation as a real
// Temporal workflow start (rather than the in-process goroutine above).
func (s *Server) startCableValidationWorkflow(ctx context.Context, tc client.Client, siteID string) error {
	opts := client.StartWorkflowOptions{
		ID:        "cable-validation-" + siteID,
		TaskQueue: taskQueue,
	}
	_, err := tc.ExecuteWorkflow(ctx, opts, CableValidationWorkflow, siteID)
	return err
}
