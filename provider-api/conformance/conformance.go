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
	"fmt"
	"testing"
	"time"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestConfig configures the conformance test run.
type TestConfig struct {
	// ServerAddr is the gRPC address of the running provider (e.g., "localhost:9443").
	ServerAddr string

	// ExpectedName is the provider name returned by GetInfo.
	ExpectedName string

	// ExpectedFeatures is the list of features the provider should declare.
	ExpectedFeatures []string

	// TestRoutes, if true, tests HandleRequest for each declared route.
	TestRoutes bool

	// InitConfig is passed to the provider's Init RPC.
	InitConfig map[string]string
}

// Run executes the conformance test suite against a running provider.
// Each protocol step is run as a subtest so failures are granular.
func Run(t *testing.T, cfg TestConfig) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Connect
	var conn *grpc.ClientConn
	t.Run("Connect", func(t *testing.T) {
		var err error
		conn, err = grpc.NewClient(
			cfg.ServerAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			t.Fatalf("failed to connect to %s: %v", cfg.ServerAddr, err)
		}
	})
	if conn == nil {
		t.Fatal("connection not established, skipping remaining tests")
		return
	}
	defer conn.Close()

	client := providerv1.NewNicoProviderClient(conn)

	// Step 2: GetInfo
	var routes *providerv1.RouteList
	t.Run("GetInfo", func(t *testing.T) {
		info, err := client.GetInfo(ctx, &providerv1.GetInfoRequest{})
		if err != nil {
			t.Fatalf("GetInfo failed: %v", err)
		}
		if info.GetName() != cfg.ExpectedName {
			t.Errorf("expected name %q, got %q", cfg.ExpectedName, info.GetName())
		}
		if info.GetVersion() == "" {
			t.Error("version must be non-empty")
		}
		if len(cfg.ExpectedFeatures) > 0 {
			got := make(map[string]bool)
			for _, f := range info.GetFeatures() {
				got[f] = true
			}
			for _, want := range cfg.ExpectedFeatures {
				if !got[want] {
					t.Errorf("expected feature %q not declared (got %v)", want, info.GetFeatures())
				}
			}
		}
	})

	// Step 3: Init
	t.Run("Init", func(t *testing.T) {
		resp, err := client.Init(ctx, &providerv1.InitRequest{
			Config: cfg.InitConfig,
		})
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if !resp.GetReady() {
			t.Errorf("Init returned Ready=false, message: %s", resp.GetMessage())
		}
	})

	// Step 4: HealthCheck
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := client.HealthCheck(ctx, &providerv1.HealthCheckRequest{})
		if err != nil {
			t.Fatalf("HealthCheck failed: %v", err)
		}
		if resp.GetStatus() != providerv1.HealthCheckResponse_SERVING {
			t.Errorf("expected SERVING status, got %v", resp.GetStatus())
		}
	})

	// Step 5: GetRoutes
	t.Run("GetRoutes", func(t *testing.T) {
		var err error
		routes, err = client.GetRoutes(ctx, &providerv1.GetRoutesRequest{})
		if err != nil {
			t.Fatalf("GetRoutes failed: %v", err)
		}
		if routes == nil {
			t.Fatal("GetRoutes returned nil")
		}
	})

	// Step 6: GetHookRegistrations
	t.Run("GetHookRegistrations", func(t *testing.T) {
		resp, err := client.GetHookRegistrations(ctx, &providerv1.GetHookRegistrationsRequest{})
		if err != nil {
			t.Fatalf("GetHookRegistrations failed: %v", err)
		}
		if resp == nil {
			t.Fatal("GetHookRegistrations returned nil")
		}
		for i, reg := range resp.GetRegistrations() {
			if reg.GetFeature() == "" {
				t.Errorf("registration[%d]: feature must be non-empty", i)
			}
			if reg.GetEvent() == "" {
				t.Errorf("registration[%d]: event must be non-empty", i)
			}
			if reg.GetType() != providerv1.HookRegistration_SYNC && reg.GetType() != providerv1.HookRegistration_ASYNC {
				t.Errorf("registration[%d]: invalid type %v", i, reg.GetType())
			}
		}
	})

	// Step 7: HandleRequest
	if cfg.TestRoutes && routes != nil {
		for _, route := range routes.GetRoutes() {
			name := fmt.Sprintf("HandleRequest/%s_%s", route.GetMethod(), route.GetPath())
			t.Run(name, func(t *testing.T) {
				req := &providerv1.HTTPRequest{
					Method: route.GetMethod(),
					Path:   route.GetPath(),
				}
				switch route.GetMethod() {
				case "POST", "PUT", "PATCH":
					req.Body = []byte(`{}`)
				}
				resp, err := client.HandleRequest(ctx, req)
				if err != nil {
					t.Fatalf("HandleRequest failed for %s %s: %v", route.GetMethod(), route.GetPath(), err)
				}
				if resp.GetStatusCode() == 0 {
					t.Errorf("HandleRequest returned status_code=0 for %s %s", route.GetMethod(), route.GetPath())
				}
			})
		}
	}

	// Step 8: GetOpenAPIFragment
	t.Run("GetOpenAPIFragment", func(t *testing.T) {
		_, err := client.GetOpenAPIFragment(ctx, &providerv1.GetOpenAPIFragmentRequest{})
		if err != nil {
			t.Fatalf("GetOpenAPIFragment failed: %v", err)
		}
	})

	// Step 9: Shutdown
	t.Run("Shutdown", func(t *testing.T) {
		_, err := client.Shutdown(ctx, &providerv1.ShutdownRequest{})
		if err != nil {
			t.Fatalf("Shutdown failed: %v", err)
		}
	})

	// Step 10: HealthCheck after shutdown
	t.Run("HealthCheckAfterShutdown", func(t *testing.T) {
		resp, err := client.HealthCheck(ctx, &providerv1.HealthCheckRequest{})
		if err != nil {
			// Connection refused or transport error is acceptable after shutdown.
			t.Logf("HealthCheck after shutdown returned error (acceptable): %v", err)
			return
		}
		if resp.GetStatus() == providerv1.HealthCheckResponse_SERVING {
			t.Log("provider still reports SERVING after shutdown (not required to change)")
		}
	})
}
