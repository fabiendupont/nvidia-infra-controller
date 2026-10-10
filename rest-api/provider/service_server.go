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

package provider

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/NVIDIA/infra-controller/rest-api/common/pkg/certs"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/providers/compute/computesvc"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/providers/networking/networkingsvc"
)

// ServiceServer exposes core resource queries over gRPC so external
// providers can perform cross-domain lookups without direct DB access.
type ServiceServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

// NewServiceServer creates a gRPC server that registers the networking and
// compute cross-domain service implementations. It listens on listenAddr
// (e.g. ":8390") and uses mTLS when certificates are available, falling
// back to insecure transport in development.
func NewServiceServer(db *cdb.Session, listenAddr string) (*ServiceServer, error) {
	creds, err := certs.GRPCServerCredentials()
	if err != nil {
		log.Warn().Err(err).Msg("cross-domain service server: no TLS certs, using insecure transport")
		creds = insecure.NewCredentials()
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("cross-domain service server: failed to listen on %s: %w", listenAddr, err)
	}

	grpcServer := grpc.NewServer(grpc.Creds(creds))

	providerv1.RegisterNicoNetworkingServiceServer(grpcServer, &networkingServiceServer{
		svc: networkingsvc.New(db),
	})
	providerv1.RegisterNicoComputeServiceServer(grpcServer, &computeServiceServer{
		svc: computesvc.New(db),
	})
	providerv1.RegisterNicoSwitchServiceServer(grpcServer, &switchServiceServer{db: db})

	log.Info().Str("addr", lis.Addr().String()).Msg("cross-domain service server ready")

	return &ServiceServer{
		grpcServer: grpcServer,
		listener:   lis,
	}, nil
}

// Serve starts serving gRPC requests. Blocks until the server is stopped.
func (s *ServiceServer) Serve() error {
	return s.grpcServer.Serve(s.listener)
}

// Address returns the listener's address (useful when port 0 is used).
func (s *ServiceServer) Address() string {
	return s.listener.Addr().String()
}

// Stop gracefully shuts down the gRPC server.
func (s *ServiceServer) Stop() {
	s.grpcServer.GracefulStop()
}
