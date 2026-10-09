// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

module github.com/NVIDIA/infra-controller/providers/provisioner-metal3

go 1.26.4

require (
	github.com/NVIDIA/infra-controller/provider-api v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/provider-sdk v0.0.0-00010101000000-000000000000
	github.com/rs/zerolog v1.34.0
	k8s.io/apimachinery v0.35.0
	k8s.io/client-go v0.35.0
	google.golang.org/grpc v1.81.0
)

replace github.com/NVIDIA/infra-controller/provider-api => ../../provider-api
replace github.com/NVIDIA/infra-controller/provider-sdk => ../../provider-sdk
