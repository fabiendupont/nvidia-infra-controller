/*
 * SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	"github.com/NVIDIA/infra-controller/rest-api/common/pkg/certs"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":9443"
	}

	// Build gRPC server options. Use mTLS when certificates are available;
	// fall back to insecure for local development.
	var opts []grpc.ServerOption
	if certs.IsTLSAvailable(certs.Config{}) {
		cred, err := certs.GRPCServerCredentials()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load TLS credentials")
		}
		opts = append(opts, grpc.Creds(cred))
		log.Info().Msg("TLS enabled")
	} else {
		log.Warn().Msg("TLS certificates not found, running insecure (dev mode)")
	}

	srv := grpc.NewServer(opts...)
	providerv1.RegisterNicoProviderServer(srv, NewNetworkingProviderServer())

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal().Err(err).Str("addr", listenAddr).Msg("failed to listen")
	}

	// Graceful shutdown on SIGTERM/SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("shutting down")
		srv.GracefulStop()
	}()

	log.Info().Str("addr", listenAddr).Msg("networking provider gRPC server starting")
	if err := srv.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("server exited with error")
	}
}
