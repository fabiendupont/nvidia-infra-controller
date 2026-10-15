// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"fmt"
)

// SwitchConfigurator applies desired switch configuration derived from NICo's
// data model to a physical switch. Implementations are vendor-specific.
type SwitchConfigurator interface {
	// Push reads desired state from NicoSwitchService and applies it to the
	// switch identified by switchID via the switch's management API.
	// Returns the stored ConfigRecord on success.
	Push(ctx context.Context, switchID string) (*ConfigRecord, error)

	// Validate compares the switch's running configuration against the
	// desired state computed from NICo's data model.
	Validate(ctx context.Context, switchID string) (*ValidationResult, error)
}

// ConfiguratorOptions carries the shared dependencies all configurator
// implementations require.
type ConfiguratorOptions struct {
	SwitchService *SwitchClient // reads NICo switch state via NicoSwitchService
	ConfigStore   *ConfigStore  // stores and versions applied configs
	InsecureTLS   bool          // skip TLS verification (dev only)
	// SwitchUsername / SwitchPassword are fallback credentials used when the
	// switch does not yet have a credential_ref resolvable via Vault.
	SwitchUsername string
	SwitchPassword string
}

// NewSwitchConfigurator returns the configurator for the given platform.
//
// Supported platforms:
//   - "spectrum", "nvos", "" — NVIDIA Spectrum Ethernet switches via NVUE REST API
//   - "nvlink" — NVLink fabric switch ASICs (NVL72 inter-node fabric);
//     managed by NICo Core via NMX-C. This case is a placeholder for
//     future per-switch NVLink operations beyond what NICo Core covers.
func NewSwitchConfigurator(platform string, opts ConfiguratorOptions) (SwitchConfigurator, error) {
	switch platform {
	case "spectrum", "nvos", "":
		return NewSpectrumConfigurator(opts)
	case "nvlink":
		// NVLink switches (NVL72 inter-node fabric ASICs) run NVOS and expose the
		// same NVUE REST API as Spectrum Ethernet switches. nv-config-manager uses
		// the NVUE firmware path (GET/POST /nvue_v1/platform/firmware) for NVLink
		// switch firmware upgrades. Full config push is out of scope for now.
		return NewNVLinkConfigurator(opts)
	default:
		return nil, fmt.Errorf("unsupported platform %q — supported: spectrum, nvos, nvlink", platform)
	}
}
