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
	"github.com/NVIDIA/infra-controller/providers/catalog/internal"
)

// catalogProviderServer implements providerv1.NicoProviderServer by
// delegating HTTP handling to the existing catalog provider via an
// internal Echo instance.
type catalogProviderServer struct {
	providerv1.UnimplementedNicoProviderServer

	provider *catalog.CatalogProvider
	echo     *echo.Echo
}

// NewCatalogProviderServer creates a new gRPC server backed by the
// catalog provider. The provider is initialized lazily via Init().
func NewCatalogProviderServer() *catalogProviderServer {
	s := &catalogProviderServer{
		provider: catalog.New(),
		echo:     echo.New(),
	}

	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *catalogProviderServer) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         s.provider.Name(),
		Version:      s.provider.Version(),
		Features:     s.provider.Features(),
		Dependencies: s.provider.Dependencies(),
	}, nil
}

// Init initializes the catalog provider.
func (s *catalogProviderServer) Init(_ context.Context, _ *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().Msg("initializing catalog provider")

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

	log.Info().Msg("catalog provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "catalog provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (s *catalogProviderServer) Shutdown(ctx context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down catalog provider")
	_ = s.provider.Shutdown(ctx)
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (s *catalogProviderServer) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles.
func (s *catalogProviderServer) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// Blueprint endpoints
			{Method: http.MethodGet, Path: prefix + "/catalog/blueprints"},
			{Method: http.MethodGet, Path: prefix + "/catalog/blueprints/:id"},
			{Method: http.MethodGet, Path: prefix + "/catalog/blueprints/:id/resolved"},
			{Method: http.MethodPost, Path: prefix + "/catalog/blueprints/:id/estimate"},
			{Method: http.MethodPost, Path: prefix + "/catalog/blueprints/:id/validate"},
			{Method: http.MethodPost, Path: prefix + "/catalog/blueprints"},
			{Method: http.MethodPatch, Path: prefix + "/catalog/blueprints/:id"},
			{Method: http.MethodDelete, Path: prefix + "/catalog/blueprints/:id"},

			// RBAC generation
			{Method: http.MethodPost, Path: prefix + "/catalog/blueprints/:id/generate-role"},

			// Resource types
			{Method: http.MethodGet, Path: prefix + "/catalog/resource-types"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router.
func (s *catalogProviderServer) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns an empty list. The catalog provider does
// not register any hooks.
func (s *catalogProviderServer) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook is a no-op — the catalog provider has no sync hooks.
func (s *catalogProviderServer) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *catalogProviderServer) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
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
