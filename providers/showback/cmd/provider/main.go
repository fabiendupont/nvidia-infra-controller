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

	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"github.com/NVIDIA/infra-controller/providers/showback/internal"
)

// showbackProviderServer implements providerv1.NicoProviderServer by
// delegating HTTP handling to the existing showback provider via an
// internal Echo instance.
type showbackProviderServer struct {
	providerv1.UnimplementedNicoProviderServer

	provider *showback.ShowbackProvider
	echo     *echo.Echo
}

// NewShowbackProviderServer creates a new gRPC server backed by the
// showback provider. The provider is initialized lazily via Init().
func NewShowbackProviderServer() *showbackProviderServer {
	s := &showbackProviderServer{
		provider: showback.New(),
		echo:     echo.New(),
	}

	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *showbackProviderServer) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         s.provider.Name(),
		Version:      s.provider.Version(),
		Features:     s.provider.Features(),
		Dependencies: s.provider.Dependencies(),
	}, nil
}

// Init initializes the showback provider.
func (s *showbackProviderServer) Init(_ context.Context, _ *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().Msg("initializing showback provider")

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

	log.Info().Msg("showback provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "showback provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (s *showbackProviderServer) Shutdown(ctx context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down showback provider")
	_ = s.provider.Shutdown(ctx)
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (s *showbackProviderServer) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles.
func (s *showbackProviderServer) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// Service usage
			{Method: http.MethodGet, Path: prefix + "/services/:id/usage"},

			// Tenant self-service usage and quotas
			{Method: http.MethodGet, Path: prefix + "/self/usage"},
			{Method: http.MethodGet, Path: prefix + "/self/usage/costs"},
			{Method: http.MethodGet, Path: prefix + "/self/quotas"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router.
func (s *showbackProviderServer) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns the async reactions for instance lifecycle
// events that drive metering.
func (s *showbackProviderServer) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			{
				Type:    providerv1.HookRegistration_ASYNC,
				Feature: "compute",
				Event:   "post-create-instance",
			},
			{
				Type:    providerv1.HookRegistration_ASYNC,
				Feature: "compute",
				Event:   "post-delete-instance",
			},
		},
	}, nil
}

// HandleSyncHook is a no-op — the showback provider has no sync hooks.
func (s *showbackProviderServer) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *showbackProviderServer) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
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
