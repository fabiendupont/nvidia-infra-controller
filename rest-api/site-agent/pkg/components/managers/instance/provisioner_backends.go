// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build !metal3

package instance

import mp "github.com/NVIDIA/infra-controller/rest-api/api/pkg/provisioner"

// RegisterProvisionerBackends is a no-op in the standard build.
// Compile with -tags metal3 to include the Metal3 Bare Metal Operator backend.
func RegisterProvisionerBackends(_ *mp.Registry) {}
