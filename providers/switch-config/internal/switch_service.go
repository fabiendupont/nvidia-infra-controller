// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"fmt"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SwitchClient wraps NicoSwitchServiceClient with convenience methods.
type SwitchClient struct {
	client providerv1.NicoSwitchServiceClient
}

// NewSwitchClient dials the NicoSwitchService endpoint and returns a client.
func NewSwitchClient(endpoint string) (*SwitchClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("switch service endpoint not provided in InitRequest.service_endpoints[\"switch\"]")
	}
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial switch service %q: %w", endpoint, err)
	}
	return &SwitchClient{client: providerv1.NewNicoSwitchServiceClient(conn)}, nil
}

// GetSwitch fetches a single switch record.
func (c *SwitchClient) GetSwitch(ctx context.Context, switchID string) (*providerv1.NicoSwitch, error) {
	resp, err := c.client.GetSwitch(ctx, &providerv1.GetSwitchRequest{Id: switchID})
	if err != nil {
		return nil, err
	}
	return resp.GetSwitch(), nil
}

// ListSwitches returns all switches for a site.
func (c *SwitchClient) ListSwitches(ctx context.Context, siteID string) ([]*providerv1.NicoSwitch, error) {
	resp, err := c.client.ListSwitches(ctx, &providerv1.ListSwitchesRequest{SiteId: siteID})
	if err != nil {
		return nil, err
	}
	return resp.GetSwitches(), nil
}

// GetRoutingConfig fetches the intended routing configuration for a switch.
func (c *SwitchClient) GetRoutingConfig(ctx context.Context, switchID string) (*providerv1.SwitchRoutingConfig, error) {
	return c.client.GetSwitchRoutingConfig(ctx, &providerv1.GetSwitchRoutingConfigRequest{SwitchId: switchID})
}

// ListExpectedNeighbors returns the expected LLDP neighbors for a switch's ports.
func (c *SwitchClient) ListExpectedNeighbors(ctx context.Context, siteID, switchID string) ([]*providerv1.CableLink, error) {
	resp, err := c.client.ListExpectedNeighbors(ctx, &providerv1.ListExpectedNeighborsRequest{SwitchId: switchID})
	if err != nil {
		return nil, err
	}
	return resp.GetExpectedNeighbors(), nil
}

// ReportObservedNeighbors writes LLDP-observed neighbors back to NICo.
func (c *SwitchClient) ReportObservedNeighbors(ctx context.Context, siteID, switchID string, neighbors []*providerv1.CableLink) error {
	_, err := c.client.ReportObservedNeighbors(ctx, &providerv1.ReportObservedNeighborsRequest{
		SwitchId:  switchID,
		ObservedNeighbors: neighbors,
	})
	return err
}
