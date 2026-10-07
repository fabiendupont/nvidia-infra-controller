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
	"net/http"

	"github.com/labstack/echo/v4"
)

// stubRoutes maps feature names to route prefix patterns that should return
// 501 when no provider is registered. Patterns use Echo wildcard syntax.
// Routes that remain in NewAPIRoutes() (e.g. /rack/:id/health-report) are
// registered with higher precedence than the wildcards below, so they
// continue to be served by the monolith handler even when the provider
// is absent.
var stubRoutes = map[string][]string{
	// Core infrastructure providers — registered by in-tree providers;
	// 501 when provider registry is empty (no providers initialized).
	"networking": {
		"/vpc", "/vpc/*",
		"/vpc-prefix", "/vpc-prefix/*",
		"/ipblock", "/ipblock/*",
		"/subnet", "/subnet/*",
		"/network-security-group", "/network-security-group/*",
		"/infiniband-interface", "/infiniband-partition", "/infiniband-partition/*",
		"/nvlink-interface", "/nvlink-logical-partition", "/nvlink-logical-partition/*",
		"/dpu-extension-service", "/dpu-extension-service/*",
		"/vpc-peering", "/vpc-peering/*",
		"/domain/nvlink", "/domain/nvlink/*",
	},
	"compute": {
		"/instance", "/instance/*",
		"/machine", "/machine/:id", "/machine/:id/status-history",
		"/machine/gpu/stats", "/machine/instance-type/stats", "/machine/instance-type/stats/summary",
		"/machine/label/key", "/machine/label/key/*",
		"/machine/:id/dpu", "/machine/:id/dpu/*",
		"/machine/:id/bmc/reset", "/machine/:id/chassis/*",
		"/machine/:id/health-report", "/machine/:id/health-report/*",
		"/machine/:id/power", "/machine/:id/decommission",
		"/machine/:id/validation/*",
		"/allocation", "/allocation/*",
		"/operating-system", "/operating-system/*",
		"/sshkey", "/sshkey/*",
		"/sshkeygroup", "/sshkeygroup/*",
		"/machine-capability",
		"/site/:siteID/machine-validation/*",
		"/dpu", "/dpu/:id",
		"/rack", "/rack/validation", "/rack/power", "/rack/firmware", "/rack/bringup",
		"/rack/:id", "/rack/:id/validation", "/rack/:id/power", "/rack/:id/firmware", "/rack/:id/bringup",
		"/rack/:id/health-report", "/rack/:id/health-report/*", "/rack/:id/task",
		"/rack/task/:id", "/rack/task/:id/cancel",
		"/tray", "/tray/validation", "/tray/power", "/tray/firmware",
		"/tray/:id", "/tray/:id/validation", "/tray/:id/power", "/tray/:id/firmware",
		"/tray/:id/health-report", "/tray/:id/health-report/*", "/tray/:id/task",
		"/sku", "/sku/:id",
		"/task", "/task/*",
	},
	"site": {
		"/site", "/site/:id", "/site/:id/status-history",
		"/expected-machine", "/expected-machine/:id",
		"/expected-machine/batch", "/expected-machine/all",
		"/expected-machine/label/key", "/expected-machine/label/key/*",
		"/expected-power-shelf", "/expected-power-shelf/:id",
		"/expected-power-shelf/all",
		"/expected-rack", "/expected-rack/:id", "/expected-rack/all",
		"/expected-rack-group", "/expected-rack-group/:id", "/expected-rack-group/all",
		"/expected-switch", "/expected-switch/:id",
		"/expected-switch/all",
	},
	// Extended features — no monolith fallback.
	"catalog":     {"/catalog/*"},
	"fulfillment": {"/catalog/orders/*", "/services/*"},
	"showback":    {"/self/usage", "/self/quotas", "/services/:id/usage"},
	"storage":     {"/:orgName/*/storage/*"},
	"dcim":        {"/dcim/*"},
	"dpf-hcp":     {"/sites/:siteId/dpf-hcp", "/sites/:siteId/dpf-hcp/status"},
}

// RegisterStubs registers 501 handlers for features that have no provider.
func RegisterStubs(group *echo.Group, registry *Registry) {
	for feature, routes := range stubRoutes {
		if _, ok := registry.FeatureProvider(feature); ok {
			continue
		}
		for _, route := range routes {
			f := feature // capture for closure
			handler := func(c echo.Context) error {
				return c.JSON(http.StatusNotImplemented, map[string]string{
					"error":   "not_implemented",
					"feature": f,
					"message": fmt.Sprintf("Feature '%s' has no provider configured.", f),
				})
			}
			group.Any(route, handler)
		}
	}
}
