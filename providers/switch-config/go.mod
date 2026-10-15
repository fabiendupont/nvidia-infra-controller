// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

module github.com/NVIDIA/infra-controller/providers/switch-config

go 1.26.4

require (
	github.com/NVIDIA/infra-controller/provider-api v0.0.0-00010101000000-000000000000
	github.com/NVIDIA/infra-controller/provider-sdk v0.0.0-00010101000000-000000000000
	github.com/fabiendupont/nvidia-nvue-client-go v0.511.0
	github.com/google/uuid v1.6.0
	github.com/labstack/echo/v4 v4.13.3
	github.com/rs/zerolog v1.34.0
	github.com/uptrace/bun v1.2.18
	github.com/uptrace/bun/dialect/pgdialect v1.2.18
	go.temporal.io/sdk v1.34.0
	google.golang.org/grpc v1.73.0
)

replace github.com/NVIDIA/infra-controller/provider-api => ../../provider-api
replace github.com/NVIDIA/infra-controller/provider-sdk => ../../provider-sdk
replace github.com/fabiendupont/nvidia-nvue-client-go => /home/fdupont/Code/github.com/NVIDIA/nvue-client-go
