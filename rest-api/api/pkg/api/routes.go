// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	tClient "go.temporal.io/sdk/client"

	"github.com/NVIDIA/infra-controller/rest-api/api/internal/config"
	apiHandler "github.com/NVIDIA/infra-controller/rest-api/api/pkg/api/handler"
	dpsclient "github.com/NVIDIA/infra-controller/rest-api/api/pkg/client/dps"
	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	corev1 "github.com/NVIDIA/infra-controller/rest-api/proto/core/gen/v1"

	sc "github.com/NVIDIA/infra-controller/rest-api/api/pkg/client/site"
)

// NewAPIRoutes returns identity and admin routes served directly by the monolith.
// All domain routes (networking, compute, site) are registered by the respective
// in-tree provider's RegisterRoutes call instead.
func NewAPIRoutes(dbSession *cdb.Session, tc tClient.Client, tnc tClient.NamespaceClient, scp *sc.ClientPool, cfg *config.Config, dpsc dpsclient.PowerProvisioner) []Route {
	apiName := cfg.GetAPIName()

	apiPathPrefix := "/org/:orgName/" + apiName

	apiRoutes := []Route{
		// Metadata endpoint
		{
			Path:    apiPathPrefix + "/metadata",
			Method:  http.MethodGet,
			Handler: apiHandler.NewMetadataHandler(),
		},
		// BMC credential endpoint (Provider Admin).
		{
			Path:    apiPathPrefix + "/credential/bmc",
			Method:  http.MethodPut,
			Handler: apiHandler.NewCreateOrUpdateBMCCredentialHandler(dbSession, scp, cfg),
		},
		// Site Explorer explored endpoints (Provider Admin).
		{
			Path:    apiPathPrefix + "/site-explorer/endpoint",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllExploredEndpointHandler(dbSession, scp, cfg),
		},
		// Site Explorer endpoint actions (Provider Admin).
		{
			Path:    apiPathPrefix + "/site-explorer/endpoint/action",
			Method:  http.MethodPost,
			Handler: apiHandler.NewSiteExplorerEndpointActionHandler(dbSession, scp, cfg),
		},
		// Site-default UEFI credential endpoint (Provider Admin).
		{
			Path:    apiPathPrefix + "/credential/uefi",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateUEFICredentialHandler(dbSession, scp),
		},
		// Measured-boot machine and profile trust approvals (Provider Admin).
		{
			Path:    apiPathPrefix + "/measured-boot/trusted-machine",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateMeasuredBootTrustedMachineHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/measured-boot/trusted-machine",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllMeasuredBootTrustedMachineHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/measured-boot/trusted-machine/:id",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteMeasuredBootTrustedMachineHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/measured-boot/trusted-profile",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateMeasuredBootTrustedProfileHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/measured-boot/trusted-profile",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllMeasuredBootTrustedProfileHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/measured-boot/trusted-profile/:id",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteMeasuredBootTrustedProfileHandler(dbSession, scp),
		},
		// Site-wide credential rotation (Provider Admin).
		{
			Path:    apiPathPrefix + "/credential/rotation",
			Method:  http.MethodPost,
			Handler: apiHandler.NewRotateCredentialHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/credential/rotation",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCredentialRotationStatusHandler(dbSession, scp),
		},
		// User endpoint
		{
			Path:    apiPathPrefix + "/user/current",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetUserHandler(dbSession),
		},
		// Service Account endpoint
		{
			Path:    apiPathPrefix + "/service-account/current",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCurrentServiceAccountHandler(dbSession),
		},
		// Infrastructure Provider endpoints
		{
			Path:    apiPathPrefix + "/infrastructure-provider",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateInfrastructureProviderHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/infrastructure-provider/current",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCurrentInfrastructureProviderHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/infrastructure-provider/current",
			Method:  http.MethodPatch,
			Handler: apiHandler.NewUpdateCurrentInfrastructureProviderHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/infrastructure-provider/current/stats",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCurrentInfrastructureProviderStatsHandler(dbSession, tc, cfg),
		},
		// Tenant endpoints
		{
			Path:    apiPathPrefix + "/tenant",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateTenantHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/current",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCurrentTenantHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/current",
			Method:  http.MethodPatch,
			Handler: apiHandler.NewUpdateCurrentTenantHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/current/stats",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCurrentTenantStatsHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/current/routing-profile",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetCurrentTenantRoutingProfileHandler(dbSession, scp),
		},
		// Tenant Instance Type Stats endpoint
		{
			Path:    apiPathPrefix + "/tenant/instance-type/stats",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetTenantInstanceTypeStatsHandler(dbSession, cfg),
		},
		// TenantAccount endpoints
		{
			Path:    apiPathPrefix + "/tenant/account",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllTenantAccountHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/account/:id",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetTenantAccountHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/account",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateTenantAccountHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/account/:id",
			Method:  http.MethodPatch,
			Handler: apiHandler.NewUpdateTenantAccountHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/tenant/account/:id",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteTenantAccountHandler(dbSession, tc, cfg),
		},

		// SpectrumXPartition endpoints. No provider exists for SpectrumX yet;
		// these remain here until a spectrum-fabric provider is implemented.
		{
			Path:    apiPathPrefix + "/spectrumx-partition",
			Method:  http.MethodPost,
			Handler: apiHandler.NewCreateSpectrumXPartitionHandler(dbSession, scp, cfg),
		},
		{
			Path:    apiPathPrefix + "/spectrumx-partition",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllSpectrumXPartitionHandler(dbSession, cfg),
		},
		{
			Path:    apiPathPrefix + "/spectrumx-partition/:id",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetSpectrumXPartitionHandler(dbSession, cfg),
		},
		{
			Path:    apiPathPrefix + "/spectrumx-partition/:id",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteSpectrumXPartitionHandler(dbSession, scp, cfg),
		},

		// iPXE Template endpoints (read-only; templates are synced from nico-core)
		{
			Path:    apiPathPrefix + "/ipxe-template",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllIpxeTemplateHandler(dbSession, tc, cfg),
		},
		{
			Path:    apiPathPrefix + "/ipxe-template/:id",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetIpxeTemplateHandler(dbSession, tc, cfg),
		},

		// Audit Log endpoints
		{
			Path:    apiPathPrefix + "/audit",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAllAuditEntryHandler(dbSession),
		},
		{
			Path:    apiPathPrefix + "/audit/:id",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetAuditEntryHandler(dbSession),
		},

		// Tenant Identity endpoints
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/config",
			Method:  http.MethodPut,
			Handler: apiHandler.NewCreateOrUpdateTenantIdentityConfigHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/config",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetTenantIdentityConfigHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/config",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteTenantIdentityConfigHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/token-delegation",
			Method:  http.MethodPut,
			Handler: apiHandler.NewCreateOrUpdateTenantIdentityTokenDelegationHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/token-delegation",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetTenantIdentityTokenDelegationHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/token-delegation",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteTenantIdentityTokenDelegationHandler(dbSession, scp),
		},
		// Host Firmware Config endpoint (Provider Admin).
		{
			Path:    apiPathPrefix + "/firmware-config/host",
			Method:  http.MethodPut,
			Handler: apiHandler.NewCreateOrUpdateHostFirmwareConfigHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/firmware-config/host",
			Method:  http.MethodDelete,
			Handler: apiHandler.NewDeleteHostFirmwareConfigHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/tenant-identity/re-encrypt",
			Method:  http.MethodPost,
			Handler: apiHandler.NewReencryptTenantIdentitySecretsHandler(dbSession, scp),
		},
	}

	return apiRoutes
}

// NewWellKnownRoutes returns the public tenant-identity discovery routes.
// Registered before the auth middleware in server.go.
func NewWellKnownRoutes(dbSession *cdb.Session, scp *sc.ClientPool, cfg *config.Config) []Route {
	apiName := cfg.GetAPIName()
	apiPathPrefix := "/org/:orgName/" + apiName

	return []Route{
		{
			Path:    apiPathPrefix + "/site/:siteID/.well-known/jwks.json",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetJWKSHandler(dbSession, scp, corev1.JwksKind_Oidc),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/.well-known/openid-configuration",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetOpenIDConfigurationHandler(dbSession, scp),
		},
		{
			Path:    apiPathPrefix + "/site/:siteID/.well-known/spiffe/jwks.json",
			Method:  http.MethodGet,
			Handler: apiHandler.NewGetJWKSHandler(dbSession, scp, corev1.JwksKind_Spiffe),
		},
	}
}
