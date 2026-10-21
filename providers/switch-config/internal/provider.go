// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	tsdkClient "go.temporal.io/sdk/client"
	tsdkWorker "go.temporal.io/sdk/worker"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	sdkdb "github.com/NVIDIA/infra-controller/provider-sdk/db"
	"github.com/NVIDIA/infra-controller/providers/switch-config/internal/migrations"
)

const (
	providerName      = "nico-switch-config"
	providerVersion   = "0.1.0"
	taskQueue         = "nico-switch-config-task-queue"
)

// Server implements providerv1.NicoProviderServer for the switch-config provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer

	echo         *echo.Echo
	store        *ConfigStore
	switchClient *SwitchClient
	configurator SwitchConfigurator
	activities   *SwitchConfigActivities
	tc           tsdkClient.Client
	worker       tsdkWorker.Worker
	siteIDs      []string
}

// NewServer returns an uninitialised switch-config provider server.
func NewServer() *Server {
	s := &Server{echo: echo.New()}
	s.echo.HideBanner = true
	s.echo.HidePort = true
	return s
}

func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         providerName,
		Version:      providerVersion,
		Features:     []string{"switch-config"},
		Dependencies: []string{"nico-networking"},
	}, nil
}

func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	// Parse site IDs from config.
	if siteIDs := req.GetConfig()["site_ids"]; siteIDs != "" {
		for _, id := range strings.Split(siteIDs, ",") {
			if id = strings.TrimSpace(id); id != "" {
				s.siteIDs = append(s.siteIDs, id)
			}
		}
	}

	// Connect to NicoSwitchService.
	switchEndpoint := req.GetServiceEndpoints()["switch"]
	sc, err := NewSwitchClient(switchEndpoint)
	if err != nil {
		return initFailed("switch service client: %v", err), nil
	}
	s.switchClient = sc

	// Connect to DB and run migrations.
	dbCfg := sdkdb.Config{
		Host:       envOrDefault("DB_HOST", "localhost"),
		Port:       envOrDefaultInt("DB_PORT", 5432),
		DBName:     envOrDefault("DB_NAME", "nico"),
		Credential: sdkdb.NewCredential(envOrDefault("DB_USER", "nico"), envOrDefault("DB_PASSWORD", "")),
	}
	db, err := sdkdb.ConnectWithSchema(context.Background(), dbCfg, "switch_config")
	if err != nil {
		log.Warn().Err(err).Msg("switch-config: DB unavailable, operating without persistence")
	} else {
		m := sdkdb.NewMigrator(db.DB, "switch_config", migrations.All())
		if err := m.Run(context.Background()); err != nil {
			return initFailed("migrations: %v", err), nil
		}
		s.store = NewConfigStore(db.DB)
	}
	// Build the switch configurator (factory: spectrum/nvos only for now).
	platform := envOrDefault("SWITCH_PLATFORM", "spectrum")
	configurator, err := NewSwitchConfigurator(platform, ConfiguratorOptions{
		SwitchService:  s.switchClient,
		ConfigStore:    s.store,
		InsecureTLS:    envOrDefault("SWITCH_INSECURE_TLS", "") == "true",
		SwitchUsername: envOrDefault("SWITCH_USERNAME", ""),
		SwitchPassword: envOrDefault("SWITCH_PASSWORD", ""),
	})
	if err != nil {
		return initFailed("switch configurator (%s): %v", platform, err), nil
	}
	s.configurator = configurator
	s.activities = NewSwitchConfigActivities(s.switchClient, s.store, configurator)

	// Build Temporal client and start worker for async workflow dispatch.
	temporalEndpoint := req.GetTemporalEndpoint()
	if temporalEndpoint == "" {
		temporalEndpoint = envOrDefault("TEMPORAL_ENDPOINT", "localhost:7233")
	}
	temporalNamespace := req.GetTemporalNamespace()
	if temporalNamespace == "" {
		temporalNamespace = envOrDefault("TEMPORAL_NAMESPACE", "default")
	}
	tc, err := tsdkClient.Dial(tsdkClient.Options{
		HostPort:  temporalEndpoint,
		Namespace: temporalNamespace,
	})
	if err != nil {
		log.Warn().Err(err).Msg("switch-config: Temporal unavailable, async routes disabled")
	} else {
		s.tc = tc
		s.worker = tsdkWorker.New(tc, taskQueue, tsdkWorker.Options{})
		s.RegisterWorkflows(s.worker)
		s.RegisterActivities(s.worker)
		go func() {
			if err := s.worker.Run(tsdkWorker.InterruptCh()); err != nil {
				log.Error().Err(err).Msg("switch-config Temporal worker stopped")
			}
		}()
	}

	// Register routes.
	registerRoutes(s.echo, s)

	log.Info().Strs("site_ids", s.siteIDs).Msg("switch-config provider initialised")
	return &providerv1.InitResponse{Ready: true, Message: "switch-config provider ready"}, nil
}

func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	if s.worker != nil {
		s.worker.Stop()
	}
	return &providerv1.ShutdownResponse{}, nil
}

func (s *Server) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{Status: providerv1.HealthCheckResponse_SERVING}, nil
}

func (s *Server) TaskQueue() string { return taskQueue }

func (s *Server) RegisterWorkflows(w tsdkWorker.Worker) {
	w.RegisterWorkflow(CableValidationWorkflow)
	w.RegisterWorkflow(SwitchConfigDeployWorkflow)
	w.RegisterWorkflow(OSUpgradeWorkflow)
}

func (s *Server) RegisterActivities(w tsdkWorker.Worker) {
	w.RegisterActivity(s.activities)
}

func (s *Server) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1/switch-config"
	return &providerv1.RouteList{Routes: []*providerv1.Route{
		{Method: http.MethodGet, Path: prefix + "/configs/:switchID"},
		{Method: http.MethodGet, Path: prefix + "/configs/:switchID/nvue"},
		{Method: http.MethodPost, Path: prefix + "/configs/:switchID/render"},
		{Method: http.MethodGet, Path: prefix + "/validation/:siteID"},
		{Method: http.MethodPost, Path: prefix + "/actions/validate-cables"},
		{Method: http.MethodPost, Path: prefix + "/site/:siteID/deploy"},
		{Method: http.MethodPost, Path: prefix + "/switches/:switchID/upgrade"},
		{Method: http.MethodGet, Path: prefix + "/switches/:switchID/health"},
		{Method: http.MethodGet, Path: prefix + "/switches/:switchID/diagnostics"},
		{Method: http.MethodGet, Path: prefix + "/workflow/:workflowID/status"},
	}}, nil
}

func (s *Server) GetResourceTypes(_ context.Context, _ *providerv1.GetResourceTypesRequest) (*providerv1.GetResourceTypesResponse, error) {
	prefix := "/api/v1/switch-config"
	return &providerv1.GetResourceTypesResponse{
		ResourceTypes: []*providerv1.ResourceTypeDescriptor{
			{
				Name:      "switch",
				Plural:    "switches",
				ApiPrefix: prefix,
				Actions: []*providerv1.ActionDescriptor{
					{Name: "deploy-site", Label: "Deploy Site Config", Method: "POST", Path: "/site/{id}/deploy", RequiresConfirmation: true},
					{Name: "push-config", Label: "Push Config", Method: "POST", Path: "/configs/{id}/render", RequiresConfirmation: true},
					{Name: "validate-cables", Label: "Validate Cables", Method: "POST", Path: "/actions/validate-cables", RequiresConfirmation: false},
					{Name: "upgrade-os", Label: "Upgrade OS", Method: "POST", Path: "/switches/{id}/upgrade", RequiresConfirmation: true},
					{Name: "hardware-health", Label: "Hardware Health", Method: "GET", Path: "/switches/{id}/health", RequiresConfirmation: false},
					{Name: "diagnostics", Label: "Diagnostics", Method: "GET", Path: "/switches/{id}/diagnostics", RequiresConfirmation: false},
					{Name: "workflow-status", Label: "Workflow Status", Method: "GET", Path: "/workflow/{id}/status", RequiresConfirmation: false},
				},
			},
		},
	}, nil
}

func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			{
				Type:    providerv1.HookRegistration_ASYNC,
				Feature: "switch-config",
				Event:   "post-switch-config-rendered",
			},
		},
	}, nil
}

func (s *Server) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	var bodyReader io.Reader
	if len(req.GetBody()) > 0 {
		bodyReader = strings.NewReader(string(req.GetBody()))
	}
	path := req.GetPath()
	if len(req.GetQueryParams()) > 0 {
		parts := make([]string, 0, len(req.GetQueryParams()))
		for k, v := range req.GetQueryParams() {
			parts = append(parts, k+"="+v)
		}
		path += "?" + strings.Join(parts, "&")
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
	body, _ := io.ReadAll(result.Body)
	headers := make(map[string]string, len(result.Header))
	for k := range result.Header {
		headers[k] = result.Header.Get(k)
	}
	return &providerv1.HTTPResponse{
		StatusCode: int32(result.StatusCode),
		Headers:    headers,
		Body:       body,
	}, nil
}

func (s *Server) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

func initFailed(format string, args ...interface{}) *providerv1.InitResponse {
	msg := fmt.Sprintf(format, args...)
	log.Error().Msg(msg)
	return &providerv1.InitResponse{Ready: false, Message: msg}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}

// jsonResponse writes a JSON response to the Echo context.
func jsonResponse(c echo.Context, code int, body interface{}) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(code)
	return json.NewEncoder(c.Response()).Encode(body)
}
