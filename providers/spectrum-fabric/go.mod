// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

module github.com/NVIDIA/infra-controller/providers/spectrum-fabric

go 1.26.4

require (
	github.com/NVIDIA/infra-controller/provider-api v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/provider-sdk v0.0.0-00010101000000-000000000000
	github.com/fabiendupont/nvidia-nvue-client-go v0.511.0
	github.com/rs/zerolog v1.34.0
	github.com/stretchr/testify v1.11.1
)

replace github.com/NVIDIA/infra-controller/provider-api => ../../provider-api
replace github.com/NVIDIA/infra-controller/provider-sdk => ../../provider-sdk
