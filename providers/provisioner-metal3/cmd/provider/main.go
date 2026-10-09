// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk"
	metal3 "github.com/NVIDIA/infra-controller/providers/provisioner-metal3/internal"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	srv, err := sdk.NewServer(sdk.ServerConfig{}, metal3.NewServer())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create gRPC server")
	}
	if err := srv.Serve(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
