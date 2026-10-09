// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build metal3

package instance

import (
	"github.com/rs/zerolog/log"

	mp "github.com/NVIDIA/infra-controller/rest-api/api/pkg/provisioner"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/provisioner/backends"
)

// RegisterProvisionerBackends reads PROVISIONER_BACKENDS and registers each
// named backend. Called at site-agent startup when compiled with -tags metal3.
func RegisterProvisionerBackends(r *mp.Registry) {
	for _, name := range mp.FromEnv() {
		switch name {
		case "metal3":
			if err := backends.RegisterMetal3(r); err != nil {
				log.Warn().Err(err).Msg("Metal3 provisioner unavailable — falling back to Core gRPC")
			} else {
				log.Info().Msg("Metal3 provisioner registered")
			}
		default:
			log.Warn().Str("backend", name).Msg("unknown provisioner backend in PROVISIONER_BACKENDS — ignored")
		}
	}
}
