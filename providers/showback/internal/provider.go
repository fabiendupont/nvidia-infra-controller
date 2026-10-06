/*
 * SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package showback

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// Server implements the NicoProviderServer gRPC interface for the showback provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer

	store UsageStoreInterface
	rates map[string]RateEntry
	echo  *echo.Echo
}

// NewServer creates a new showback provider gRPC server.
func NewServer() *Server {
	s := &Server{
		echo: echo.New(),
	}

	// Silence Echo's default logger -- we use zerolog.
	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "nico-showback",
		Version:      "0.1.0",
		Features:     []string{"showback"},
		Dependencies: []string{"nico-compute"},
	}, nil
}

// Init initializes the showback provider with in-memory stores and default rates.
func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing showback provider")

	// TODO: use sdk.ConnectWithSchema(ctx, dsn, "showback") for SQL persistence (Phase 3).
	s.store = NewUsageStore() // replace with NewUsageSQLStore(db)

	// Default rate table -- maps metric names to per-unit costs.
	s.rates = map[string]RateEntry{
		"gpu-hours":        {Rate: 10.00, Currency: "USD"},
		"storage-gb-hours": {Rate: 0.015, Currency: "USD"},
	}

	// Register routes on the internal Echo instance.
	prefix := "/api/v1"
	group := s.echo.Group("")
	group.Add(http.MethodGet, prefix+"/services/:id/usage", s.handleGetServiceUsage)
	group.Add(http.MethodGet, prefix+"/self/usage", s.handleGetSelfUsage)
	group.Add(http.MethodGet, prefix+"/self/usage/costs", s.handleGetSelfUsageCosts)
	group.Add(http.MethodGet, prefix+"/self/quotas", s.handleGetSelfQuotas)

	log.Info().Msg("showback provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "showback provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("showback provider shutting down")
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (s *Server) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles.
func (s *Server) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			{Method: http.MethodGet, Path: prefix + "/services/:id/usage"},
			{Method: http.MethodGet, Path: prefix + "/self/usage"},
			{Method: http.MethodGet, Path: prefix + "/self/usage/costs"},
			{Method: http.MethodGet, Path: prefix + "/self/quotas"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response. This bridges gRPC transport to the existing Echo-based
// handler logic.
func (s *Server) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	// Build the HTTP request.
	var bodyReader io.Reader
	if len(req.GetBody()) > 0 {
		bodyReader = strings.NewReader(string(req.GetBody()))
	}

	path := req.GetPath()
	if len(req.GetQueryParams()) > 0 {
		params := make([]string, 0, len(req.GetQueryParams()))
		for k, v := range req.GetQueryParams() {
			params = append(params, k+"="+v)
		}
		path += "?" + strings.Join(params, "&")
	}

	httpReq := httptest.NewRequest(req.GetMethod(), path, bodyReader)
	for k, v := range req.GetHeaders() {
		httpReq.Header.Set(k, v)
	}
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()

	// Serve through the Echo router.
	s.echo.ServeHTTP(rec, httpReq)

	result := rec.Result()
	defer result.Body.Close()

	respBody, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	headers := make(map[string]string, len(result.Header))
	for k := range result.Header {
		headers[k] = result.Header.Get(k)
	}

	return &providerv1.HTTPResponse{
		StatusCode: int32(result.StatusCode),
		Headers:    headers,
		Body:       respBody,
	}, nil
}

// GetHookRegistrations returns the async reactions that the showback provider
// registers for instance lifecycle events.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "compute",
				Event:          "post-create-instance",
				TargetWorkflow: "showback-metering-watcher",
				SignalName:     "instance-created",
			},
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "compute",
				Event:          "post-delete-instance",
				TargetWorkflow: "showback-metering-watcher",
				SignalName:     "instance-deleted",
			},
		},
	}, nil
}

// HandleSyncHook dispatches a synchronous hook invocation. The showback
// provider does not register any sync hooks.
func (s *Server) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment. The showback provider's
// OpenAPI spec is not yet extracted as a standalone fragment.
func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}
