// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package provisioner defines the pluggable machine provisioner interface.
// Implementations include the built-in Core gRPC path (CoreGRPCProvisioner)
// and external systems such as Metal3 Bare Metal Operator and OpenStack Ironic.
//
// The interface is deliberately narrow: it covers the three operations that
// differ across backends (provision, deprovision, power control). All other
// machine lifecycle operations (inventory sync, credential rotation, firmware
// updates) remain with the Core gRPC path regardless of which provisioner is
// registered.
package provisioner

import "context"

// MachineProvisioner is the interface for bare-metal machine provisioning backends.
type MachineProvisioner interface {
	// Name identifies the backend for observability (e.g. "core-grpc", "metal3", "ironic").
	Name() string

	// Provision installs an OS image on a machine and brings it to running state.
	// The call must block until provisioning completes or ctx is cancelled.
	Provision(ctx context.Context, req ProvisionRequest) error

	// Deprovision wipes the machine's disk and returns it to the available pool.
	Deprovision(ctx context.Context, machineID string) error

	// PowerControl issues a power action to the machine's BMC.
	PowerControl(ctx context.Context, machineID string, action PowerAction) error
}

// ProvisionRequest carries all inputs a provisioner needs to image a machine.
type ProvisionRequest struct {
	MachineID         string
	BMCURL            string
	BMCUsername       string
	BMCPassword       string
	ImageURL          string
	ImageChecksum     string
	ImageChecksumType string
	Hostname          string
	SSHKeys           []string
	// Extra holds provisioner-specific key-value configuration, e.g. Metal3
	// network data, Ironic deploy interface, IPMI driver hints.
	Extra map[string]string
}

// PowerAction is a machine power command.
type PowerAction string

const (
	PowerOn    PowerAction = "on"
	PowerOff   PowerAction = "off"
	PowerCycle PowerAction = "cycle"
	PowerReset PowerAction = "reset"
)

// CoreGRPCProvisioner is the default backend. It is a sentinel that tells the
// site-workflow activity to use the existing Core gRPC path unchanged. Provision
// and Deprovision are no-ops here because the ManageInstance activity handles
// them by calling Core gRPC directly; PowerControl is similarly a no-op because
// the existing power-control workflow path handles it.
//
// When no external MachineProvisioner is registered in ProviderContext, the
// compute provider falls back to this sentinel and the legacy path is used
// transparently.
type CoreGRPCProvisioner struct{}

func (*CoreGRPCProvisioner) Name() string                                            { return "core-grpc" }
func (*CoreGRPCProvisioner) Provision(_ context.Context, _ ProvisionRequest) error  { return nil }
func (*CoreGRPCProvisioner) Deprovision(_ context.Context, _ string) error          { return nil }
func (*CoreGRPCProvisioner) PowerControl(_ context.Context, _ string, _ PowerAction) error {
	return nil
}
