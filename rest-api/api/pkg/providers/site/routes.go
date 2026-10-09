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

package site

import (
	"net/http"

	echo "github.com/labstack/echo/v4"

	apiHandler "github.com/NVIDIA/infra-controller/rest-api/api/pkg/api/handler"
)

// RegisterRoutes registers all site-related API routes on the given Echo group.
func (p *SiteProvider) RegisterRoutes(group *echo.Group) {
	prefix := ""

	// Site endpoints
	group.Add(http.MethodPost, prefix+"/site", apiHandler.NewCreateSiteHandler(p.dbSession, p.tc, p.tnc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site", apiHandler.NewGetAllSiteHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:id", apiHandler.NewGetSiteHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/site/:id", apiHandler.NewUpdateSiteHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/site/:id", apiHandler.NewDeleteSiteHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:id/status-history", apiHandler.NewGetSiteStatusDetailsHandler(p.dbSession).Handle)

	// ExpectedMachine endpoints
	group.Add(http.MethodPost, prefix+"/expected-machine", apiHandler.NewCreateExpectedMachineHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-machine", apiHandler.NewGetAllExpectedMachineHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-machine/:id", apiHandler.NewGetExpectedMachineHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/expected-machine/:id", apiHandler.NewUpdateExpectedMachineHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-machine/:id", apiHandler.NewDeleteExpectedMachineHandler(p.dbSession, p.scp, p.cfg).Handle)

	// ExpectedPowerShelf endpoints
	group.Add(http.MethodPost, prefix+"/expected-power-shelf", apiHandler.NewCreateExpectedPowerShelfHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-power-shelf", apiHandler.NewGetAllExpectedPowerShelfHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-power-shelf/:id", apiHandler.NewGetExpectedPowerShelfHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/expected-power-shelf/:id", apiHandler.NewUpdateExpectedPowerShelfHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-power-shelf/:id", apiHandler.NewDeleteExpectedPowerShelfHandler(p.dbSession, p.scp, p.cfg).Handle)

	// ExpectedSwitch endpoints
	group.Add(http.MethodPost, prefix+"/expected-switch", apiHandler.NewCreateExpectedSwitchHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-switch", apiHandler.NewGetAllExpectedSwitchHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-switch/:id", apiHandler.NewGetExpectedSwitchHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/expected-switch/:id", apiHandler.NewUpdateExpectedSwitchHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-switch/:id", apiHandler.NewDeleteExpectedSwitchHandler(p.dbSession, p.scp, p.cfg).Handle)

	// ExpectedMachine bulk/label operations
	group.Add(http.MethodPost, prefix+"/expected-machine/batch", apiHandler.NewCreateExpectedMachinesHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/expected-machine/batch", apiHandler.NewUpdateExpectedMachinesHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/expected-machine/all", apiHandler.NewReplaceAllExpectedMachinesHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-machine/all", apiHandler.NewDeleteAllExpectedMachinesHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-machine/label/key", apiHandler.NewGetAllExpectedMachineLabelKeyHandler(p.dbSession).Handle)
	group.Add(http.MethodGet, prefix+"/expected-machine/label/key/:key/value", apiHandler.NewGetAllExpectedMachineLabelValueHandler(p.dbSession).Handle)

	// ExpectedPowerShelf bulk operations
	group.Add(http.MethodPut, prefix+"/expected-power-shelf/all", apiHandler.NewReplaceAllExpectedPowerShelvesHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-power-shelf/all", apiHandler.NewDeleteAllExpectedPowerShelvesHandler(p.dbSession, p.scp, p.cfg).Handle)

	// ExpectedRack endpoints
	group.Add(http.MethodPost, prefix+"/expected-rack", apiHandler.NewCreateExpectedRackHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-rack", apiHandler.NewGetAllExpectedRackHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/expected-rack/all", apiHandler.NewReplaceAllExpectedRacksHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/expected-rack", apiHandler.NewReplaceAllExpectedRacksHandler(p.dbSession, p.scp, p.cfg).Handle) // deprecated compat
	group.Add(http.MethodDelete, prefix+"/expected-rack/all", apiHandler.NewDeleteAllExpectedRacksHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-rack/:id", apiHandler.NewGetExpectedRackHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/expected-rack/:id", apiHandler.NewUpdateExpectedRackHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-rack/:id", apiHandler.NewDeleteExpectedRackHandler(p.dbSession, p.scp, p.cfg).Handle)

	// ExpectedRackGroup endpoints
	group.Add(http.MethodPost, prefix+"/expected-rack-group", apiHandler.NewCreateExpectedRackGroupHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-rack-group", apiHandler.NewGetAllExpectedRackGroupHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/expected-rack-group/all", apiHandler.NewReplaceAllExpectedRackGroupsHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/expected-rack-group", apiHandler.NewReplaceAllExpectedRackGroupsHandler(p.dbSession, p.scp, p.cfg).Handle) // deprecated compat
	group.Add(http.MethodDelete, prefix+"/expected-rack-group/all", apiHandler.NewDeleteAllExpectedRackGroupsHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/expected-rack-group/:id", apiHandler.NewGetExpectedRackGroupHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/expected-rack-group/:id", apiHandler.NewUpdateExpectedRackGroupHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-rack-group/:id", apiHandler.NewDeleteExpectedRackGroupHandler(p.dbSession, p.scp, p.cfg).Handle)

	// ExpectedSwitch bulk operations
	group.Add(http.MethodPut, prefix+"/expected-switch/all", apiHandler.NewReplaceAllExpectedSwitchesHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/expected-switch/all", apiHandler.NewDeleteAllExpectedSwitchesHandler(p.dbSession, p.scp, p.cfg).Handle)

	// Credential management (Provider Admin)
	group.Add(http.MethodPut, prefix+"/credential/bmc", apiHandler.NewCreateOrUpdateBMCCredentialHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/credential/uefi", apiHandler.NewCreateUEFICredentialHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodPost, prefix+"/credential/rotation", apiHandler.NewRotateCredentialHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodGet, prefix+"/credential/rotation", apiHandler.NewGetCredentialRotationStatusHandler(p.dbSession, p.scp).Handle)

	// Site Explorer (Provider Admin)
	group.Add(http.MethodGet, prefix+"/site-explorer/endpoint", apiHandler.NewGetAllExploredEndpointHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/site-explorer/endpoint/action", apiHandler.NewSiteExplorerEndpointActionHandler(p.dbSession, p.scp, p.cfg).Handle)

	// Measured-boot trust approvals (Provider Admin)
	group.Add(http.MethodPost, prefix+"/measured-boot/trusted-machine", apiHandler.NewCreateMeasuredBootTrustedMachineHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodGet, prefix+"/measured-boot/trusted-machine", apiHandler.NewGetAllMeasuredBootTrustedMachineHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodDelete, prefix+"/measured-boot/trusted-machine/:id", apiHandler.NewDeleteMeasuredBootTrustedMachineHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodPost, prefix+"/measured-boot/trusted-profile", apiHandler.NewCreateMeasuredBootTrustedProfileHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodGet, prefix+"/measured-boot/trusted-profile", apiHandler.NewGetAllMeasuredBootTrustedProfileHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodDelete, prefix+"/measured-boot/trusted-profile/:id", apiHandler.NewDeleteMeasuredBootTrustedProfileHandler(p.dbSession, p.scp).Handle)

	// iPXE Template endpoints (read-only; templates synced from nico-core)
	group.Add(http.MethodGet, prefix+"/ipxe-template", apiHandler.NewGetAllIpxeTemplateHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/ipxe-template/:id", apiHandler.NewGetIpxeTemplateHandler(p.dbSession, p.tc, p.cfg).Handle)

	// Host Firmware Config (Provider Admin)
	group.Add(http.MethodPut, prefix+"/firmware-config/host", apiHandler.NewCreateOrUpdateHostFirmwareConfigHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodDelete, prefix+"/firmware-config/host", apiHandler.NewDeleteHostFirmwareConfigHandler(p.dbSession, p.scp).Handle)
}
