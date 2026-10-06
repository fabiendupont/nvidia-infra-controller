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

// Package conformance provides a test harness for verifying that a
// NicoProvider gRPC server correctly implements the provider protocol.
//
// The harness tests the full NicoProvider lifecycle: connection, info,
// initialization, health checks, route handling, hook registrations,
// OpenAPI fragments, and shutdown. It is designed to be called from a
// provider's own test suite.
//
// Usage from a provider's test file:
//
//	func TestConformance(t *testing.T) {
//	    // Start your provider server on a random port.
//	    lis, _ := net.Listen("tcp", "localhost:0")
//	    srv := grpc.NewServer()
//	    providerv1.RegisterNicoProviderServer(srv, myProvider)
//	    go srv.Serve(lis)
//	    defer srv.Stop()
//
//	    conformance.Run(t, conformance.TestConfig{
//	        ServerAddr:   lis.Addr().String(),
//	        ExpectedName: "my-provider",
//	    })
//	}
package conformance
