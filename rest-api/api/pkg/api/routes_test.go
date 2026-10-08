// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"testing"

	"github.com/NVIDIA/infra-controller/rest-api/api/internal/config"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/api/handler/util/common"
	sc "github.com/NVIDIA/infra-controller/rest-api/api/pkg/client/site"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	"github.com/stretchr/testify/assert"

	temporalClient "go.temporal.io/sdk/client"
	tmocks "go.temporal.io/sdk/mocks"
)

// TestNewAPIRoutes guards the identity and admin routes served directly by the
// monolith API server. Domain routes (networking, compute, site) are registered
// by the respective in-tree providers and are not returned by NewAPIRoutes.
func TestNewAPIRoutes(t *testing.T) {
	type args struct {
		dbSession *cdb.Session
		tc        temporalClient.Client
		tnc       temporalClient.NamespaceClient
		scp       *sc.ClientPool
		cfg       *config.Config
	}

	tc := &tmocks.Client{}
	tnc := &tmocks.NamespaceClient{}

	cfg := common.GetTestConfig()
	tcfg, _ := cfg.GetTemporalConfig()
	scp := sc.NewClientPool(tcfg)

	// Only identity and admin routes remain in NewAPIRoutes. Domain routes
	// (vpc, subnet, machine, instance, expected-*, etc.) moved to providers.
	routeCount := map[string]int{
		"metadata":                  1, // GET /metadata
		"user":                      1, // GET /user/current
		"service-account":           1, // GET /service-account/current
		"infrastructure-provider":   4, // POST, GET /current, PATCH /current, GET /current/stats
		"tenant":                    6, // POST, GET+PATCH /current, GET /current/routing-profile, GET /current/stats, GET /instance-type/stats
		"tenant-account":            5, // POST, GET, GET/:id, PATCH/:id, DELETE/:id
		"identity-config":           3, // GET, POST, DELETE /site/:siteID/tenant-identity/config
		"identity-token-delegation": 3, // GET, POST, DELETE /site/:siteID/tenant-identity/token-delegation
		"identity-reencrypt":        1, // POST /site/:siteID/tenant-identity/re-encrypt
	}

	totalRouteCount := 0
	for _, v := range routeCount {
		totalRouteCount += v
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "test initializing API routes",
			args: args{
				dbSession: &cdb.Session{},
				tc:        tc,
				tnc:       tnc,
				scp:       scp,
				cfg:       cfg,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAPIRoutes(tt.args.dbSession, tt.args.tc, tt.args.tnc, tt.args.scp, tt.args.cfg, nil)

			assert.Equal(t, totalRouteCount, len(got))

			for _, route := range got {
				assert.Contains(t, route.Path, "/org/:orgName/"+cfg.GetAPIName())
			}

			// Verify the identity-reencrypt route is present — it guards the
			// tenant-identity re-encryption endpoint used during key rotation.
			reencryptPath := "/org/:orgName/" + cfg.GetAPIName() + "/site/:siteID/tenant-identity/re-encrypt"
			assertRouteExists(t, got, http.MethodPost, reencryptPath)

			// Verify tenant routing-profile route is present (used by networking layer).
			tenantPath := "/org/:orgName/" + cfg.GetAPIName() + "/tenant"
			assertRouteExists(t, got, http.MethodGet, tenantPath+"/current/routing-profile")
		})
	}
}

func assertRouteExists(t *testing.T, routes []Route, method, path string) {
	t.Helper()

	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return
		}
	}

	assert.Failf(t, "route not found", "missing %s %s", method, path)
}

func assertRouteBefore(t *testing.T, routes []Route, firstMethod, firstPath, secondMethod, secondPath string) {
	t.Helper()

	firstIndex := -1
	secondIndex := -1
	for i, route := range routes {
		if route.Method == firstMethod && route.Path == firstPath {
			firstIndex = i
		}
		if route.Method == secondMethod && route.Path == secondPath {
			secondIndex = i
		}
	}

	assert.NotEqual(t, -1, firstIndex, "missing %s %s", firstMethod, firstPath)
	assert.NotEqual(t, -1, secondIndex, "missing %s %s", secondMethod, secondPath)
	assert.Less(t, firstIndex, secondIndex, "%s %s must be registered before %s %s", firstMethod, firstPath, secondMethod, secondPath)
}

// TestNewWellKnownRoutes guards the unauthenticated .well-known/* surface
// returned by NewWellKnownRoutes. These routes are mounted on the root echo
// (before the versioned auth middleware in server.go) so that JWT verifiers
// without credentials can fetch JWKS / OIDC discovery; any drift in the count
// or path shape of this set is security-relevant and must fail loudly.
func TestNewWellKnownRoutes(t *testing.T) {
	cfg := common.GetTestConfig()
	tcfg, _ := cfg.GetTemporalConfig()
	scp := sc.NewClientPool(tcfg)

	got := NewWellKnownRoutes(&cdb.Session{}, scp, cfg)

	wantPaths := map[string]string{
		"/org/:orgName/" + cfg.GetAPIName() + "/site/:siteID/.well-known/jwks.json":            "GET",
		"/org/:orgName/" + cfg.GetAPIName() + "/site/:siteID/.well-known/openid-configuration": "GET",
		"/org/:orgName/" + cfg.GetAPIName() + "/site/:siteID/.well-known/spiffe/jwks.json":     "GET",
	}

	assert.Equal(t, len(wantPaths), len(got))

	gotByPath := make(map[string]string, len(got))
	for _, r := range got {
		gotByPath[r.Path] = r.Method
	}
	for path, method := range wantPaths {
		assert.Equal(t, method, gotByPath[path], "well-known route %s missing or wrong method", path)
	}
}
