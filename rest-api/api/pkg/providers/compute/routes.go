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

package compute

import (
	"net/http"

	echo "github.com/labstack/echo/v4"

	apiHandler "github.com/NVIDIA/infra-controller/rest-api/api/pkg/api/handler"
)

// RegisterRoutes registers all compute-related API routes.
func (p *ComputeProvider) RegisterRoutes(group *echo.Group) {
	prefix := ""

	// Instance endpoints
	group.Add(http.MethodPost, prefix+"/instance", apiHandler.NewCreateInstanceHandler(p.dbSession, p.tc, p.scp, p.cfg, p.dps).WithHooks(p.hooks).Handle)
	group.Add(http.MethodPost, prefix+"/instance/batch", apiHandler.NewBatchCreateInstanceHandler(p.dbSession, p.tc, p.scp, p.cfg, p.dps).Handle)
	group.Add(http.MethodGet, prefix+"/instance", apiHandler.NewGetAllInstanceHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/instance/:id", apiHandler.NewGetInstanceHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/instance/:id", apiHandler.NewUpdateInstanceHandler(p.dbSession, p.tc, p.scp, p.cfg, p.dps).Handle)
	group.Add(http.MethodDelete, prefix+"/instance/:id", apiHandler.NewDeleteInstanceHandler(p.dbSession, p.tc, p.scp, p.cfg, p.dps).Handle)
	group.Add(http.MethodGet, prefix+"/instance/:id/status-history", apiHandler.NewGetInstanceStatusDetailsHandler(p.dbSession).Handle)

	// Instance Type endpoints
	group.Add(http.MethodPost, prefix+"/instance/type", apiHandler.NewCreateInstanceTypeHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/instance/type", apiHandler.NewGetAllInstanceTypeHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/instance/type/:id", apiHandler.NewGetInstanceTypeHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/instance/type/:id", apiHandler.NewUpdateInstanceTypeHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/instance/type/:id", apiHandler.NewDeleteInstanceTypeHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Interface endpoints under instance
	group.Add(http.MethodGet, prefix+"/instance/:instanceId/interface", apiHandler.NewGetAllInterfaceHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/instance/:instanceId/infiniband-interface", apiHandler.NewGetAllInstanceInfiniBandInterfaceHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/instance/:instanceId/nvlink-interface", apiHandler.NewGetAllInstanceNVLinkInterfaceHandler(p.dbSession, p.tc, p.cfg).Handle)

	// Machine endpoints
	group.Add(http.MethodGet, prefix+"/machine", apiHandler.NewGetAllMachineHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/machine/:id", apiHandler.NewGetMachineHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/machine/:id", apiHandler.NewUpdateMachineHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/machine/:id", apiHandler.NewDeleteMachineHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/machine/:id/status-history", apiHandler.NewGetMachineStatusDetailsHandler(p.dbSession).Handle)
	group.Add(http.MethodGet, prefix+"/machine/gpu/stats", apiHandler.NewGetMachineGPUStatsHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/machine/instance-type/stats/summary", apiHandler.NewGetMachineInstanceTypeSummaryHandler(p.dbSession, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/machine/instance-type/stats", apiHandler.NewGetMachineInstanceTypeStatsHandler(p.dbSession, p.cfg).Handle)

	// Machine/Instance Type association endpoints
	group.Add(http.MethodPost, prefix+"/instance/type/:instanceTypeId/machine", apiHandler.NewCreateMachineInstanceTypeHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/instance/type/:instanceTypeId/machine", apiHandler.NewGetAllMachineInstanceTypeHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/instance/type/:instanceTypeId/machine/:id", apiHandler.NewDeleteMachineInstanceTypeHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Allocation endpoints
	group.Add(http.MethodPost, prefix+"/allocation", apiHandler.NewCreateAllocationHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/allocation", apiHandler.NewGetAllAllocationHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/allocation/:id", apiHandler.NewGetAllocationHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/allocation/:id", apiHandler.NewUpdateAllocationHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/allocation/:id", apiHandler.NewDeleteAllocationHandler(p.dbSession, p.tc, p.cfg).Handle)

	// AllocationConstraint update endpoint
	group.Add(http.MethodPatch, prefix+"/allocation/:allocationId/constraint/:id", apiHandler.NewUpdateAllocationConstraintHandler(p.dbSession, p.tc, p.cfg).Handle)

	// OperatingSystem endpoints
	group.Add(http.MethodPost, prefix+"/operating-system", apiHandler.NewCreateOperatingSystemHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/operating-system", apiHandler.NewGetAllOperatingSystemHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/operating-system/:id", apiHandler.NewGetOperatingSystemHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/operating-system/:id", apiHandler.NewUpdateOperatingSystemHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/operating-system/:id", apiHandler.NewDeleteOperatingSystemHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// SSHKey endpoints
	group.Add(http.MethodPost, prefix+"/sshkey", apiHandler.NewCreateSSHKeyHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/sshkey", apiHandler.NewGetAllSSHKeyHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/sshkey/:id", apiHandler.NewGetSSHKeyHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/sshkey/:id", apiHandler.NewUpdateSSHKeyHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/sshkey/:id", apiHandler.NewDeleteSSHKeyHandler(p.dbSession, p.tc, p.cfg).Handle)

	// SSHKeyGroup endpoints
	group.Add(http.MethodPost, prefix+"/sshkeygroup", apiHandler.NewCreateSSHKeyGroupHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/sshkeygroup", apiHandler.NewGetAllSSHKeyGroupHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/sshkeygroup/:id", apiHandler.NewGetSSHKeyGroupHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/sshkeygroup/:id", apiHandler.NewUpdateSSHKeyGroupHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/sshkeygroup/:id", apiHandler.NewDeleteSSHKeyGroupHandler(p.dbSession, p.tc, p.cfg).Handle)

	// Machine Capability endpoints
	group.Add(http.MethodGet, prefix+"/machine-capability", apiHandler.NewGetAllMachineCapabilityHandler(p.dbSession).Handle)

	// Machine Validation endpoints
	group.Add(http.MethodPost, prefix+"/site/:siteID/machine-validation/test", apiHandler.NewCreateMachineValidationTestHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/site/:siteID/machine-validation/test/:id/version/:version", apiHandler.NewUpdateMachineValidationTestHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:siteID/machine-validation/test", apiHandler.NewGetAllMachineValidationTestHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:siteID/machine-validation/test/:id/version/:version", apiHandler.NewGetMachineValidationTestHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:siteID/machine-validation/machine/:machineID/results", apiHandler.NewGetMachineValidationResultsHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:siteID/machine-validation/machine/:machineID/runs", apiHandler.NewGetAllMachineValidationRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:siteID/machine-validation/external-config", apiHandler.NewGetAllMachineValidationExternalConfigHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/site/:siteID/machine-validation/external-config/:cfgName", apiHandler.NewGetMachineValidationExternalConfigHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/site/:siteID/machine-validation/external-config", apiHandler.NewCreateMachineValidationExternalConfigHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/site/:siteID/machine-validation/external-config/:cfgName", apiHandler.NewUpdateMachineValidationExternalConfigHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/site/:siteID/machine-validation/external-config/:cfgName", apiHandler.NewDeleteMachineValidationExternalConfigHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// SKU endpoints
	group.Add(http.MethodGet, prefix+"/sku", apiHandler.NewGetAllSkuHandler(p.dbSession, p.tc, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/sku/:id", apiHandler.NewGetSkuHandler(p.dbSession, p.tc, p.cfg).Handle)

	// Rack endpoints (RLA)
	group.Add(http.MethodGet, prefix+"/rack/task/:id", apiHandler.NewGetTaskHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/rack", apiHandler.NewGetAllRackHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/rack/validation", apiHandler.NewValidateRacksHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/rack/power", apiHandler.NewBatchUpdateRackPowerStateHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/rack/firmware", apiHandler.NewBatchUpdateRackFirmwareHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/rack/bringup", apiHandler.NewBatchBringUpRackHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/rack/:id", apiHandler.NewGetRackHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/rack/:id/validation", apiHandler.NewValidateRackHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/rack/:id/power", apiHandler.NewUpdateRackPowerStateHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/rack/:id/firmware", apiHandler.NewUpdateRackFirmwareHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/rack/:id/bringup", apiHandler.NewBringUpRackHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Tray endpoints (RLA)
	group.Add(http.MethodGet, prefix+"/tray", apiHandler.NewGetAllTrayHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/tray/power", apiHandler.NewBatchUpdateTrayPowerStateHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/tray/firmware", apiHandler.NewBatchUpdateTrayFirmwareHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/tray/validation", apiHandler.NewValidateTraysHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/tray/:id", apiHandler.NewGetTrayHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/tray/:id/power", apiHandler.NewUpdateTrayPowerStateHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/tray/:id/firmware", apiHandler.NewUpdateTrayFirmwareHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/tray/:id/validation", apiHandler.NewValidateTrayHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// DPU machine endpoints
	group.Add(http.MethodGet, prefix+"/dpu", apiHandler.NewGetAllDpuMachinesHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodGet, prefix+"/dpu/:id", apiHandler.NewGetDpuMachineHandler(p.dbSession, p.scp).Handle)

	// Machine label and sub-resource endpoints
	group.Add(http.MethodGet, prefix+"/machine/label/key", apiHandler.NewGetAllMachineLabelKeyHandler(p.dbSession).Handle)
	group.Add(http.MethodGet, prefix+"/machine/label/key/:key/value", apiHandler.NewGetAllMachineLabelValueHandler(p.dbSession).Handle)
	group.Add(http.MethodGet, prefix+"/machine/:id/dpu", apiHandler.NewGetAllDpuMachineHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodPatch, prefix+"/machine/:id/dpu/reprovision", apiHandler.NewReprovisionMachineDpuHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/machine/:id/bmc/reset", apiHandler.NewResetMachineBMCHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/machine/:id/chassis/:chassisId/reset", apiHandler.NewResetMachineChassisHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodGet, prefix+"/machine/:id/health-report", apiHandler.NewGetAllMachineHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/machine/:id/health-report", apiHandler.NewCreateOrUpdateMachineHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/machine/:id/health-report/:source", apiHandler.NewDeleteMachineHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/machine/:id/power", apiHandler.NewMachinePowerControlHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/machine/:id/decommission", apiHandler.NewDecommissionMachineHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/machine/:id/validation/run", apiHandler.NewCreateMachineValidationRunHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/machine/:id/validation/run", apiHandler.NewGetAllMachineValidationRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/machine/:id/validation/result", apiHandler.NewGetMachineValidationResultsHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// SKU admin endpoints
	group.Add(http.MethodPost, prefix+"/sku", apiHandler.NewCreateSkuHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodPatch, prefix+"/sku/:id", apiHandler.NewUpdateSkuHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodDelete, prefix+"/sku/:id", apiHandler.NewDeleteSkuHandler(p.dbSession, p.scp).Handle)

	// Task endpoints
	group.Add(http.MethodPost, prefix+"/rack/task/:id/cancel", apiHandler.NewCancelTaskHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/task", apiHandler.NewGetAllTaskHandler(p.dbSession, p.scp).Handle)
	group.Add(http.MethodGet, prefix+"/task/:id", apiHandler.NewGetTaskHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/task/:id/cancel", apiHandler.NewCancelTaskHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Task Rule endpoints
	group.Add(http.MethodPost, prefix+"/task/rule", apiHandler.NewCreateTaskRuleHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/task/rule", apiHandler.NewGetAllTaskRuleHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/task/rule/:id", apiHandler.NewGetTaskRuleHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPatch, prefix+"/task/rule/:id", apiHandler.NewUpdateTaskRuleHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/task/rule/:id", apiHandler.NewDeleteTaskRuleHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Task Run endpoints
	group.Add(http.MethodPost, prefix+"/task/run", apiHandler.NewCreateTaskRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/task/run", apiHandler.NewGetAllTaskRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/task/run/:id", apiHandler.NewGetTaskRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/task/run/:id/target", apiHandler.NewGetAllTaskRunTargetHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/task/run/:id/pause", apiHandler.NewPauseTaskRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/task/run/:id/resume", apiHandler.NewResumeTaskRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/task/run/:id/advance", apiHandler.NewAdvanceTaskRunPhaseHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)
	group.Add(http.MethodPost, prefix+"/task/run/:id/cancel", apiHandler.NewCancelTaskRunHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Rack health report and task sub-routes
	group.Add(http.MethodGet, prefix+"/rack/:id/health-report", apiHandler.NewGetAllRackHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/rack/:id/health-report", apiHandler.NewCreateOrUpdateRackHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/rack/:id/health-report/:source", apiHandler.NewDeleteRackHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/rack/:id/task", apiHandler.NewGetRackTasksHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Tray health report and task sub-routes
	group.Add(http.MethodGet, prefix+"/tray/:id/health-report", apiHandler.NewGetAllTrayHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodPut, prefix+"/tray/:id/health-report", apiHandler.NewCreateOrUpdateTrayHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodDelete, prefix+"/tray/:id/health-report/:source", apiHandler.NewDeleteTrayHealthReportHandler(p.dbSession, p.scp, p.cfg).Handle)
	group.Add(http.MethodGet, prefix+"/tray/:id/task", apiHandler.NewGetTrayTasksHandler(p.dbSession, p.tc, p.scp, p.cfg).Handle)

	// Audit Log endpoints
	group.Add(http.MethodGet, prefix+"/audit", apiHandler.NewGetAllAuditEntryHandler(p.dbSession).Handle)
	group.Add(http.MethodGet, prefix+"/audit/:id", apiHandler.NewGetAuditEntryHandler(p.dbSession).Handle)
}
