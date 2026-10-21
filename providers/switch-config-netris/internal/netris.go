// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package netris

import (
	"context"
	"fmt"

	cfg "github.com/NVIDIA/infra-controller/providers/switch-config/configurator"
)

// NetrisConfigurator implements cfg.SwitchConfigurator for switches managed
// by the Netris SDN controller. Netris provides a declarative API for BGP,
// EVPN, and VxLAN on supported switch platforms (Cumulus, SONiC, Arista, etc.)
//
// Annotation: nico.nvidia.com/switch-config-provider=netris on expected_switch
//
// Netris REST API: https://<controller>/api/v1/
// Reference: https://www.netris.io/docs/api/
type NetrisConfigurator struct {
	apiURL   string // e.g. https://netris.example.com
	apiToken string
}

// NewNetrisConfigurator creates a NetrisConfigurator.
func NewNetrisConfigurator(apiURL, apiToken string) *NetrisConfigurator {
	return &NetrisConfigurator{apiURL: apiURL, apiToken: apiToken}
}

// ConfiguratorProvider returns the annotation value this implementation handles.
func (n *NetrisConfigurator) ConfiguratorProvider() string { return "netris" }

// Push pushes NICo-derived VRF, BGP, and VxLAN config to the Netris controller.
// Netris then programs the underlying switches.
//
// Netris API:
//   - POST /api/v1/vpc         — create VPC (maps to NICo VPC/VRF)
//   - POST /api/v1/bgp         — create BGP sessions
//   - POST /api/v1/vnet        — create L2 VXLAN segment
//   - POST /api/v1/ipam/prefix — allocate subnet prefix
func (n *NetrisConfigurator) Push(_ context.Context, switchID string) (*cfg.PushResult, error) {
	return nil, fmt.Errorf(
		"NetrisConfigurator.Push(%s): not yet implemented — "+
			"POST %s/api/v1/vpc for VRFs, "+
			"/api/v1/bgp for BGP sessions, "+
			"/api/v1/vnet for L2 VXLAN segments",
		switchID, n.apiURL)
}

// Validate compares Netris-managed state against NICo's desired state.
//
// Netris API:
//   - GET /api/v1/vpc    — list VPCs; compare names/routes vs NICo VPCs
//   - GET /api/v1/bgp    — list BGP sessions; compare vs NICo expected peers
//   - GET /api/v1/vnet   — list L2 segments; compare vs NICo subnets
func (n *NetrisConfigurator) Validate(_ context.Context, switchID string) (*cfg.ValidateResult, error) {
	return nil, fmt.Errorf(
		"NetrisConfigurator.Validate(%s): not yet implemented — "+
			"GET %s/api/v1/vpc and /api/v1/bgp to compare vs NICo model",
		switchID, n.apiURL)
}

// UpgradeOS is not applicable for Netris. Netris is a software SDN controller;
// underlying switch OS upgrades use the switch vendor's own mechanism.
func (n *NetrisConfigurator) UpgradeOS(_ context.Context, switchID, targetVersion string) error {
	return fmt.Errorf(
		"NetrisConfigurator.UpgradeOS(%s, %s): not applicable — "+
			"Netris is an SDN controller; upgrade the switch OS directly via ONIE or the vendor CLI",
		switchID, targetVersion)
}

// ValidateHardware is not applicable for Netris. Hardware health is reported
// by the switch itself (NVUE, EOS, etc.), not via the Netris controller.
func (n *NetrisConfigurator) ValidateHardware(_ context.Context, switchID string) (*cfg.HardwareHealth, error) {
	return nil, fmt.Errorf(
		"NetrisConfigurator.ValidateHardware(%s): not applicable — "+
			"use the switch vendor API (NVUE, EOS) directly for hardware health",
		switchID)
}

// GetDiagnostics returns Netris-level operational state.
//
// Netris API:
//   - GET /api/v1/bgp?switch=<id>  — BGP session state as seen by Netris
//   - GET /api/v1/vnet?switch=<id> — L2 segment state
func (n *NetrisConfigurator) GetDiagnostics(_ context.Context, switchID string) (*cfg.Diagnostics, error) {
	return nil, fmt.Errorf(
		"NetrisConfigurator.GetDiagnostics(%s): not yet implemented — "+
			"GET %s/api/v1/bgp?switch=%s for BGP state",
		switchID, n.apiURL, switchID)
}

// Verify interface compliance at compile time.
var _ cfg.SwitchConfigurator = (*NetrisConfigurator)(nil)
