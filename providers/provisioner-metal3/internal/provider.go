// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metal3

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// Server implements providerv1.NicoProviderServer for the Metal3 provisioner.
// It is purely hook-driven — no HTTP routes are exposed.
type Server struct {
	providerv1.UnimplementedNicoProviderServer
	provisioner   *Metal3Provisioner
	inspector     *IronicInspectorClient          // nil when IRONIC_INSPECTOR_URL is unset
	computeClient providerv1.NicoComputeServiceClient // nil when no compute endpoint given
	// machineToNode maps NICo machine IDs to their BareMetalHost names (= Ironic node UUIDs).
	// Populated when pre-machine-provision creates a BareMetalHost; read by post-machine-inspect.
	machineToNode sync.Map
}

// NewServer creates a new Metal3 provider gRPC server.
func NewServer() *Server {
	return &Server{}
}

// GetInfo returns provider metadata.
func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "nico-metal3-provisioner",
		Version:      "1.0.0",
		Features:     []string{"provisioner"},
		Dependencies: []string{"compute"},
	}, nil
}

// Init connects to the management cluster and creates the Metal3Provisioner.
func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Msg("initializing Metal3 provisioner provider")

	cfg := LoadConfig()
	p, err := NewMetal3Provisioner(cfg.KubeconfigPath, cfg.Metal3Namespace)
	if err != nil {
		msg := "failed to connect to management cluster: " + err.Error()
		log.Error().Err(err).Msg(msg)
		return &providerv1.InitResponse{Ready: false, Message: msg}, nil
	}
	s.provisioner = p

	if cfg.IronicInspectorURL != "" {
		s.inspector = NewIronicInspectorClient(cfg.IronicInspectorURL, cfg.IronicToken)
		log.Info().Str("url", cfg.IronicInspectorURL).Msg("Ironic Inspector client configured")
	}

	// Connect to NicoComputeService for attestation state read-back (Gap 4).
	if endpoint := req.GetServiceEndpoints()["compute"]; endpoint != "" {
		conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("could not connect to NicoComputeService; attestation label fallback disabled")
		} else {
			s.computeClient = providerv1.NewNicoComputeServiceClient(conn)
			log.Info().Str("endpoint", endpoint).Msg("NicoComputeService client configured for attestation label fallback")
		}
	}

	log.Info().
		Str("namespace", cfg.Metal3Namespace).
		Msg("Metal3 provisioner initialized")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "Metal3 provisioner ready",
	}, nil
}

// Shutdown is a no-op — no persistent connections to close beyond the k8s client.
func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("Metal3 provisioner shutting down")
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports serving status.
func (s *Server) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns empty — the Metal3 provider exposes no HTTP routes.
func (s *Server) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	return &providerv1.RouteList{}, nil
}

// HandleRequest is unused (no routes registered).
func (s *Server) HandleRequest(_ context.Context, _ *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	return &providerv1.HTTPResponse{StatusCode: 404}, nil
}

// GetHookRegistrations declares the two sync hooks this provider handles.
func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "pre-machine-provision",
			},
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "post-machine-inspect",
			},
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "post-machine-provision",
			},
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "post-machine-deprovision",
			},
		},
	}, nil
}

// HandleSyncHook dispatches to the appropriate hook handler.
func (s *Server) HandleSyncHook(ctx context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	var rawPayload interface{}
	if b := event.GetPayload(); len(b) > 0 {
		_ = json.Unmarshal(b, &rawPayload)
	}

	var err error
	switch event.GetEvent() {
	case "pre-machine-provision":
		err = s.handlePreMachineProvision(ctx, rawPayload)
	case "post-machine-inspect":
		err = s.handlePostMachineInspect(ctx, rawPayload)
	case "post-machine-provision":
		err = s.handlePostMachineProvision(ctx, rawPayload)
	case "post-machine-deprovision":
		err = s.handlePostMachineDeprovision(ctx, rawPayload)
	default:
		return &providerv1.HookResult{
			Success:      false,
			ErrorMessage: "unknown hook event: " + event.GetEvent(),
		}, nil
	}

	if err != nil {
		return &providerv1.HookResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}
	return &providerv1.HookResult{Success: true}, nil
}

// GetOpenAPIFragment returns empty — no routes, no OpenAPI contribution.
func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}
