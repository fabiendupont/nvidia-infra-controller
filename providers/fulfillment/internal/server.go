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

package fulfillment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	sdkdb "github.com/NVIDIA/infra-controller/provider-sdk/db"
	"github.com/NVIDIA/infra-controller/providers/fulfillment/internal/migrations"
)

// Server implements the NicoProviderServer gRPC interface for the
// fulfillment provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer

	orderStore   OrderStoreInterface
	serviceStore ServiceStoreInterface
	echo         *echo.Echo

	temporalClient client.Client
	temporalWorker tsdkWorker.Worker
}

// NewServer creates a new fulfillment provider gRPC server.
func NewServer() *Server {
	s := &Server{
		echo: echo.New(),
	}
	s.echo.HideBanner = true
	s.echo.HidePort = true
	return s
}

// GetInfo returns the provider's metadata.
func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "nico-fulfillment",
		Version:      "0.1.0",
		Features:     []string{"fulfillment"},
		Dependencies: []string{"nico-networking", "nico-compute", "nico-catalog"},
	}, nil
}

// Init initializes the fulfillment provider. It creates in-memory stores,
// registers routes, connects to Temporal, and starts a workflow worker.
func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Int("service_endpoints", len(req.GetServiceEndpoints())).
		Msg("initializing fulfillment provider")

	// Connect to PostgreSQL in the fulfillment schema, falling back to
	// in-memory stores if no database is configured.
	dbCfg, err := sdkdb.ConfigFromEnv()
	if err == nil {
		session, dbErr := sdkdb.ConnectWithSchema(context.Background(), dbCfg, "fulfillment")
		if dbErr != nil {
			log.Warn().Err(dbErr).Msg("fulfillment DB connection failed; using in-memory stores")
			s.orderStore = NewOrderStore()
			s.serviceStore = NewServiceStore()
		} else {
			migrator := sdkdb.NewMigrator(session.DB, "fulfillment", migrations.All())
			if migrateErr := migrator.Run(context.Background()); migrateErr != nil {
				log.Warn().Err(migrateErr).Msg("fulfillment migration failed; using in-memory stores")
				s.orderStore = NewOrderStore()
				s.serviceStore = NewServiceStore()
			} else {
				s.orderStore = NewOrderSQLStore(session.DB)
				s.serviceStore = NewServiceSQLStore(session.DB)
				log.Info().Msg("fulfillment using PostgreSQL store (schema: fulfillment)")
			}
		}
	} else {
		log.Info().Msg("fulfillment DB not configured; using in-memory stores")
		s.orderStore = NewOrderStore()
		s.serviceStore = NewServiceStore()
	}

	// Register routes on the internal Echo instance.
	prefix := "/api/v1"
	orderHandler := NewOrderHandler(s.orderStore)
	serviceHandler := NewServiceHandler(s.serviceStore)

	group := s.echo.Group("")
	group.Add(http.MethodPost, prefix+"/catalog/orders", orderHandler.Create)
	group.Add(http.MethodGet, prefix+"/catalog/orders", orderHandler.List)
	group.Add(http.MethodGet, prefix+"/catalog/orders/:id", orderHandler.Get)
	group.Add(http.MethodDelete, prefix+"/catalog/orders/:id", orderHandler.Cancel)
	group.Add(http.MethodGet, prefix+"/services", serviceHandler.List)
	group.Add(http.MethodGet, prefix+"/services/:id", serviceHandler.Get)
	group.Add(http.MethodPatch, prefix+"/services/:id", serviceHandler.Update)
	group.Add(http.MethodDelete, prefix+"/services/:id", serviceHandler.Delete)

	// Connect to Temporal and start a worker for fulfillment workflows.
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

		w := tsdkWorker.New(tc, "fulfillment-tasks", tsdkWorker.Options{})

		// Register workflows
		w.RegisterWorkflow(TenantProvisioningWorkflow)
		w.RegisterWorkflow(TenantTeardownWorkflow)
		w.RegisterWorkflow(ServiceScaleWorkflow)
		w.RegisterWorkflow(BlueprintExecutionWorkflow)

		// Register activities
		activities := &FulfillmentActivities{
			orderStore:   s.orderStore,
			serviceStore: s.serviceStore,
		}
		w.RegisterActivity(activities)

		execActivities := NewExecutionActivities(s.orderStore, s.serviceStore)
		w.RegisterActivity(execActivities)

		if err := w.Start(); err != nil {
			tc.Close()
			return &providerv1.InitResponse{
				Ready:   false,
				Message: fmt.Sprintf("failed to start Temporal worker: %v", err),
			}, nil
		}
		s.temporalWorker = w

		log.Info().
			Str("task_queue", "fulfillment-tasks").
			Msg("Temporal worker started")
	}

	log.Info().Msg("fulfillment provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "fulfillment provider ready",
	}, nil
}

// Shutdown gracefully stops the provider, including the Temporal worker.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down fulfillment provider")

	if s.temporalWorker != nil {
		s.temporalWorker.Stop()
	}
	if s.temporalClient != nil {
		s.temporalClient.Close()
	}

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

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response.
func (s *Server) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns an empty list. The fulfillment provider does
// not register any hooks.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook is a no-op.
func (s *Server) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}
