// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build metal3

// Package backends provides optional compiled-in provisioner backends for the
// site-agent. Each backend is gated behind a build tag so the standard binary
// carries no Metal3 or Ironic dependencies.
//
// Build with Metal3 support: go build -tags metal3 ./workflow/cmd/workflow/
package backends

import (
	"fmt"
	"os"

	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/provisioner"
	metal3 "github.com/NVIDIA/infra-controller/providers/provisioner-metal3"
)

// RegisterMetal3 constructs a Metal3Provisioner and registers it in the
// registry under the name "metal3". Called from main() when "metal3" is
// present in PROVISIONER_BACKENDS.
//
// Configuration via environment variables:
//   - KUBECONFIG or in-cluster service account (for BareMetalHost CR management)
//   - METAL3_NAMESPACE (default: "metal3-system")
func RegisterMetal3(r *provisioner.Registry) error {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	namespace := os.Getenv("METAL3_NAMESPACE")
	if namespace == "" {
		namespace = "metal3-system"
	}

	p, err := metal3.New(kubeconfigPath, namespace)
	if err != nil {
		return fmt.Errorf("construct Metal3 provisioner: %w", err)
	}

	r.Register("metal3", p)
	return nil
}
