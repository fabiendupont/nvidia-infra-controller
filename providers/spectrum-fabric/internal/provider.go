// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package spectrumfabric

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"github.com/fabiendupont/nvidia-nvue-client-go/pkg/nvue"
)

// Server implements NicoProviderServer for the spectrum-fabric provider.
// It manages Spectrum-X switch fabric directly via the NVUE REST API,
// reacting to NICo VPC and Subnet lifecycle events.
type Server struct {
	providerv1.UnimplementedNicoProviderServer
	config ProviderConfig
	client *nvue.Client
	syncMu sync.Mutex
}

func NewServer() *Server {
	return &Server{config: ConfigFromEnv()}
}

func (s *Server) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "spectrum-fabric",
		Version:      "0.1.0",
		Features:     []string{"spectrum-fabric"},
		Dependencies: []string{"nico-networking"},
	}, nil
}

func (s *Server) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	log.Info().
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Msg("initializing spectrum-fabric provider")

	if err := s.config.Validate(); err != nil {
		return initFailed("invalid config: %v", err), nil
	}

	opts := []nvue.Option{}
	if s.config.TLSSkipVerify {
		opts = append(opts, nvue.WithInsecureSkipVerify())
	}
	if s.config.RevisionPollInterval > 0 {
		opts = append(opts, nvue.WithPollInterval(s.config.RevisionPollInterval))
	}
	if s.config.RevisionTimeout > 0 {
		opts = append(opts, nvue.WithApplyTimeout(s.config.RevisionTimeout))
	}

	s.client = nvue.NewClientFromURL(s.config.NVUEURL, s.config.NVUEUsername, s.config.NVUEPassword, opts...)

	// Verify connectivity by fetching the system configuration.
	if _, err := s.client.Get(context.Background(), "/system", ""); err != nil {
		return initFailed("NVUE connectivity check failed: %v", err), nil
	}

	log.Info().
		Bool("sync_vpc", s.config.Features.SyncVPC).
		Bool("sync_subnet", s.config.Features.SyncSubnet).
		Msg("spectrum-fabric provider initialized")

	return &providerv1.InitResponse{Ready: true, Message: "spectrum-fabric provider ready"}, nil
}

func (s *Server) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Msg("shutting down spectrum-fabric provider")
	return &providerv1.ShutdownResponse{}, nil
}

func (s *Server) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{Status: providerv1.HealthCheckResponse_SERVING}, nil
}

// GetRoutes returns no HTTP routes — this provider is hook-driven only.
func (s *Server) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	return &providerv1.RouteList{}, nil
}

// HandleRequest is not used since GetRoutes returns empty.
func (s *Server) HandleRequest(_ context.Context, _ *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	return &providerv1.HTTPResponse{StatusCode: 404}, nil
}

func (s *Server) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	regs := []*providerv1.HookRegistration{}

	if s.config.Features.SyncVPC {
		regs = append(regs,
			&providerv1.HookRegistration{
				Type: providerv1.HookRegistration_ASYNC, Feature: "networking", Event: "post-create-vpc",
			},
			&providerv1.HookRegistration{
				Type: providerv1.HookRegistration_ASYNC, Feature: "networking", Event: "post-delete-vpc",
			},
		)
	}

	if s.config.Features.SyncSubnet {
		regs = append(regs,
			&providerv1.HookRegistration{
				Type: providerv1.HookRegistration_ASYNC, Feature: "networking", Event: "post-create-subnet",
			},
			&providerv1.HookRegistration{
				Type: providerv1.HookRegistration_ASYNC, Feature: "networking", Event: "post-delete-subnet",
			},
			&providerv1.HookRegistration{
				Type: providerv1.HookRegistration_SYNC, Feature: "networking", Event: "pre-create-subnet",
			},
		)
	}

	return &providerv1.HookRegistrationList{Registrations: regs}, nil
}

func (s *Server) HandleSyncHook(_ context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	if event.GetFeature() == "networking" && event.GetEvent() == "pre-create-subnet" {
		if err := s.validateSubnetConfig(context.Background(), event.GetPayload()); err != nil {
			return &providerv1.HookResult{Success: false, ErrorMessage: err.Error()}, nil
		}
		return &providerv1.HookResult{Success: true}, nil
	}
	return &providerv1.HookResult{
		Success:      false,
		ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
	}, nil
}

func (s *Server) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

// validateSubnetConfig performs a dry-run NVUE configuration check before NICo commits the subnet.
func (s *Server) validateSubnetConfig(ctx context.Context, payload interface{}) error {
	payloadMap, ok := payload.(map[string]string)
	if !ok {
		return nil
	}

	prefix := payloadMap["prefix"]
	if prefix == "" {
		prefix = payloadMap["cidr"]
	}
	vpcID := payloadMap["vpc_id"]

	if prefix == "" || vpcID == "" {
		return nil
	}

	revID, err := s.client.CreateRevisionID(ctx)
	if err != nil {
		return fmt.Errorf("NVUE dry-run: creating revision: %w", err)
	}

	vrfName := fmt.Sprintf("nico-%s", vpcID)
	sviConfig := map[string]any{
		"type": "svi",
		"ip": map[string]any{
			"address": map[string]any{prefix: map[string]any{}},
			"vrf":     vrfName,
		},
	}

	if err := s.client.Patch(ctx, "/interface/validation-probe", revID, sviConfig); err != nil {
		_ = s.client.Delete(ctx, fmt.Sprintf("/revision/%s", revID), "")
		return fmt.Errorf("NVUE dry-run: subnet config rejected by Spectrum fabric: %w", err)
	}

	_ = s.client.Delete(ctx, fmt.Sprintf("/revision/%s", revID), "")
	return nil
}

func initFailed(format string, args ...interface{}) *providerv1.InitResponse {
	msg := fmt.Sprintf(format, args...)
	log.Error().Msg(msg)
	return &providerv1.InitResponse{Ready: false, Message: msg}
}
