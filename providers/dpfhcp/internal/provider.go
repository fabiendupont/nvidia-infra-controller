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

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	tsdkClient "go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// Server implements the NicoProviderServer gRPC interface for the DPF HCP provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer

	store     *ProvisioningStore
	k8sClient *DPFHCPClient
	echo      *echo.Echo

	temporalClient tsdkClient.Client
	temporalWorker tsdkWorker.Worker
}

// NewServer creates a new DPF HCP provider gRPC server.
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
		Name:         "nico-dpfhcp",
		Version:      "0.1.0",
		Features:     []string{"dpf-hcp"},
		Dependencies: []string{"nico-site"},
	}, nil
}

// Init initializes the DPF HCP provider. It sets up the in-memory store, K8s
// client, registers routes on the internal Echo instance, and starts a
// Temporal worker for the provider's workflows.
func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing dpfhcp provider")

	// Initialize the in-memory store.
	s.store = NewProvisioningStore()

	// Initialize K8s client for DPFHCPProvisioner CR management.
	// Try in-cluster config first, fall back to kubeconfig.
	k8sClient, err := NewDPFHCPClient()
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
		k8sClient, err = NewDPFHCPClientFromKubeconfig(kubeconfigPath)
		if err != nil {
			log.Warn().Err(err).Msg("K8s client not available; DPF HCP CR operations will fail")
		}
	}
	s.k8sClient = k8sClient

	// Register routes on the internal Echo instance.
	group := s.echo.Group("")
	registerRoutes(group, s.store, s.temporalClient, taskQueue)

	// Start a Temporal worker if an endpoint was provided.
	if ep := req.GetTemporalEndpoint(); ep != "" {
		ns := req.GetTemporalNamespace()
		if ns == "" {
			ns = "default"
		}

		tc, err := tsdkClient.Dial(tsdkClient.Options{
			HostPort:  ep,
			Namespace: ns,
		})
		if err != nil {
			return &providerv1.InitResponse{
				Ready:   false,
				Message: fmt.Sprintf("failed to connect to Temporal at %s: %v", ep, err),
			}, nil
		}
		s.temporalClient = tc

		w := tsdkWorker.New(tc, taskQueue, tsdkWorker.Options{})
		registerWorkflows(w)
		registerActivities(w, s.store, s.k8sClient)

		if err := w.Start(); err != nil {
			tc.Close()
			s.temporalClient = nil
			return &providerv1.InitResponse{
				Ready:   false,
				Message: fmt.Sprintf("failed to start Temporal worker: %v", err),
			}, nil
		}
		s.temporalWorker = w

		// Re-register routes now that the Temporal client is available.
		s.echo = echo.New()
		s.echo.HideBanner = true
		s.echo.HidePort = true
		group = s.echo.Group("")
		registerRoutes(group, s.store, s.temporalClient, taskQueue)
	}

	log.Info().Msg("dpfhcp provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "dpfhcp provider ready",
	}, nil
}

// Shutdown gracefully stops the provider and the Temporal worker.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down dpfhcp provider")

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
			{Method: http.MethodPost, Path: prefix + "/sites/:siteId/dpf-hcp"},
			{Method: http.MethodGet, Path: prefix + "/sites/:siteId/dpf-hcp"},
			{Method: http.MethodDelete, Path: prefix + "/sites/:siteId/dpf-hcp"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response.
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

// GetHookRegistrations returns the hook registrations for the DPF HCP provider.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			// Async: auto-provision on site creation
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "site",
				Event:          "post-create-site",
				TargetWorkflow: "dpfhcp-site-watcher",
				SignalName:     "site-created",
			},
			// Async: teardown on site deletion
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "site",
				Event:          "post-delete-site-components",
				TargetWorkflow: "dpfhcp-site-watcher",
				SignalName:     "site-deleted",
			},
			// Sync: block instance creation if DPF not ready
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "pre-create-instance",
			},
		},
	}, nil
}

// HandleSyncHook dispatches a synchronous hook invocation. The only sync
// hook the dpfhcp provider registers is "compute/pre-create-instance" which
// blocks instance creation when DPF HCP infrastructure is not ready.
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

		if err := preCreateInstanceCheck(s.store, ctx, payload); err != nil {
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

// GetOpenAPIFragment returns an empty OpenAPI fragment. The DPF HCP provider's
// OpenAPI spec is not yet extracted as a standalone fragment.
func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}
