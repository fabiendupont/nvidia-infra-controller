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
	"strconv"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	tsdkClient "go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	"github.com/NVIDIA/infra-controller/rest-api/api/internal/config"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/providers/networking"
	subnetActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/subnet"
	vpcActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/vpc"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// networkingProviderServer implements providerv1.NicoProviderServer by delegating
// HTTP handling to the existing networking provider via an internal Echo instance
// and running a Temporal worker for workflow processing.
type networkingProviderServer struct {
	providerv1.UnimplementedNicoProviderServer

	provider *networking.NetworkingProvider
	echo     *echo.Echo
	worker   tsdkWorker.Worker
	tc       tsdkClient.Client
}

// NewNetworkingProviderServer creates a new gRPC server backed by the
// networking provider. The provider is not yet initialized; call Init via
// gRPC to complete setup.
func NewNetworkingProviderServer() *networkingProviderServer {
	s := &networkingProviderServer{
		provider: networking.New(),
		echo:     echo.New(),
	}

	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *networkingProviderServer) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         s.provider.Name(),
		Version:      s.provider.Version(),
		Features:     s.provider.Features(),
		Dependencies: s.provider.Dependencies(),
	}, nil
}

// Init initializes the networking provider with database, Temporal, and
// configuration from environment variables and the gRPC InitRequest.
func (s *networkingProviderServer) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing networking provider")

	// --- Database ---
	dbHost := envOrDefault("DB_HOST", "localhost")
	dbPortStr := envOrDefault("DB_PORT", "5432")
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return initFailed("invalid DB_PORT: %v", err), nil
	}
	dbName := envOrDefault("DB_NAME", "nico")
	dbUser := envOrDefault("DB_USER", "nico")
	dbPass := envOrDefault("DB_PASSWORD", "")

	dbSession, err := cdb.NewSession(context.Background(), dbHost, dbPort, dbName, dbUser, dbPass, "")
	if err != nil {
		return initFailed("database connection failed: %v", err), nil
	}

	// --- Temporal ---
	temporalEndpoint := req.GetTemporalEndpoint()
	if temporalEndpoint == "" {
		temporalEndpoint = envOrDefault("TEMPORAL_ENDPOINT", "localhost:7233")
	}
	temporalNamespace := req.GetTemporalNamespace()
	if temporalNamespace == "" {
		temporalNamespace = envOrDefault("TEMPORAL_NAMESPACE", "default")
	}
	temporalQueue := envOrDefault("TEMPORAL_QUEUE", "nico-task-queue")

	tc, err := tsdkClient.Dial(tsdkClient.Options{
		HostPort:  temporalEndpoint,
		Namespace: temporalNamespace,
	})
	if err != nil {
		return initFailed("temporal connection failed: %v", err), nil
	}
	s.tc = tc

	// --- Config ---
	cfg := config.NewConfig()

	// --- Initialize provider ---
	if err := s.provider.InitStandalone(dbSession, tc, cfg, temporalNamespace, temporalQueue); err != nil {
		return initFailed("provider initialization failed: %v", err), nil
	}

	// Register routes on the internal Echo instance.
	group := s.echo.Group("")
	s.provider.RegisterRoutes(group)

	// --- Temporal worker ---
	s.worker = tsdkWorker.New(tc, s.provider.TaskQueue(), tsdkWorker.Options{})
	s.provider.RegisterWorkflows(s.worker)

	// --- Prometheus lifecycle metrics ---
	reg := prometheus.NewRegistry()

	vpcLifecycleMetrics := vpcActivity.NewManageVpcLifecycleMetrics(reg, dbSession, temporalNamespace)
	s.worker.RegisterActivity(&vpcLifecycleMetrics)

	subnetLifecycleMetrics := subnetActivity.NewManageSubnetLifecycleMetrics(reg, dbSession, temporalNamespace)
	s.worker.RegisterActivity(&subnetLifecycleMetrics)

	metricsAddr := envOrDefault("METRICS_ADDR", ":9090")
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		log.Info().Str("addr", metricsAddr).Msg("starting metrics server")
		if err := http.ListenAndServe(metricsAddr, mux); err != nil {
			log.Error().Err(err).Msg("metrics server exited with error")
		}
	}()

	go func() {
		if err := s.worker.Run(tsdkWorker.InterruptCh()); err != nil {
			log.Error().Err(err).Msg("temporal worker exited with error")
		}
	}()

	log.Info().Msg("networking provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "networking provider ready",
	}, nil
}

// Shutdown gracefully stops the provider and its Temporal worker.
func (s *networkingProviderServer) Shutdown(ctx context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down networking provider")

	if s.worker != nil {
		s.worker.Stop()
	}
	if s.tc != nil {
		s.tc.Close()
	}

	_ = s.provider.Shutdown(ctx)
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (s *networkingProviderServer) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles.
func (s *networkingProviderServer) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// VPC
			{Method: http.MethodPost, Path: prefix + "/vpc"},
			{Method: http.MethodGet, Path: prefix + "/vpc"},
			{Method: http.MethodGet, Path: prefix + "/vpc/:id"},
			{Method: http.MethodPatch, Path: prefix + "/vpc/:id"},
			{Method: http.MethodDelete, Path: prefix + "/vpc/:id"},
			{Method: http.MethodPatch, Path: prefix + "/vpc/:id/virtualization"},

			// VpcPrefix
			{Method: http.MethodPost, Path: prefix + "/vpc-prefix"},
			{Method: http.MethodGet, Path: prefix + "/vpc-prefix"},
			{Method: http.MethodGet, Path: prefix + "/vpc-prefix/:id"},
			{Method: http.MethodPatch, Path: prefix + "/vpc-prefix/:id"},
			{Method: http.MethodDelete, Path: prefix + "/vpc-prefix/:id"},

			// IPBlock
			{Method: http.MethodPost, Path: prefix + "/ipblock"},
			{Method: http.MethodGet, Path: prefix + "/ipblock"},
			{Method: http.MethodGet, Path: prefix + "/ipblock/:id"},
			{Method: http.MethodGet, Path: prefix + "/ipblock/:id/derived"},
			{Method: http.MethodPatch, Path: prefix + "/ipblock/:id"},
			{Method: http.MethodDelete, Path: prefix + "/ipblock/:id"},

			// Subnet
			{Method: http.MethodPost, Path: prefix + "/subnet"},
			{Method: http.MethodGet, Path: prefix + "/subnet"},
			{Method: http.MethodGet, Path: prefix + "/subnet/:id"},
			{Method: http.MethodPatch, Path: prefix + "/subnet/:id"},
			{Method: http.MethodDelete, Path: prefix + "/subnet/:id"},

			// NetworkSecurityGroup
			{Method: http.MethodPost, Path: prefix + "/network-security-group"},
			{Method: http.MethodGet, Path: prefix + "/network-security-group"},
			{Method: http.MethodGet, Path: prefix + "/network-security-group/:id"},
			{Method: http.MethodPatch, Path: prefix + "/network-security-group/:id"},
			{Method: http.MethodDelete, Path: prefix + "/network-security-group/:id"},

			// Interface
			{Method: http.MethodGet, Path: prefix + "/instance/:instanceId/interface"},

			// InfiniBandInterface
			{Method: http.MethodGet, Path: prefix + "/instance/:instanceId/infiniband-interface"},
			{Method: http.MethodGet, Path: prefix + "/infiniband-interface"},

			// NVLinkInterface
			{Method: http.MethodGet, Path: prefix + "/instance/:instanceId/nvlink-interface"},
			{Method: http.MethodGet, Path: prefix + "/nvlink-interface"},

			// InfiniBandPartition
			{Method: http.MethodPost, Path: prefix + "/infiniband-partition"},
			{Method: http.MethodGet, Path: prefix + "/infiniband-partition"},
			{Method: http.MethodGet, Path: prefix + "/infiniband-partition/:id"},
			{Method: http.MethodPatch, Path: prefix + "/infiniband-partition/:id"},
			{Method: http.MethodDelete, Path: prefix + "/infiniband-partition/:id"},

			// NVLinkLogicalPartition
			{Method: http.MethodPost, Path: prefix + "/nvlink-logical-partition"},
			{Method: http.MethodGet, Path: prefix + "/nvlink-logical-partition"},
			{Method: http.MethodGet, Path: prefix + "/nvlink-logical-partition/:id"},
			{Method: http.MethodPatch, Path: prefix + "/nvlink-logical-partition/:id"},
			{Method: http.MethodDelete, Path: prefix + "/nvlink-logical-partition/:id"},

			// DPU Extension Service
			{Method: http.MethodPost, Path: prefix + "/dpu-extension-service"},
			{Method: http.MethodGet, Path: prefix + "/dpu-extension-service"},
			{Method: http.MethodGet, Path: prefix + "/dpu-extension-service/:id"},
			{Method: http.MethodPatch, Path: prefix + "/dpu-extension-service/:id"},
			{Method: http.MethodDelete, Path: prefix + "/dpu-extension-service/:id"},
			{Method: http.MethodGet, Path: prefix + "/dpu-extension-service/:id/version/:version"},
			{Method: http.MethodDelete, Path: prefix + "/dpu-extension-service/:id/version/:version"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response. This bridges gRPC transport to the existing Echo-based
// handler logic.
func (s *networkingProviderServer) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns the hook events that the networking provider
// fires. These are async hooks emitted by VPC workflow activities.
func (s *networkingProviderServer) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			{
				Type:    providerv1.HookRegistration_ASYNC,
				Feature: "networking",
				Event:   "post-create-vpc",
			},
			{
				Type:    providerv1.HookRegistration_ASYNC,
				Feature: "networking",
				Event:   "post-delete-vpc",
			},
		},
	}, nil
}

// HandleSyncHook dispatches a synchronous hook invocation. The networking
// provider does not register any sync hooks, so all calls are rejected.
func (s *networkingProviderServer) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment. The networking
// provider's OpenAPI spec is not yet extracted as a standalone fragment.
func (s *networkingProviderServer) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initFailed(format string, args ...interface{}) *providerv1.InitResponse {
	msg := fmt.Sprintf(format, args...)
	log.Error().Msg(msg)
	return &providerv1.InitResponse{
		Ready:   false,
		Message: msg,
	}
}
