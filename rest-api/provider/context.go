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
	tsdkClient "go.temporal.io/sdk/client"

	echo "github.com/labstack/echo/v4"

	dpsclient "github.com/NVIDIA/infra-controller/rest-api/api/pkg/client/dps"
	sc "github.com/NVIDIA/infra-controller/rest-api/api/pkg/client/site"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
)

// ProviderContext is what the core provides to providers at initialization.
// Config is typed as interface{} because the concrete config type is in an
// internal package; providers type-assert to the config they need.
type ProviderContext struct {
	DB             *cdb.Session
	Temporal       tsdkClient.Client
	TemporalNS     tsdkClient.NamespaceClient
	SiteClientPool *sc.ClientPool
	Config         interface{}
	Registry       *Registry

	// APIPathPrefix is the route prefix for API endpoints (e.g., "/org/:orgName/carbide").
	// Set by the core before calling Init on API providers.
	APIPathPrefix string

	// TemporalNamespace is the Temporal namespace this worker operates in
	// ("cloud" or "site"). Providers use this to register the appropriate
	// workflows for the namespace.
	TemporalNamespace string

	// TemporalQueue is the Temporal task queue name.
	TemporalQueue string

	// WorkflowSiteClientPool is the workflow binary's site client pool.
	// Typed as interface{} because the API and workflow binaries use
	// different client pool types from different packages. Providers
	// type-assert to *workflow/pkg/client/site.ClientPool when registering
	// activities. Nil when running in the API binary.
	WorkflowSiteClientPool interface{}

	// ServiceEndpoints maps service names to gRPC addresses for
	// cross-domain access. Passed to external providers in InitRequest.
	ServiceEndpoints map[string]string

	// Hooks provides hook firing capabilities for activities. Created
	// by the main binary from the Registry and Temporal client. Nil
	// when running without hook support.
	Hooks *HookRunner

	// DPS is the hardware power provisioner, used by instance handlers
	// that delegate power operations to the DPS service.
	DPS dpsclient.PowerProvisioner

	// RouteGroup is the org-scoped Echo group where provider routes are
	// registered. Populated by InitAPIServer; used by KubernetesDiscovery
	// to register dynamically discovered providers' routes at runtime.
	RouteGroup *echo.Group
}
