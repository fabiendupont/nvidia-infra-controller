// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package configurator defines the SwitchConfigurator interface — the contract
// for the switch-config feature. Both the built-in Spectrum provider and external
// providers (Netris, custom) implement this interface.
//
// Runtime discovery uses ConfigMap labels (nico.nvidia.com/switch-config-provider)
// rather than compile-time registration, but the interface is shared as a Go
// module dependency — the same model as NicoProviderServer in provider-api.
package configurator

import "context"

// AnnotationKey is the expected_switch label that routes a switch to a specific
// switch-config provider implementation.
//
//	nico.nvidia.com/switch-config-provider: spectrum   → nico-switch-config-spectrum
//	nico.nvidia.com/switch-config-provider: netris     → nico-switch-config-netris
const AnnotationKey = "nico.nvidia.com/switch-config-provider"

// SwitchRef identifies a switch and the provider that should configure it.
type SwitchRef struct {
	SwitchID string
	Name     string
	// Provider is the value of AnnotationKey. Defaults to "spectrum" when absent.
	Provider string
}

// SwitchConfigurator is the interface all switch-config provider implementations
// must satisfy. Use ConfiguratorProvider() to declare which annotation value
// ("spectrum", "netris", etc.) the implementation handles.
type SwitchConfigurator interface {
	// ConfiguratorProvider returns the annotation value this implementation handles.
	ConfiguratorProvider() string

	// Push reads desired state from NicoSwitchService and applies it to the switch.
	Push(ctx context.Context, switchID string) (*PushResult, error)

	// Validate compares the switch's running config against NICo's desired state.
	Validate(ctx context.Context, switchID string) (*ValidateResult, error)

	// UpgradeOS upgrades the switch OS/firmware to targetVersion.
	UpgradeOS(ctx context.Context, switchID, targetVersion string) error

	// ValidateHardware checks physical hardware health (PSUs, fans, transceivers).
	ValidateHardware(ctx context.Context, switchID string) (*HardwareHealth, error)

	// GetDiagnostics returns a point-in-time operational snapshot.
	GetDiagnostics(ctx context.Context, switchID string) (*Diagnostics, error)
}

// PushResult is returned by Push on success.
type PushResult struct {
	SwitchID    string
	Format      string // e.g. "nvue-json", "netris-api"
	ContentHash string // SHA-256 of the applied config
}

// ValidateResult is returned by Validate.
type ValidateResult struct {
	SwitchID  string
	Compliant bool
	Diffs     []string
	Errors    []string
}

// HardwareHealth is returned by ValidateHardware.
type HardwareHealth struct {
	Healthy      bool
	PSUs         []ComponentHealth
	Fans         []ComponentHealth
	Transceivers []ComponentHealth
	Errors       []string
}

// ComponentHealth describes the state of one hardware component.
type ComponentHealth struct {
	Name   string
	State  string // "ok", "failed", "absent"
	Detail string
}

// Diagnostics is returned by GetDiagnostics.
type Diagnostics struct {
	SwitchID     string
	BGPNeighbors []BGPNeighbor
	MACEntries   []MACEntry
	MLAGState    string
	Errors       []string
}

// BGPNeighbor is one BGP peer from the switch routing table.
type BGPNeighbor struct {
	PeerIP   string
	RemoteAS uint32
	State    string // "Established", "Idle", "Active", etc.
	VRF      string
}

// MACEntry is one MAC forwarding table entry.
type MACEntry struct {
	MAC       string
	VID       int
	Interface string
}
