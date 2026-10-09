// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package provisioner

import (
	"os"
	"strings"
)

const (
	// AnnotationKey is the machine label key that selects a provisioner backend.
	// Set this on NICo machine resources to route provisioning to a specific backend.
	// Example: nico.nvidia.com/provisioner=metal3
	AnnotationKey = "nico.nvidia.com/provisioner"

	// BackendCoreGRPC is the built-in Core gRPC provisioner name (always-present default).
	BackendCoreGRPC = "core-grpc"
)

// Registry maps provisioner backend names to MachineProvisioner implementations.
// CoreGRPCProvisioner is always registered under "" and "core-grpc" as the
// fallback for machines that carry no provisioner annotation.
type Registry struct {
	backends map[string]MachineProvisioner
}

// NewRegistry returns a Registry with CoreGRPCProvisioner pre-registered as
// the default. Additional backends are added via Register.
func NewRegistry() *Registry {
	r := &Registry{backends: make(map[string]MachineProvisioner)}
	def := &CoreGRPCProvisioner{}
	r.backends[""] = def
	r.backends[BackendCoreGRPC] = def
	return r
}

// Register adds a named backend. name must match the annotation value on the
// machine (e.g. "metal3" matches nico.nvidia.com/provisioner=metal3).
func (r *Registry) Register(name string, p MachineProvisioner) {
	r.backends[name] = p
}

// Select returns the MachineProvisioner for the given annotation value.
// If the annotation is absent or the named backend is unknown, CoreGRPCProvisioner
// is returned so the existing Core gRPC path runs unchanged.
func (r *Registry) Select(annotationValue string) MachineProvisioner {
	if p, ok := r.backends[annotationValue]; ok {
		return p
	}
	return r.backends[""]
}

// FromEnv reads the PROVISIONER_BACKENDS environment variable (comma-separated
// backend names) and returns the list of names to register. "core-grpc" and
// empty strings are filtered out — CoreGRPCProvisioner is always registered.
// Example: PROVISIONER_BACKENDS=metal3
func FromEnv() []string {
	raw := os.Getenv("PROVISIONER_BACKENDS")
	if raw == "" {
		return nil
	}
	var names []string
	for _, n := range strings.Split(raw, ",") {
		n = strings.TrimSpace(n)
		if n != "" && n != BackendCoreGRPC {
			names = append(names, n)
		}
	}
	return names
}
