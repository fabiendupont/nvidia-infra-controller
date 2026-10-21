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
	// Async: dispatches Temporal workflow when tc is set, falls back to goroutine
	g.POST("/actions/validate-cables", s.handleValidateCables)
	g.POST("/site/:siteID/deploy", s.handleDeploySite)
	// Async OS upgrade (long-running — dispatched as Temporal workflow)
	g.POST("/switches/:switchID/upgrade", s.handleUpgradeOS)
	// Sync reads
	g.GET("/switches/:switchID/health", s.handleHardwareHealth)
	g.GET("/switches/:switchID/diagnostics", s.handleDiagnostics)
	// Workflow status polling
	g.GET("/workflow/:workflowID/status", s.handleWorkflowStatus)
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

	if s.configurator == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "configurator not available"})
	}
	rec, err := s.configurator.Push(ctx, switchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	resp := map[string]interface{}{
		"switch_id": switchID,
		"format":    "nvue-json",
	}
	if rec != nil {
		resp["content_hash"] = rec.ContentHash
		resp["id"] = rec.ID
	}
	return c.JSON(http.StatusOK, resp)
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

func (s *Server) handleUpgradeOS(c echo.Context) error {
	switchID := c.Param("switchID")
	type req struct {
		TargetVersion string `json:"target_version"`
	}
	var body req
	if err := c.Bind(&body); err != nil || body.TargetVersion == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "target_version required"})
	}
	if s.configurator == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "configurator not available"})
	}
	// Dispatch as Temporal workflow when client is available (upgrade can take 20+ min).
	if s.tc != nil {
		opts := client.StartWorkflowOptions{
			ID:        "os-upgrade-" + switchID,
			TaskQueue: taskQueue,
		}
		run, err := s.tc.ExecuteWorkflow(c.Request().Context(), opts, OSUpgradeWorkflow, switchID, body.TargetVersion)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusAccepted, map[string]string{
			"workflow_id":    run.GetID(),
			"run_id":         run.GetRunID(),
			"switch_id":      switchID,
			"target_version": body.TargetVersion,
		})
	}
	// Fallback: synchronous (no Temporal client — development/test only).
	if err := s.configurator.UpgradeOS(c.Request().Context(), switchID, body.TargetVersion); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, map[string]string{
		"switch_id":      switchID,
		"target_version": body.TargetVersion,
		"message":        "OS upgrade initiated (synchronous fallback)",
	})
}

// handleDeploySite dispatches SwitchConfigDeployWorkflow for all switches in a site.
func (s *Server) handleDeploySite(c echo.Context) error {
	siteID := c.Param("siteID")
	if siteID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "site_id required"})
	}
	if s.tc != nil {
		opts := client.StartWorkflowOptions{
			ID:        "switch-config-deploy-" + siteID,
			TaskQueue: taskQueue,
		}
		run, err := s.tc.ExecuteWorkflow(c.Request().Context(), opts, SwitchConfigDeployWorkflow, siteID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusAccepted, map[string]string{
			"workflow_id": run.GetID(),
			"run_id":      run.GetRunID(),
			"site_id":     siteID,
		})
	}
	// Fallback: push synchronously to each switch.
	go func() {
		ctx := context.Background()
		switches, err := s.activities.ListSwitchesForSiteActivity(ctx, siteID)
		if err != nil {
			return
		}
		for _, sw := range switches {
			_ = s.activities.PushConfigActivity(ctx, sw.SwitchID)
		}
	}()
	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "deploy started for site " + siteID + " (synchronous fallback)",
		"site_id": siteID,
	})
}

// handleWorkflowStatus returns the current status of a Temporal workflow run.
func (s *Server) handleWorkflowStatus(c echo.Context) error {
	wfID := c.Param("workflowID")
	if s.tc == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Temporal client not available"})
	}
	resp, err := s.tc.DescribeWorkflowExecution(c.Request().Context(), wfID, "")
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	info := resp.GetWorkflowExecutionInfo()
	return c.JSON(http.StatusOK, map[string]string{
		"workflow_id": info.GetExecution().GetWorkflowId(),
		"run_id":      info.GetExecution().GetRunId(),
		"status":      info.GetStatus().String(),
	})
}

func (s *Server) handleHardwareHealth(c echo.Context) error {
	switchID := c.Param("switchID")
	if s.configurator == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "configurator not available"})
	}
	result, err := s.configurator.ValidateHardware(c.Request().Context(), switchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleDiagnostics(c echo.Context) error {
	switchID := c.Param("switchID")
	if s.configurator == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "configurator not available"})
	}
	diag, err := s.configurator.GetDiagnostics(c.Request().Context(), switchID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, diag)
}

func (s *Server) handleValidateCables(c echo.Context) error {
	type req struct {
		SiteID string `json:"site_id"`
	}
	var body req
	if err := c.Bind(&body); err != nil || body.SiteID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "site_id required"})
	}

	// Dispatch as Temporal workflow for durability when client is available.
	if s.tc != nil {
		run, err := s.startCableValidationWorkflow(c.Request().Context(), s.tc, body.SiteID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusAccepted, map[string]string{
			"workflow_id": run.GetID(),
			"run_id":      run.GetRunID(),
			"site_id":     body.SiteID,
		})
	}

	// Fallback: run activities directly in a goroutine (dev/test only).
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

// startCableValidationWorkflow dispatches CableValidationWorkflow and returns the run.
func (s *Server) startCableValidationWorkflow(ctx context.Context, tc client.Client, siteID string) (client.WorkflowRun, error) {
	opts := client.StartWorkflowOptions{
		ID:        "cable-validation-" + siteID,
		TaskQueue: taskQueue,
	}
	return tc.ExecuteWorkflow(ctx, opts, CableValidationWorkflow, siteID)
}
