// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

module github.com/NVIDIA/infra-controller/providers/fulfillment

go 1.26.4

require (
	github.com/NVIDIA/infra-controller/provider-api v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/provider-sdk v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/providers/catalog v0.0.0-00010101000000-000000000000
	// TODO: cross-domain access to networking/compute should use provider-api/rest/* gRPC clients,
	// not direct rest-api imports. Refactor provisioning_activities.go in Phase 1.
	github.com/NVIDIA/infra-controller/rest-api v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/labstack/echo/v4 v4.13.3
	github.com/rs/zerolog v1.34.0
	github.com/stretchr/testify v1.11.1
	go.temporal.io/sdk v1.34.0
)
replace github.com/NVIDIA/infra-controller/provider-api => ../../provider-api
replace github.com/NVIDIA/infra-controller/provider-sdk => ../../provider-sdk
replace github.com/NVIDIA/infra-controller/providers/catalog => ../../providers/catalog
replace github.com/NVIDIA/infra-controller/rest-api => ../../rest-api
