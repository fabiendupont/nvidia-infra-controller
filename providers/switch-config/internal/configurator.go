// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"fmt"
)

// SwitchConfigurator is the internal interface for in-module implementations
// (Spectrum, NVLink). It uses ConfigRecord and ValidationResult from
// config_store.go — the bun DB models — as return types so results are
// automatically persisted.
//
// External providers (Netris, custom) implement the public interface defined
// in ../configurator/configurator.go, which uses simpler result types.
// Both interfaces share the AnnotationKey convention and ConfigMap discovery.
type SwitchConfigurator interface {
	// ConfiguratorProvider returns the annotation value this implementation handles.
	ConfiguratorProvider() string

	Push(ctx context.Context, switchID string) (*ConfigRecord, error)
	Validate(ctx context.Context, switchID string) (*ValidationResult, error)
	UpgradeOS(ctx context.Context, switchID, targetVersion string) error
	ValidateHardware(ctx context.Context, switchID string) (*HardwareHealthResult, error)
	GetDiagnostics(ctx context.Context, switchID string) (*SwitchDiagnostics, error)
}

// SwitchRef identifies a switch and the switch-config provider that should
// configure it, derived from the nico.nvidia.com/switch-config-provider label.
type SwitchRef struct {
	SwitchID string
	Name     string
	// Provider is the value of nico.nvidia.com/switch-config-provider.
	// Defaults to "spectrum" when absent.
	Provider string
}

// HardwareHealthResult holds the outcome of a hardware health check.
type HardwareHealthResult struct {
	Healthy      bool
	PSUs         []ComponentHealth
	Fans         []ComponentHealth
	Transceivers []ComponentHealth
	Errors       []string
}

// ComponentHealth describes the state of a single hardware component.
type ComponentHealth struct {
	Name   string
	State  string // "ok", "failed", "absent"
	Detail string
}

// SwitchDiagnostics holds a point-in-time operational snapshot of a switch.
type SwitchDiagnostics struct {
	SwitchID     string
	BGPNeighbors []BGPNeighbor
	MACEntries   []MACEntry
	MLAGState    string
	Errors       []string
}

// BGPNeighbor is a single BGP peer entry from the switch's routing table.
type BGPNeighbor struct {
	PeerIP   string
	RemoteAS uint32
	State    string
	VRF      string
}

// MACEntry is a single MAC forwarding table entry.
type MACEntry struct {
	MAC       string
	VID       int
	Interface string
}

// ConfiguratorOptions carries the shared dependencies for in-module configurator
// implementations (Spectrum, NVLink).
type ConfiguratorOptions struct {
	SwitchService  *SwitchClient
	ConfigStore    *ConfigStore
	InsecureTLS    bool
	SwitchUsername string
	SwitchPassword string
}

// NewSwitchConfigurator returns the in-module configurator for the given platform.
// For external platforms (netris, etc.), deploy the dedicated provider instead.
func NewSwitchConfigurator(platform string, opts ConfiguratorOptions) (SwitchConfigurator, error) {
	switch platform {
	case "spectrum", "nvos", "":
		return NewSpectrumConfigurator(opts)
	case "nvlink":
		return NewNVLinkConfigurator(opts)
	default:
		return nil, fmt.Errorf("unsupported platform %q — built-in: spectrum, nvos, nvlink; "+
			"for netris deploy the nico-switch-config-netris provider", platform)
	}
}
