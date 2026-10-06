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
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"github.com/NVIDIA/infra-controller/providers/fulfillment/internal"
)

// fulfillmentProviderServer implements providerv1.NicoProviderServer by
// delegating HTTP handling to the existing fulfillment provider via an
// internal Echo instance, and running a Temporal worker for workflows.
type fulfillmentProviderServer struct {
	providerv1.UnimplementedNicoProviderServer

	provider *fulfillment.FulfillmentProvider
	echo     *echo.Echo

	temporalClient client.Client
	temporalWorker tsdkWorker.Worker
}

// NewFulfillmentProviderServer creates a new gRPC server backed by the
// fulfillment provider. The provider is initialized lazily via Init().
func NewFulfillmentProviderServer() *fulfillmentProviderServer {
	s := &fulfillmentProviderServer{
		provider: fulfillment.New(),
		echo:     echo.New(),
	}

	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *fulfillmentProviderServer) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         s.provider.Name(),
		Version:      s.provider.Version(),
		Features:     s.provider.Features(),
		Dependencies: s.provider.Dependencies(),
	}, nil
}

// Init initializes the fulfillment provider.
func (s *fulfillmentProviderServer) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing fulfillment provider")

	// --- Database (optional) ---
	var dbSession *cdb.Session
	dbCfg, err := cdb.ConfigFromEnv()
	if err == nil {
		dbSession, err = cdb.NewSessionFromConfig(context.Background(), dbCfg)
		if err != nil {
			log.Warn().Err(err).Msg("database connection failed, using in-memory store")
			dbSession = nil
		}
	}

	if err := s.provider.InitStandalone(dbSession); err != nil {
		return initFailed("initialization failed: %v", err), nil
	}

	// Register routes on the internal Echo instance.
	group := s.echo.Group("")
	s.provider.RegisterRoutes(group)

	// Connect to Temporal and start a worker for fulfillment workflows.
	if req.GetTemporalEndpoint() != "" {
		tc, err := client.Dial(client.Options{
			HostPort:  req.GetTemporalEndpoint(),
			Namespace: req.GetTemporalNamespace(),
		})
		if err != nil {
			return initFailed("failed to connect to Temporal: %v", err), nil
		}
		s.temporalClient = tc

		w := tsdkWorker.New(tc, s.provider.TaskQueue(), tsdkWorker.Options{})
		s.provider.RegisterWorkflows(w)
		s.provider.RegisterActivities(w)

		if err := w.Start(); err != nil {
			tc.Close()
			return initFailed("failed to start Temporal worker: %v", err), nil
		}
		s.temporalWorker = w

		log.Info().
			Str("task_queue", s.provider.TaskQueue()).
			Msg("Temporal worker started")
	}

	log.Info().Msg("fulfillment provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "fulfillment provider ready",
	}, nil
}

// Shutdown gracefully stops the provider, including the Temporal worker.
func (s *fulfillmentProviderServer) Shutdown(ctx context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down fulfillment provider")

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
func (s *fulfillmentProviderServer) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles.
func (s *fulfillmentProviderServer) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// Order endpoints
			{Method: http.MethodPost, Path: prefix + "/catalog/orders"},
			{Method: http.MethodGet, Path: prefix + "/catalog/orders"},
			{Method: http.MethodGet, Path: prefix + "/catalog/orders/:id"},
			{Method: http.MethodDelete, Path: prefix + "/catalog/orders/:id"},

			// Service endpoints
			{Method: http.MethodGet, Path: prefix + "/services"},
			{Method: http.MethodGet, Path: prefix + "/services/:id"},
			{Method: http.MethodPatch, Path: prefix + "/services/:id"},
			{Method: http.MethodDelete, Path: prefix + "/services/:id"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router.
func (s *fulfillmentProviderServer) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns an empty list. The fulfillment provider
// does not register any hooks.
func (s *fulfillmentProviderServer) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook is a no-op — the fulfillment provider has no sync hooks.
func (s *fulfillmentProviderServer) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *fulfillmentProviderServer) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
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
