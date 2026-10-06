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
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tsdkClient "go.temporal.io/sdk/client"
	tsdkConverter "go.temporal.io/sdk/converter"
	tsdkWorker "go.temporal.io/sdk/worker"
	zlogadapter "logur.dev/adapter/zerolog"
	"logur.dev/logur"

	apiSiteClient "github.com/NVIDIA/infra-controller/rest-api/api/pkg/client/site"
	cconfig "github.com/NVIDIA/infra-controller/rest-api/common/pkg/config"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/providers/site"
	wfSiteClient "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/client/site"
)

// siteProviderServer implements providerv1.NicoProviderServer by delegating
// HTTP handling to the existing site provider via an internal Echo instance
// and running a Temporal worker for site workflows.
type siteProviderServer struct {
	providerv1.UnimplementedNicoProviderServer

	provider  *site.SiteProvider
	echo      *echo.Echo
	dbSession *cdb.Session
	tc        tsdkClient.Client
	worker    tsdkWorker.Worker
}

// NewSiteProviderServer creates a new gRPC server backed by the site provider.
func NewSiteProviderServer() *siteProviderServer {
	s := &siteProviderServer{
		provider: site.New(),
		echo:     echo.New(),
	}

	// Silence Echo's default logger — we use zerolog.
	s.echo.HideBanner = true
	s.echo.HidePort = true

	return s
}

// GetInfo returns the provider's metadata.
func (s *siteProviderServer) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         s.provider.Name(),
		Version:      s.provider.Version(),
		Features:     s.provider.Features(),
		Dependencies: s.provider.Dependencies(),
	}, nil
}

// Init initializes the site provider. It establishes a database connection,
// creates Temporal clients and site client pools, initializes the provider,
// registers routes, and starts a Temporal worker for site workflows.
func (s *siteProviderServer) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing site provider")

	// Connect to the database from environment variables.
	dbCfg, err := cdb.ConfigFromEnv()
	if err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("database configuration error: %v", err),
		}, nil
	}

	dbSession, err := cdb.NewSessionFromConfig(context.Background(), dbCfg)
	if err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("database connection failed: %v", err),
		}, nil
	}
	s.dbSession = dbSession

	// Determine Temporal endpoint.
	temporalEndpoint := req.GetTemporalEndpoint()
	if temporalEndpoint == "" {
		temporalEndpoint = envOrDefault("TEMPORAL_HOST", "localhost") + ":" + envOrDefault("TEMPORAL_PORT", "7233")
	}

	temporalNamespace := req.GetTemporalNamespace()
	if temporalNamespace == "" {
		temporalNamespace = envOrDefault("TEMPORAL_NAMESPACE", "cloud")
	}

	taskQueue := envOrDefault("TEMPORAL_QUEUE", "nico-site-queue")

	// Create Temporal client.
	tLogger := logur.LoggerToKV(zlogadapter.New(zerolog.New(os.Stderr)))
	tc, err := tsdkClient.Dial(tsdkClient.Options{
		HostPort:  temporalEndpoint,
		Namespace: temporalNamespace,
		DataConverter: tsdkConverter.NewCompositeDataConverter(
			tsdkConverter.NewNilPayloadConverter(),
			tsdkConverter.NewByteSlicePayloadConverter(),
			tsdkConverter.NewProtoJSONPayloadConverterWithOptions(tsdkConverter.ProtoJSONPayloadConverterOptions{
				AllowUnknownFields: true,
			}),
			tsdkConverter.NewProtoPayloadConverter(),
			tsdkConverter.NewJSONPayloadConverter(),
		),
		Logger: tLogger,
	})
	if err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("temporal connection failed: %v", err),
		}, nil
	}
	s.tc = tc

	// Create Temporal namespace client.
	tnc, err := tsdkClient.NewNamespaceClient(tsdkClient.Options{
		HostPort: temporalEndpoint,
		Logger:   tLogger,
	})
	if err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("temporal namespace client failed: %v", err),
		}, nil
	}

	// Build site client pools for API and workflow layers. Both use the
	// same Temporal config pointing at the Temporal server.
	temporalHost, temporalPortStr := parseHostPort(temporalEndpoint)
	temporalPort, _ := strconv.Atoi(temporalPortStr)

	tcfg := &cconfig.TemporalConfig{
		Host: temporalHost,
		Port: temporalPort,
	}

	apiSCP := apiSiteClient.NewClientPool(tcfg)
	wfSCP := wfSiteClient.NewClientPool(tcfg)

	apiPathPrefix := envOrDefault("API_PATH_PREFIX", "/api/v1")

	// Initialize the site provider with standalone dependencies.
	if err := s.provider.InitStandalone(dbSession, tc, tnc, apiSCP, wfSCP, apiPathPrefix, temporalNamespace, taskQueue); err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("provider initialization failed: %v", err),
		}, nil
	}

	// Register routes on the internal Echo instance.
	group := s.echo.Group("")
	s.provider.RegisterRoutes(group)

	// Start a Temporal worker for site workflows.
	w := tsdkWorker.New(tc, taskQueue, tsdkWorker.Options{})
	s.provider.RegisterWorkflows(w)

	if err := w.Start(); err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("temporal worker start failed: %v", err),
		}, nil
	}
	s.worker = w

	log.Info().
		Str("task_queue", taskQueue).
		Str("namespace", temporalNamespace).
		Msg("site provider initialized with temporal worker")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "site provider ready",
	}, nil
}

// Shutdown gracefully stops the provider and its Temporal worker.
func (s *siteProviderServer) Shutdown(ctx context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down site provider")

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
func (s *siteProviderServer) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles.
func (s *siteProviderServer) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1"

	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			// Site endpoints
			{Method: http.MethodPost, Path: prefix + "/site"},
			{Method: http.MethodGet, Path: prefix + "/site"},
			{Method: http.MethodGet, Path: prefix + "/site/:id"},
			{Method: http.MethodPatch, Path: prefix + "/site/:id"},
			{Method: http.MethodDelete, Path: prefix + "/site/:id"},
			{Method: http.MethodGet, Path: prefix + "/site/:id/status-history"},

			// ExpectedMachine endpoints
			{Method: http.MethodPost, Path: prefix + "/expected-machine"},
			{Method: http.MethodGet, Path: prefix + "/expected-machine"},
			{Method: http.MethodGet, Path: prefix + "/expected-machine/:id"},
			{Method: http.MethodPatch, Path: prefix + "/expected-machine/:id"},
			{Method: http.MethodDelete, Path: prefix + "/expected-machine/:id"},

			// ExpectedPowerShelf endpoints
			{Method: http.MethodPost, Path: prefix + "/expected-power-shelf"},
			{Method: http.MethodGet, Path: prefix + "/expected-power-shelf"},
			{Method: http.MethodGet, Path: prefix + "/expected-power-shelf/:id"},
			{Method: http.MethodPatch, Path: prefix + "/expected-power-shelf/:id"},
			{Method: http.MethodDelete, Path: prefix + "/expected-power-shelf/:id"},

			// ExpectedSwitch endpoints
			{Method: http.MethodPost, Path: prefix + "/expected-switch"},
			{Method: http.MethodGet, Path: prefix + "/expected-switch"},
			{Method: http.MethodGet, Path: prefix + "/expected-switch/:id"},
			{Method: http.MethodPatch, Path: prefix + "/expected-switch/:id"},
			{Method: http.MethodDelete, Path: prefix + "/expected-switch/:id"},
		},
	}, nil
}

// HandleRequest dispatches an HTTP request to the internal Echo router and
// returns the response.
func (s *siteProviderServer) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
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

// GetHookRegistrations returns an empty list. The site provider does not
// register any hooks in its built-in implementation. Hook firing is done
// internally by the ManageSite activity when a HookRunner is available.
func (s *siteProviderServer) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook returns an error because the site provider does not register
// any synchronous hooks.
func (s *siteProviderServer) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *siteProviderServer) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

// envOrDefault returns the environment variable value or a default.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseHostPort splits a host:port string. If no port is present, it returns
// the input as host and "7233" as the default Temporal port.
func parseHostPort(endpoint string) (string, string) {
	if idx := strings.LastIndex(endpoint, ":"); idx >= 0 {
		return endpoint[:idx], endpoint[idx+1:]
	}
	return endpoint, "7233"
}
