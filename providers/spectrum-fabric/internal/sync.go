// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package spectrumfabric

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/fabiendupont/nvidia-nvue-client-go/pkg/nvue"
)

// SyncVPCToFabric creates a VRF on the Spectrum switches for the given NICo VPC.
func (p *Server) SyncVPCToFabric(ctx context.Context, vpcID, vpcName, tenantID string) error {
	if !p.config.Features.SyncVPC {
		return nil
	}

	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	vrfName := fmt.Sprintf("nico-%s", vpcID)
	log.Info().Str("provider", "spectrum-fabric").Str("vrf_name", vrfName).Msg("creating VRF on Spectrum fabric")

	vrfConfig := map[string]any{
		"router": map[string]any{
			"bgp": map[string]any{
				"enable":    "on",
				"router-id": "auto",
				"address-family": map[string]any{
					"ipv4-unicast": map[string]any{
						"enable":       "on",
						"redistribute": map[string]any{"connected": map[string]any{"enable": "on"}},
						"route-export": map[string]any{"to-evpn": map[string]any{"enable": "on"}},
					},
					"l2vpn-evpn": map[string]any{"enable": "on"},
				},
			},
		},
		"evpn": map[string]any{"enable": "on"},
	}

	_, err := p.client.ConfigureAndApply(ctx, []nvue.PatchOp{
		{Path: fmt.Sprintf("/vrf/%s", vrfName), Payload: vrfConfig},
	})
	if err != nil {
		return fmt.Errorf("creating VRF %s on Spectrum fabric: %w", vrfName, err)
	}

	log.Info().Str("vrf_name", vrfName).Msg("VRF created on Spectrum fabric")
	return nil
}

// RemoveVPCFromFabric removes the VRF from the Spectrum switches for the given NICo VPC.
func (p *Server) RemoveVPCFromFabric(ctx context.Context, vpcID string) error {
	if !p.config.Features.SyncVPC {
		return nil
	}

	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	vrfName := fmt.Sprintf("nico-%s", vpcID)
	log.Info().Str("vrf_name", vrfName).Msg("removing VRF from Spectrum fabric")

	revID, err := p.client.CreateRevisionID(ctx)
	if err != nil {
		return fmt.Errorf("creating NVUE revision: %w", err)
	}

	if err := p.client.Delete(ctx, fmt.Sprintf("/vrf/%s", vrfName), revID); err != nil {
		return fmt.Errorf("deleting VRF %s: %w", vrfName, err)
	}

	if _, err := p.client.ApplyAndWait(ctx, revID); err != nil {
		return fmt.Errorf("applying VRF deletion for %s: %w", vrfName, err)
	}

	return nil
}

// SyncSubnetToFabric creates VxLAN VNI and bridge VLAN configuration on the Spectrum switches.
func (p *Server) SyncSubnetToFabric(ctx context.Context, subnetID, vpcID, prefix, subnetName string, vlanID, vni int) error {
	if !p.config.Features.SyncSubnet {
		return nil
	}

	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	vrfName := fmt.Sprintf("nico-%s", vpcID)
	sviName := fmt.Sprintf("vlan%d", vlanID)

	bridgeConfig := map[string]any{
		"vlan": map[string]any{
			fmt.Sprintf("%d", vlanID): map[string]any{
				"vni": map[string]any{
					fmt.Sprintf("%d", vni): map[string]any{
						"flooding": map[string]any{
							"enable":               "on",
							"head-end-replication": map[string]any{},
						},
					},
				},
			},
		},
	}

	sviConfig := map[string]any{
		"type": "svi",
		"ip": map[string]any{
			"address": map[string]any{prefix: map[string]any{}},
			"vrf":     vrfName,
		},
	}

	nveConfig := map[string]any{"enable": "on"}

	_, err := p.client.ConfigureAndApply(ctx, []nvue.PatchOp{
		{Path: "/bridge/domain/br_default", Payload: bridgeConfig},
		{Path: fmt.Sprintf("/interface/%s", sviName), Payload: sviConfig},
		{Path: "/nve/vxlan", Payload: nveConfig},
	})
	if err != nil {
		return fmt.Errorf("creating VxLAN VNI %d for subnet %s: %w", vni, subnetID, err)
	}

	return nil
}

// RemoveSubnetFromFabric removes VxLAN VNI and bridge VLAN configuration from the Spectrum switches.
func (p *Server) RemoveSubnetFromFabric(ctx context.Context, subnetID, vpcID string, vlanID int) error {
	if !p.config.Features.SyncSubnet {
		return nil
	}

	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	sviName := fmt.Sprintf("vlan%d", vlanID)

	revID, err := p.client.CreateRevisionID(ctx)
	if err != nil {
		return fmt.Errorf("creating NVUE revision: %w", err)
	}

	if err := p.client.Delete(ctx, fmt.Sprintf("/interface/%s", sviName), revID); err != nil {
		return fmt.Errorf("deleting SVI %s: %w", sviName, err)
	}

	if err := p.client.Delete(ctx, fmt.Sprintf("/bridge/domain/br_default/vlan/%d", vlanID), revID); err != nil {
		return fmt.Errorf("deleting bridge VLAN %d: %w", vlanID, err)
	}

	if _, err := p.client.ApplyAndWait(ctx, revID); err != nil {
		return fmt.Errorf("applying VxLAN VNI removal for subnet %s: %w", subnetID, err)
	}

	return nil
}
