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

package health

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	sdkdb "github.com/NVIDIA/infra-controller/provider-sdk/db"
	"github.com/NVIDIA/infra-controller/providers/health/internal/migrations"
)

// Server implements providerv1.NicoProviderServer by delegating HTTP handling
// to the health provider business logic via an internal Echo instance.
type Server struct {
	providerv1.UnimplementedNicoProviderServer

	echo          *echo.Echo
	apiPathPrefix string

	// Stores
	faultStore             FaultStoreI
	serviceEventStore      ServiceEventStoreI
	faultServiceEventStore FaultServiceEventStoreI
	classificationStore    *ClassificationStore

	// Handlers
	faultHandler          *FaultHandler
	serviceEventHandler   *ServiceEventHandler
	classificationHandler *ClassificationHandler
	webhookHandler        *WebhookHandler

	// Metrics
	metrics         *FaultMetrics
	metricsRegistry *prometheus.Registry
}

// NewServer creates a new health provider gRPC server.
func NewServer() *Server {
	s := &Server{
		echo: echo.New(),
	}

	// Silence Echo's default logger — we use zerolog.
	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "nico-health",
		Version:      "2.0.0",
		Features:     []string{"health", "fault-management"},
		Dependencies: []string{"nico-compute"},
	}, nil
}

// Init initializes the health provider with PostgreSQL persistence when a
// database is configured, falling back to in-memory stores otherwise.
func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing health provider")

	s.apiPathPrefix = "/api/v1"

	// Connect to PostgreSQL in the health schema; fall back to in-memory.
	dbCfg, err := sdkdb.ConfigFromEnv()
	if err == nil {
		session, dbErr := sdkdb.ConnectWithSchema(context.Background(), dbCfg, "health")
		if dbErr != nil {
			log.Warn().Err(dbErr).Msg("health DB connection failed; using in-memory stores")
		} else {
			migrator := sdkdb.NewMigrator(session.DB, "health", migrations.All())
			if migrateErr := migrator.Run(context.Background()); migrateErr != nil {
				log.Warn().Err(migrateErr).Msg("health migration failed; using in-memory stores")
			} else {
				s.faultStore = NewFaultEventSQLStore(session.DB)
				s.serviceEventStore = NewServiceEventSQLStore(session.DB)
				s.faultServiceEventStore = NewFaultServiceEventStore()
				log.Info().Msg("health using PostgreSQL stores (schema: health)")
			}
		}
	}
	if s.faultStore == nil {
		log.Info().Msg("health using in-memory stores")
		s.faultStore = NewFaultStore()
		s.serviceEventStore = NewServiceEventStore()
		s.faultServiceEventStore = NewFaultServiceEventStore()
	}

	s.classificationStore = NewClassificationStore(loadDefaultClassifications())

	// Create handlers
	s.faultHandler = NewFaultHandler(s.faultStore, s.classificationStore)
	s.serviceEventHandler = NewServiceEventHandler(s.serviceEventStore, s.faultServiceEventStore)
	s.classificationHandler = NewClassificationHandler(s.classificationStore)
	s.webhookHandler = NewWebhookHandler(s.faultStore, s.classificationStore)

	// Create Prometheus metrics
	s.metricsRegistry = prometheus.NewRegistry()
	s.metrics = NewFaultMetrics(s.metricsRegistry, s.faultStore)

	// Register routes on the internal Echo instance.
	group := s.echo.Group("")
	s.registerRoutes(group)

	log.Info().Msg("health provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "health provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down health provider")
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
			// Fault event routes
			{Method: http.MethodPost, Path: prefix + "/health/events/ingest"},
			{Method: http.MethodGet, Path: prefix + "/health/events/summary"},
			{Method: http.MethodGet, Path: prefix + "/health/events"},
			{Method: http.MethodGet, Path: prefix + "/health/events/:id"},
			{Method: http.MethodPatch, Path: prefix + "/health/events/:id"},
			{Method: http.MethodPost, Path: prefix + "/health/events/:id/remediate"},

			// Service event routes
			{Method: http.MethodGet, Path: prefix + "/tenant/:tenantId/service-events"},
			{Method: http.MethodGet, Path: prefix + "/tenant/:tenantId/service-events/active"},
			{Method: http.MethodGet, Path: prefix + "/tenant/:tenantId/service-events/:id"},

			// Classification routes
			{Method: http.MethodGet, Path: prefix + "/health/classifications"},
			{Method: http.MethodPut, Path: prefix + "/health/classifications/:classification"},

			// Webhook routes
			{Method: http.MethodPost, Path: prefix + "/health/webhooks/alertmanager"},

			// Metrics route
			{Method: http.MethodGet, Path: prefix + "/health/metrics"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response. This is the key method that bridges gRPC transport
// to the existing Echo-based handler logic.
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

// GetHookRegistrations returns the hook registrations for the health provider.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			// Async reactions — fault lifecycle
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "health",
				Event:          "post-health-event-ingested",
				TargetWorkflow: "health-fault-remediation",
				SignalName:     "fault-ingested",
			},
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "health",
				Event:          "post-fault-remediation",
				TargetWorkflow: "health-fault-watcher",
				SignalName:     "fault-remediated",
			},
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "health",
				Event:          "post-fault-resolved",
				TargetWorkflow: "health-fault-watcher",
				SignalName:     "fault-resolved",
			},
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "health",
				Event:          "post-fault-escalated",
				TargetWorkflow: "health-fault-watcher",
				SignalName:     "fault-escalated",
			},
			// Sync hook — block instance creation on faulty machines
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "pre-create-instance",
			},
		},
	}, nil
}

// HandleSyncHook dispatches a synchronous hook invocation. The only sync
// hook the health provider registers is "compute/pre-create-instance" which
// blocks instance creation on machines with open critical faults.
func (s *Server) HandleSyncHook(ctx context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	log.Info().
		Str("feature", event.GetFeature()).
		Str("event", event.GetEvent()).
		Msg("handling sync hook")

	if event.GetFeature() == "compute" && event.GetEvent() == "pre-create-instance" {
		var payload map[string]interface{}
		if err := json.Unmarshal(event.GetPayload(), &payload); err != nil {
			return &providerv1.HookResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("failed to parse hook payload: %v", err),
			}, nil
		}

		if err := s.blockInstanceOnFaultyMachine(ctx, payload); err != nil {
			return &providerv1.HookResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}

		return &providerv1.HookResult{Success: true}, nil
	}

	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment. The health provider's
// OpenAPI spec is not yet extracted as a standalone fragment.
func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

// blockInstanceOnFaultyMachine prevents instance creation on machines that
// have open critical faults.
func (s *Server) blockInstanceOnFaultyMachine(ctx context.Context, payload interface{}) error {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return nil
	}
	machineID, ok := data["machine_id"].(string)
	if !ok || machineID == "" {
		return nil
	}

	faults, err := s.faultStore.ListOpenCriticalByMachine(ctx, machineID)
	if err != nil {
		return fmt.Errorf("failed to check faults for machine %s: %w", machineID, err)
	}

	if len(faults) > 0 {
		return fmt.Errorf("machine %s has %d open critical fault(s); instance creation blocked", machineID, len(faults))
	}

	return nil
}
