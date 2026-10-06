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

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	"github.com/NVIDIA/infra-controller/rest-api/providers/compute"
	instanceActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/instance"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// computeProviderServer implements providerv1.NicoProviderServer by
// delegating HTTP handling to the existing compute provider via an
// internal Echo instance, and running a Temporal worker for workflows.
type computeProviderServer struct {
	providerv1.UnimplementedNicoProviderServer

	provider *compute.ComputeProvider
	echo     *echo.Echo

	temporalClient client.Client
	temporalWorker tsdkWorker.Worker
}

// NewComputeProviderServer creates a new gRPC server backed by the
// compute provider. The provider is initialized lazily via Init().
func NewComputeProviderServer() *computeProviderServer {
	s := &computeProviderServer{
		provider: compute.New(),
		echo:     echo.New(),
	}

	// Silence Echo's default logger — we use zerolog.
	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *computeProviderServer) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         s.provider.Name(),
		Version:      s.provider.Version(),
		Features:     s.provider.Features(),
		Dependencies: s.provider.Dependencies(),
	}, nil
}

// Init initializes the compute provider. It connects to PostgreSQL,
// registers routes, connects to Temporal, and starts a workflow worker.
func (s *computeProviderServer) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Int("service_endpoints", len(req.GetServiceEndpoints())).
		Msg("initializing compute provider")

	// Initialize the provider with DB from environment variables.
	if err := s.provider.InitStandalone(); err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("initialization failed: %v", err),
		}, nil
	}

	// Register routes on the internal Echo instance.
	group := s.echo.Group("")
	s.provider.RegisterRoutes(group)

	// Connect to Temporal and start a worker for compute workflows.
	if req.GetTemporalEndpoint() != "" {
		tc, err := client.Dial(client.Options{
			HostPort:  req.GetTemporalEndpoint(),
			Namespace: req.GetTemporalNamespace(),
		})
		if err != nil {
			return &providerv1.InitResponse{
				Ready:   false,
				Message: fmt.Sprintf("failed to connect to Temporal: %v", err),
			}, nil
		}
		s.temporalClient = tc

		w := tsdkWorker.New(tc, s.provider.TaskQueue(), tsdkWorker.Options{})
		s.provider.RegisterWorkflows(w)
		s.provider.RegisterActivities(w)

		// --- Prometheus lifecycle metrics ---
		reg := prometheus.NewRegistry()

		instanceLifecycleMetrics := instanceActivity.NewManageInstanceLifecycleMetrics(reg, s.provider.DBSession())
		w.RegisterActivity(&instanceLifecycleMetrics)

		metricsAddr := envOrDefault("METRICS_ADDR", ":9090")
		go func() {
			mux := http.NewServeMux()
			mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
			log.Info().Str("addr", metricsAddr).Msg("starting metrics server")
			if err := http.ListenAndServe(metricsAddr, mux); err != nil {
				log.Error().Err(err).Msg("metrics server exited with error")
			}
		}()

		if err := w.Start(); err != nil {
			tc.Close()
			return &providerv1.InitResponse{
				Ready:   false,
				Message: fmt.Sprintf("failed to start Temporal worker: %v", err),
			}, nil
		}
		s.temporalWorker = w

		log.Info().
			Str("task_queue", s.provider.TaskQueue()).
			Msg("Temporal worker started")
	}

	log.Info().Msg("compute provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "compute provider ready",
	}, nil
}

// Shutdown gracefully stops the provider, including the Temporal worker.
func (s *computeProviderServer) Shutdown(ctx context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down compute provider")

	if s.temporalWorker != nil {
		s.temporalWorker.Stop()
	}
	if s.temporalClient != nil {
		s.temporalClient.Close()
	}

	_ = s.provider.Shutdown(ctx)
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (s *computeProviderServer) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles. The route list
// matches what RegisterRoutes registers on the Echo group.
func (s *computeProviderServer) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// Instance endpoints
			{Method: http.MethodPost, Path: prefix + "/instance"},
			{Method: http.MethodPost, Path: prefix + "/instance/batch"},
			{Method: http.MethodGet, Path: prefix + "/instance"},
			{Method: http.MethodGet, Path: prefix + "/instance/:id"},
			{Method: http.MethodPatch, Path: prefix + "/instance/:id"},
			{Method: http.MethodDelete, Path: prefix + "/instance/:id"},
			{Method: http.MethodGet, Path: prefix + "/instance/:id/status-history"},

			// Instance Type endpoints
			{Method: http.MethodPost, Path: prefix + "/instance/type"},
			{Method: http.MethodGet, Path: prefix + "/instance/type"},
			{Method: http.MethodGet, Path: prefix + "/instance/type/:id"},
			{Method: http.MethodPatch, Path: prefix + "/instance/type/:id"},
			{Method: http.MethodDelete, Path: prefix + "/instance/type/:id"},

			// Interface endpoints under instance
			{Method: http.MethodGet, Path: prefix + "/instance/:instanceId/interface"},
			{Method: http.MethodGet, Path: prefix + "/instance/:instanceId/infiniband-interface"},
			{Method: http.MethodGet, Path: prefix + "/instance/:instanceId/nvlink-interface"},

			// Machine endpoints
			{Method: http.MethodGet, Path: prefix + "/machine"},
			{Method: http.MethodGet, Path: prefix + "/machine/:id"},
			{Method: http.MethodPatch, Path: prefix + "/machine/:id"},
			{Method: http.MethodDelete, Path: prefix + "/machine/:id"},
			{Method: http.MethodGet, Path: prefix + "/machine/:id/status-history"},
			{Method: http.MethodGet, Path: prefix + "/machine/gpu/stats"},
			{Method: http.MethodGet, Path: prefix + "/machine/instance-type/stats/summary"},
			{Method: http.MethodGet, Path: prefix + "/machine/instance-type/stats"},

			// Machine/Instance Type association endpoints
			{Method: http.MethodPost, Path: prefix + "/instance/type/:instanceTypeId/machine"},
			{Method: http.MethodGet, Path: prefix + "/instance/type/:instanceTypeId/machine"},
			{Method: http.MethodDelete, Path: prefix + "/instance/type/:instanceTypeId/machine/:id"},

			// Allocation endpoints
			{Method: http.MethodPost, Path: prefix + "/allocation"},
			{Method: http.MethodGet, Path: prefix + "/allocation"},
			{Method: http.MethodGet, Path: prefix + "/allocation/:id"},
			{Method: http.MethodPatch, Path: prefix + "/allocation/:id"},
			{Method: http.MethodDelete, Path: prefix + "/allocation/:id"},

			// AllocationConstraint update endpoint
			{Method: http.MethodPatch, Path: prefix + "/allocation/:allocationId/constraint/:id"},

			// OperatingSystem endpoints
			{Method: http.MethodPost, Path: prefix + "/operating-system"},
			{Method: http.MethodGet, Path: prefix + "/operating-system"},
			{Method: http.MethodGet, Path: prefix + "/operating-system/:id"},
			{Method: http.MethodPatch, Path: prefix + "/operating-system/:id"},
			{Method: http.MethodDelete, Path: prefix + "/operating-system/:id"},

			// SSHKey endpoints
			{Method: http.MethodPost, Path: prefix + "/sshkey"},
			{Method: http.MethodGet, Path: prefix + "/sshkey"},
			{Method: http.MethodGet, Path: prefix + "/sshkey/:id"},
			{Method: http.MethodPatch, Path: prefix + "/sshkey/:id"},
			{Method: http.MethodDelete, Path: prefix + "/sshkey/:id"},

			// SSHKeyGroup endpoints
			{Method: http.MethodPost, Path: prefix + "/sshkeygroup"},
			{Method: http.MethodGet, Path: prefix + "/sshkeygroup"},
			{Method: http.MethodGet, Path: prefix + "/sshkeygroup/:id"},
			{Method: http.MethodPatch, Path: prefix + "/sshkeygroup/:id"},
			{Method: http.MethodDelete, Path: prefix + "/sshkeygroup/:id"},

			// Machine Capability endpoints
			{Method: http.MethodGet, Path: prefix + "/machine-capability"},

			// Machine Validation endpoints
			{Method: http.MethodPost, Path: prefix + "/site/:siteID/machine-validation/test"},
			{Method: http.MethodPatch, Path: prefix + "/site/:siteID/machine-validation/test/:id/version/:version"},
			{Method: http.MethodGet, Path: prefix + "/site/:siteID/machine-validation/test"},
			{Method: http.MethodGet, Path: prefix + "/site/:siteID/machine-validation/test/:id/version/:version"},
			{Method: http.MethodGet, Path: prefix + "/site/:siteID/machine-validation/machine/:machineID/results"},
			{Method: http.MethodGet, Path: prefix + "/site/:siteID/machine-validation/machine/:machineID/runs"},
			{Method: http.MethodGet, Path: prefix + "/site/:siteID/machine-validation/external-config"},
			{Method: http.MethodGet, Path: prefix + "/site/:siteID/machine-validation/external-config/:cfgName"},
			{Method: http.MethodPost, Path: prefix + "/site/:siteID/machine-validation/external-config"},
			{Method: http.MethodPatch, Path: prefix + "/site/:siteID/machine-validation/external-config/:cfgName"},
			{Method: http.MethodDelete, Path: prefix + "/site/:siteID/machine-validation/external-config/:cfgName"},

			// SKU endpoints
			{Method: http.MethodGet, Path: prefix + "/sku"},
			{Method: http.MethodGet, Path: prefix + "/sku/:id"},

			// Rack endpoints (RLA)
			{Method: http.MethodGet, Path: prefix + "/rack/task/:id"},
			{Method: http.MethodGet, Path: prefix + "/rack"},
			{Method: http.MethodGet, Path: prefix + "/rack/validation"},
			{Method: http.MethodPatch, Path: prefix + "/rack/power"},
			{Method: http.MethodPatch, Path: prefix + "/rack/firmware"},
			{Method: http.MethodPost, Path: prefix + "/rack/bringup"},
			{Method: http.MethodGet, Path: prefix + "/rack/:id"},
			{Method: http.MethodGet, Path: prefix + "/rack/:id/validation"},
			{Method: http.MethodPatch, Path: prefix + "/rack/:id/power"},
			{Method: http.MethodPatch, Path: prefix + "/rack/:id/firmware"},
			{Method: http.MethodPost, Path: prefix + "/rack/:id/bringup"},

			// Tray endpoints (RLA)
			{Method: http.MethodGet, Path: prefix + "/tray"},
			{Method: http.MethodPatch, Path: prefix + "/tray/power"},
			{Method: http.MethodPatch, Path: prefix + "/tray/firmware"},
			{Method: http.MethodGet, Path: prefix + "/tray/validation"},
			{Method: http.MethodGet, Path: prefix + "/tray/:id"},
			{Method: http.MethodPatch, Path: prefix + "/tray/:id/power"},
			{Method: http.MethodPatch, Path: prefix + "/tray/:id/firmware"},
			{Method: http.MethodGet, Path: prefix + "/tray/:id/validation"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response. This is the key method that bridges gRPC transport
// to the existing Echo-based handler logic.
func (s *computeProviderServer) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns an empty list. The compute provider does not
// register hooks of its own — it is a hook consumer (e.g., health provider
// registers pre-create-instance hooks on the compute feature). Hook firing
// in activities is handled through the HookRunner passed at Init time; in
// standalone mode hooks are dispatched by the core proxy.
func (s *computeProviderServer) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook is a no-op — the compute provider has no sync hooks.
func (s *computeProviderServer) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *computeProviderServer) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
