// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ufmfabric

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/fabiendupont/nvidia-ufm-api/pkg/ufmclient"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// UFMFabricProvider manages InfiniBand partition (PKEY) lifecycle on
// UFM Enterprise directly via its REST API. It reacts to NICo IB partition
// events and translates them into UFM PKEY operations.
type UFMFabricProvider struct {
	providerv1.UnimplementedNicoProviderServer

	config ProviderConfig
	client *ufmclient.Client
	syncMu sync.Mutex
}

// New creates a new UFMFabricProvider with explicit configuration.
func New(cfg ProviderConfig) *UFMFabricProvider {
	return &UFMFabricProvider{config: cfg}
}

// NewFromEnv creates a new UFMFabricProvider reading configuration
// from environment variables.
func NewFromEnv() *UFMFabricProvider {
	return New(ConfigFromEnv())
}

// GetInfo returns the provider's metadata.
func (p *UFMFabricProvider) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:         "ufm-fabric",
		Version:      "0.1.0",
		Features:     []string{"ufm-fabric", "ib-fabric"},
		Dependencies: []string{"nico-networking"},
	}, nil
}

func (p *UFMFabricProvider) Name() string { return "ufm-fabric" }

// Init initializes the provider: validates config and establishes UFM connectivity.
func (p *UFMFabricProvider) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	logger := log.With().Str("provider", p.Name()).Logger()

	logger.Info().
		Str("url", p.config.UFMURL).
		Str("temporal_endpoint", req.GetTemporalEndpoint()).
		Msg("initializing ufm-fabric provider")

	if err := p.config.Validate(); err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("configuration invalid: %v", err),
		}, nil
	}

	opts := []ufmclient.Option{}
	if p.config.TLSSkipVerify {
		opts = append(opts, ufmclient.WithTLSSkipVerify())
	}
	if p.config.JobTimeout > 0 {
		opts = append(opts, ufmclient.WithTimeout(p.config.JobTimeout))
	}

	p.client = ufmclient.New(p.config.UFMURL, p.config.UFMUsername, p.config.UFMPassword, opts...)

	version, err := p.client.GetVersion(context.Background())
	if err != nil {
		return &providerv1.InitResponse{
			Ready:   false,
			Message: fmt.Sprintf("UFM connectivity check failed: %v", err),
		}, nil
	}

	logger.Info().
		Str("ufm_version", version.UFMReleaseVersion).
		Msg("UFM connectivity verified")

	return &providerv1.InitResponse{
		Ready:   true,
		Message: "ufm-fabric provider ready",
	}, nil
}

// Shutdown gracefully stops the provider.
func (p *UFMFabricProvider) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Info().Str("provider", p.Name()).Msg("shutting down ufm-fabric provider")
	return &providerv1.ShutdownResponse{}, nil
}

// HealthCheck reports the provider's serving status.
func (p *UFMFabricProvider) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

// GetRoutes returns no routes — the UFM fabric provider operates via hooks only.
func (p *UFMFabricProvider) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	return &providerv1.RouteList{}, nil
}

// HandleRequest is a no-op since the UFM fabric provider has no routes.
func (p *UFMFabricProvider) HandleRequest(_ context.Context, req *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	return &providerv1.HTTPResponse{
		StatusCode: 404,
		Body:       []byte(`{"error":"ufm-fabric provider has no routes"}`),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

// GetHookRegistrations returns the hook registrations for IB partition events.
func (p *UFMFabricProvider) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	if !p.config.Features.SyncIBPartition {
		return &providerv1.HookRegistrationList{}, nil
	}

	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			// Async: when NICo creates an IB partition, create the PKEY on UFM.
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "networking",
				Event:          "post-create-ib-partition",
				TargetWorkflow: "ufm-fabric-sync",
				SignalName:     "ib-partition-created",
			},
			// Async: when NICo deletes an IB partition, remove the PKEY from UFM.
			{
				Type:           providerv1.HookRegistration_ASYNC,
				Feature:        "networking",
				Event:          "post-delete-ib-partition",
				TargetWorkflow: "ufm-fabric-sync",
				SignalName:     "ib-partition-deleted",
			},
			// Sync: validate the PKEY doesn't conflict before NICo commits.
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "networking",
				Event:   "pre-create-ib-partition",
			},
		},
	}, nil
}

// HandleSyncHook handles the pre-create-ib-partition validation hook.
func (p *UFMFabricProvider) HandleSyncHook(ctx context.Context, event *providerv1.HookEvent) (*providerv1.HookResult, error) {
	if event.GetFeature() != "networking" || event.GetEvent() != "pre-create-ib-partition" {
		return &providerv1.HookResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("unknown sync hook: %s/%s", event.GetFeature(), event.GetEvent()),
		}, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(event.GetPayload(), &payload); err != nil {
		// Fail open — don't block partition creation on a parse error.
		log.Warn().Err(err).Msg("cannot parse IB partition hook payload, skipping UFM validation")
		return &providerv1.HookResult{Success: true}, nil
	}

	pkey, _ := payload["pkey"].(string)
	if pkey == "" {
		return &providerv1.HookResult{Success: true}, nil
	}

	existing, err := p.client.GetPKey(ctx, pkey, false)
	if err != nil {
		// 404 or other error means PKEY doesn't exist — validation passes.
		log.Debug().Err(err).Str("pkey", pkey).Msg("PKEY not found on UFM, validation passed")
		return &providerv1.HookResult{Success: true}, nil
	}

	if existing.Partition != "" {
		return &providerv1.HookResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("UFM validation failed: PKEY %s already exists as partition %q on UFM", pkey, existing.Partition),
		}, nil
	}

	return &providerv1.HookResult{Success: true}, nil
}

// GetOpenAPIFragment returns an empty OpenAPI fragment.
func (p *UFMFabricProvider) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}
