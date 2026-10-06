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

package conformance

import (
	"context"
	"net"
	"testing"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"google.golang.org/grpc"
)

// stubProvider implements all 9 NicoProvider RPCs with minimal valid responses.
type stubProvider struct {
	providerv1.UnimplementedNicoProviderServer
}

func (s *stubProvider) GetInfo(_ context.Context, _ *providerv1.GetInfoRequest) (*providerv1.ProviderInfo, error) {
	return &providerv1.ProviderInfo{
		Name:    "stub",
		Version: "0.0.1",
	}, nil
}

func (s *stubProvider) Init(_ context.Context, _ *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	return &providerv1.InitResponse{
		Ready:   true,
		Message: "stub ready",
	}, nil
}

func (s *stubProvider) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	return &providerv1.ShutdownResponse{}, nil
}

func (s *stubProvider) HealthCheck(_ context.Context, _ *providerv1.HealthCheckRequest) (*providerv1.HealthCheckResponse, error) {
	return &providerv1.HealthCheckResponse{
		Status: providerv1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *stubProvider) GetRoutes(_ context.Context, _ *providerv1.GetRoutesRequest) (*providerv1.RouteList, error) {
	return &providerv1.RouteList{
		Routes: []*providerv1.Route{
			{Method: "GET", Path: "/test"},
		},
	}, nil
}

func (s *stubProvider) HandleRequest(_ context.Context, _ *providerv1.HTTPRequest) (*providerv1.HTTPResponse, error) {
	return &providerv1.HTTPResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{}`),
	}, nil
}

func (s *stubProvider) GetHookRegistrations(_ context.Context, _ *providerv1.GetHookRegistrationsRequest) (*providerv1.HookRegistrationList, error) {
	return &providerv1.HookRegistrationList{
		Registrations: []*providerv1.HookRegistration{
			{
				Type:    providerv1.HookRegistration_SYNC,
				Feature: "compute",
				Event:   "post-create-instance",
			},
		},
	}, nil
}

func (s *stubProvider) HandleSyncHook(_ context.Context, _ *providerv1.HookEvent) (*providerv1.HookResult, error) {
	return &providerv1.HookResult{
		Success: true,
	}, nil
}

func (s *stubProvider) GetOpenAPIFragment(_ context.Context, _ *providerv1.GetOpenAPIFragmentRequest) (*providerv1.OpenAPIFragment, error) {
	return &providerv1.OpenAPIFragment{}, nil
}

func TestConformance(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	providerv1.RegisterNicoProviderServer(srv, &stubProvider{})
	go func() {
		if err := srv.Serve(lis); err != nil {
			// Serve returns after Stop is called; ignore.
		}
	}()
	defer srv.Stop()

	Run(t, TestConfig{
		ServerAddr:   lis.Addr().String(),
		ExpectedName: "stub",
		TestRoutes:   true,
	})
}
