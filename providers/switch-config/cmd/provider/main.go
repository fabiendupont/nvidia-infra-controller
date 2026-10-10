// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk"
	switchconfig "github.com/NVIDIA/infra-controller/providers/switch-config/internal"
)

func main() {
	srv, err := sdk.NewServer(sdk.ServerConfig{}, switchconfig.NewServer())
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	if err := srv.Serve(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
