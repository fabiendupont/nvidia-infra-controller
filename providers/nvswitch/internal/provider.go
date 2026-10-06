// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// Server implements the NicoProviderServer gRPC interface for the NVSwitch provider.
type Server struct {
	providerv1.UnimplementedNicoProviderServer
}

// NewServer creates a new NVSwitch provider gRPC server.
func NewServer() *Server {
	return &Server{}
}

// GetInfo returns the provider's metadata.
func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "nico-nvswitch",
		Version:      "1.0.6",
		Features:     []string{"nvswitch"},
		Dependencies: []string{"nico-networking"},
	}, nil
}

// Init initializes the provider with core runtime context.
func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Str("temporal_namespace", req.GetTemporalNamespace()).
		Msg("initializing nvswitch provider")

	log.Info().Msg("nvswitch provider initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "nvswitch provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down nvswitch provider")
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (s *Server) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns the HTTP routes this provider handles. The NVSwitch
// provider does not expose any routes.
func (s *Server) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	return &providerv1.RouteList{}, nil
}

// HandleRequest is a no-op since the NVSwitch provider has no routes.
func (s *Server) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	return &providerv1.HTTPResponse{
		StatusCode: 404,
		Body:       []byte(fmt.Sprintf(`{"error":"nvswitch provider has no route for %s %s"}`, req.GetMethod(), req.GetPath())),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

// GetHookRegistrations returns an empty list since the NVSwitch provider
// does not register any hooks.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{}, nil
}

// HandleSyncHook is a no-op since the NVSwitch provider has no sync hooks.
func (s *Server) HandleSyncHook(_ context.Context, _ *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: "nvswitch provider has no sync hooks",
	}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}
