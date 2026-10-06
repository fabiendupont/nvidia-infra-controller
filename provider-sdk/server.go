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

package sdk

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type ServerConfig struct {
	ListenAddr string
}

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func NewServer(cfg ServerConfig, provider providerv1.NicoProviderServer) (*Server, error) {
	addr := cfg.ListenAddr
	if addr == "" {
		addr = os.Getenv("LISTEN_ADDR")
	}
	if addr == "" {
		addr = ":9443"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	var opts []grpc.ServerOption

	creds, err := LoadServerTLSCredentials()
	if err == nil {
		opts = append(opts, grpc.Creds(creds))
	} else {
		log.Warn().Err(err).Msg("TLS certs not available, running insecure")
	}

	srv := grpc.NewServer(opts...)
	providerv1.RegisterNicoProviderServer(srv, provider)

	log.Info().Str("addr", addr).Msg("provider server created")

	return &Server{
		grpcServer: srv,
		listener:   lis,
	}, nil
}

func (s *Server) Serve() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("received signal, shutting down")
		s.grpcServer.GracefulStop()
	}()

	log.Info().Str("addr", s.listener.Addr().String()).Msg("provider listening")
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
