// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"fmt"
	"os"

	"github.com/fabiendupont/nvidia-nvue-client-go/pkg/nvue"
)

// NVLinkConfigurator manages NVLink fabric switch ASICs (NVL72 inter-node fabric).
// NVLink switches run NVOS and expose the same NVUE REST API as Spectrum Ethernet
// switches. nv-config-manager uses the NVUE firmware path for upgrades:
//
//	GET  /nvue_v1/platform/firmware   — query installed firmware
//	POST /nvue_v1/platform/firmware   — initiate upgrade
//
// Full switch config push (VRF, BGP, EVPN) is not applicable to NVLink fabric
// switches — their connectivity is managed by NICo Core via NMX-C. This
// configurator covers only NVUE-accessible operations like firmware management.
type NVLinkConfigurator struct {
	opts ConfiguratorOptions
}

// NewNVLinkConfigurator creates an NVLinkConfigurator.
func NewNVLinkConfigurator(opts ConfiguratorOptions) (*NVLinkConfigurator, error) {
	return &NVLinkConfigurator{opts: opts}, nil
}

// Push for NVLink switches is not implemented — NVLink fabric connectivity is
// managed by NICo Core via NMX-C. Returns an explanatory error.
func (n *NVLinkConfigurator) Push(_ context.Context, switchID string) (*ConfigRecord, error) {
	return nil, fmt.Errorf("NVLinkConfigurator.Push: NVLink fabric configuration is managed by NICo Core via NMX-C; NVUE operations (e.g. firmware) are not dispatched through Push — switch_id=%s", switchID)
}

// Validate queries the NVLink switch's firmware version via the NVUE API and
// returns it as validation metadata. This mirrors nv-config-manager's use of
// GET /nvue_v1/platform/firmware for NVLink switch health checks.
func (n *NVLinkConfigurator) Validate(ctx context.Context, switchID string) (*ValidationResult, error) {
	if n.opts.SwitchService == nil {
		return nil, fmt.Errorf("switch service not available")
	}
	sw, err := n.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return nil, fmt.Errorf("switch %s has no management_ip", switchID)
	}

	username := os.Getenv("SWITCH_USERNAME")
	if username == "" {
		username = n.opts.SwitchUsername
	}
	password := os.Getenv("SWITCH_PASSWORD")
	if password == "" {
		password = n.opts.SwitchPassword
	}

	client := nvue.NewClient(sw.GetManagementIp(), username, password,
		nvue.WithInsecureSkipVerify())

	fwRaw, err := client.Get(ctx, "/platform/firmware", "applied")
	if err != nil {
		return nil, fmt.Errorf("NVUE GET /platform/firmware switch %s: %w", switchID, err)
	}

	return &ValidationResult{
		SwitchID:     switchID,
		SiteID:       sw.GetSiteId(),
		Match:        true,
		ObservedPeer: fmt.Sprintf("firmware: %s", fwRaw),
	}, nil
}

// UpgradeOS triggers a firmware upgrade on the NVLink switch via NVUE.
// NVLink switches run NVOS and support the same NVUE firmware path as Spectrum.
func (n *NVLinkConfigurator) UpgradeOS(ctx context.Context, switchID, targetVersion string) error {
	if n.opts.SwitchService == nil {
		return fmt.Errorf("switch service not available")
	}
	sw, err := n.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return fmt.Errorf("switch %s has no management_ip", switchID)
	}
	username := os.Getenv("SWITCH_USERNAME")
	if username == "" {
		username = n.opts.SwitchUsername
	}
	password := os.Getenv("SWITCH_PASSWORD")
	if password == "" {
		password = n.opts.SwitchPassword
	}
	client := nvue.NewClient(sw.GetManagementIp(), username, password, nvue.WithInsecureSkipVerify())
	patches := []nvue.PatchOp{{
		Path:    "/platform/firmware",
		Payload: map[string]interface{}{"target": targetVersion},
	}}
	if _, err := client.ConfigureAndApply(ctx, patches); err != nil {
		return fmt.Errorf("NVUE firmware upgrade NVLink switch %s to %s: %w", switchID, targetVersion, err)
	}
	return nil
}

// ValidateHardware is not applicable to NVLink fabric switches — hardware health
// is monitored by NICo Core. Returns a not-implemented error.
func (n *NVLinkConfigurator) ValidateHardware(_ context.Context, switchID string) (*HardwareHealthResult, error) {
	return nil, fmt.Errorf("ValidateHardware: NVLink switch hardware monitoring handled by NICo Core — switch_id=%s", switchID)
}

// GetDiagnostics is not applicable to NVLink fabric switches. Returns a
// not-implemented error.
func (n *NVLinkConfigurator) GetDiagnostics(_ context.Context, switchID string) (*SwitchDiagnostics, error) {
	return nil, fmt.Errorf("GetDiagnostics: NVLink switch diagnostics handled by NICo Core — switch_id=%s", switchID)
}
