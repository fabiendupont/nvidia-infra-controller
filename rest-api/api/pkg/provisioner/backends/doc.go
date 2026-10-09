// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package backends contains optional compiled-in provisioner backends for the
// site-agent workflow binary. Each backend is gated behind a build tag:
//
//   - metal3: go build -tags metal3 ./workflow/cmd/workflow/
//
// The standard binary (no tags) contains only CoreGRPCProvisioner, which
// delegates to the existing NICo Core gRPC path unchanged.
package backends
