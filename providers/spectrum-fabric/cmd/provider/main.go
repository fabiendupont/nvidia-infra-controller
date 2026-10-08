// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk"
	spectrumfabric "github.com/NVIDIA/infra-controller/providers/spectrum-fabric/internal"
)

func main() {
	srv, err := sdk.NewServer(sdk.ServerConfig{}, spectrumfabric.NewServer())
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	if err := srv.Serve(); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
