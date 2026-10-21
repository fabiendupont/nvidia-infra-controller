// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package netris implements the NICo switch-config provider for Netris-managed
// switches. Switches annotated with:
//
//	nico.nvidia.com/switch-config-provider: netris
//
// are configured via the Netris SDN controller API rather than direct NVUE calls.
package netris

import (
	"context"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

const (
	providerName = "nico-switch-config-netris"
	taskQueue    = "nico-switch-config-netris-task-queue"
)

// Server implements providerv1.NicoProviderServer for the Netris switch-config provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer
	configurator *NetrisConfigurator
}

func NewServer() *Server { return &Server{} }

func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         providerName,
		Version:      "0.1.0",
		Features:     []string{"switch-config"},
		Dependencies: []string{},
	}, nil
}

func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	apiURL := os.Getenv("NETRIS_API_URL")
	if apiURL == "" {
		apiURL = req.GetConfig()["netris_api_url"]
	}
	apiToken := os.Getenv("NETRIS_API_TOKEN")
	if apiToken == "" {
		apiToken = req.GetConfig()["netris_api_token"]
	}
	if apiURL == "" {
		log.Warn().Msg("NETRIS_API_URL not configured — push/validate will return errors")
	}
	s.configurator = NewNetrisConfigurator(apiURL, apiToken)
	log.Info().Str("api_url", apiURL).Msg("Netris switch-config provider initialised")
	return &providerv1.InitResponse{Ready: true, Message: "Netris switch-config provider ready"}, nil
}

func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	return &providerv1.ShutdownResponse{}, nil
}

func (s *Server) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{Status: providerv1.HealthCheckResponse_SERVING}, nil
}

func (s *Server) TaskQueue() string { return taskQueue }

func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

func (s *Server) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{Success: false, ErrorMessage: "unknown hook: " + event.GetEvent()}, nil
}

func (s *Server) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	prefix := "/api/v1/switch-config-netris"
	return &providerv1.RouteList{Routes: []*providerv1.Route{
		{Method: http.MethodPost, Path: prefix + "/switches/:switchID/push"},
		{Method: http.MethodGet, Path: prefix + "/switches/:switchID/validate"},
		{Method: http.MethodGet, Path: prefix + "/switches/:switchID/diagnostics"},
	}}, nil
}

func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

func (s *Server) HandleRequest(_ context.Context, _ *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	return &providerv1.HTTPResponse{
		StatusCode: http.StatusNotImplemented,
		Body:       []byte(`{"error":"Netris switch-config HTTP routes not yet implemented"}`),
	}, nil
}

func (s *Server) GetResourceTypes(_ context.Context, _ *providerv1.GetResourceTypesRequest) (*providerv1.GetResourceTypesResponse, error) {
	return &providerv1.GetResourceTypesResponse{
		ResourceTypes: []*providerv1.ResourceTypeDescriptor{{
			Name:      "switch",
			Plural:    "switches",
			ApiPrefix: "/api/v1/switch-config-netris",
			Actions: []*providerv1.ActionDescriptor{
				{Name: "push-config", Label: "Push Config", Method: http.MethodPost, Path: "/switches/:id/push"},
				{Name: "validate", Label: "Validate", Method: http.MethodGet, Path: "/switches/:id/validate"},
				{Name: "get-diagnostics", Label: "Diagnostics", Method: http.MethodGet, Path: "/switches/:id/diagnostics"},
			},
		}},
	}, nil
}
