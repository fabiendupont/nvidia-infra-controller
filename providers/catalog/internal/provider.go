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

package catalog

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
	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// Server implements the NicoProviderServer gRPC interface for the catalog provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer

	store   BlueprintStoreInterface
	handler *BlueprintHandler
	echo    *echo.Echo
}

// NewServer creates a new catalog provider gRPC server.
func NewServer() *Server {
	return &Server{}
}

// GetInfo returns the provider's metadata.
func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "nico-catalog",
		Version:      "0.1.0",
		Features:     []string{"catalog"},
		Dependencies: []string{},
	}, nil
}

// Init initializes the catalog provider, connecting to PostgreSQL in the
// "catalog" schema and running migrations before serving requests.
func (s *Server) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing catalog provider")

	port, _ := strconv.Atoi(envOrDefault("DB_PORT", "5432"))
	dbCfg := sdk.Config{
		Host:       envOrDefault("DB_HOST", "localhost"),
		Port:       port,
		DBName:     envOrDefault("DB_NAME", "nico"),
		Credential: sdk.NewCredential(envOrDefault("DB_USER", "nico"), envOrDefault("DB_PASSWORD", "")),
	}

	db, err := sdk.ConnectWithSchema(ctx, dbCfg, "catalog")
	if err != nil {
		return initFailed("database connection failed: %v", err), nil
	}

	migrator := sdk.NewMigrator(db.DB, "catalog", catalogMigrations)
	if err := migrator.Run(ctx); err != nil {
		return initFailed("migrations failed: %v", err), nil
	}

	s.store = NewBlueprintSQLStore(db)
	s.handler = NewBlueprintHandler(s.store)

	// Set up an internal Echo instance for HTTP dispatch.
	s.echo = echo.New()
	s.echo.HideBanner = true
	s.echo.HidePort = true

	// Register blueprint routes.
	prefix := "/api/v1"
	bp := prefix + "/catalog/blueprints"

	// Read endpoints
	s.echo.GET(bp, s.handler.handleListBlueprints)
	s.echo.GET(bp+"/:id", s.handler.handleGetBlueprint)
	s.echo.GET(bp+"/:id/resolved", s.handler.handleResolvedBlueprint)
	s.echo.POST(bp+"/:id/estimate", s.handler.handleEstimateCost)
	s.echo.POST(bp+"/:id/validate", s.handler.handleValidateBlueprint)
	s.echo.GET(prefix+"/catalog/resource-types", s.handler.handleListResourceTypes)

	// RBAC generation
	s.echo.POST(bp+"/:id/generate-role", s.handler.handleGenerateRole)

	// Write endpoints
	s.echo.POST(bp, s.handler.handleCreateBlueprint)
	s.echo.PATCH(bp+"/:id", s.handler.handleUpdateBlueprint)

	// Delete
	s.echo.DELETE(bp+"/:id", s.handler.handleDeleteBlueprint)

	// Load seed data.
	loadSeedData(s.store)

	log.Info().Msg("catalog provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "catalog provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down catalog provider")
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
	bp := prefix + "/catalog/blueprints"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// Read endpoints
			{Method: http.MethodGet, Path: bp},
			{Method: http.MethodGet, Path: bp + "/:id"},
			{Method: http.MethodGet, Path: bp + "/:id/resolved"},
			{Method: http.MethodPost, Path: bp + "/:id/estimate"},
			{Method: http.MethodPost, Path: bp + "/:id/validate"},
			{Method: http.MethodGet, Path: prefix + "/catalog/resource-types"},

			// RBAC generation
			{Method: http.MethodPost, Path: bp + "/:id/generate-role"},

			// Write endpoints
			{Method: http.MethodPost, Path: bp},
			{Method: http.MethodPatch, Path: bp + "/:id"},

			// Delete
			{Method: http.MethodDelete, Path: bp + "/:id"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response. This bridges gRPC transport to the Echo-based handler logic.
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

// GetHookRegistrations returns an empty list. The catalog provider does not
// register any hooks.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook is a no-op for the catalog provider -- it has no sync hooks.
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

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initFailed(format string, args ...interface{}) *providerv1.InitResponse {
	msg := fmt.Sprintf(format, args...)
	log.Error().Msg(msg)
	return &providerv1.InitResponse{Ready: false, Message: msg}
}
