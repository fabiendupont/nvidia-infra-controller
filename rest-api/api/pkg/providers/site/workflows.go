// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package site

import (
	tsdkWorker "go.temporal.io/sdk/worker"

	sc "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/client/site"

	expectedMachineActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/expectedmachine"
	expectedPowerShelfActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/expectedpowershelf"
	expectedRackActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/expectedrack"
	expectedRackGroupActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/expectedrackgroup"
	expectedSwitchActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/expectedswitch"

	expectedMachineWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/expectedmachine"
	expectedPowerShelfWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/expectedpowershelf"
	expectedRackWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/expectedrack"
	expectedRackGroupWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/expectedrackgroup"
	expectedSwitchWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/expectedswitch"
)

const siteTaskQueue = "nico-site-task-queue"

func (p *SiteProvider) TaskQueue() string { return siteTaskQueue }

func (p *SiteProvider) RegisterWorkflows(w tsdkWorker.Worker) {
	w.RegisterWorkflow(expectedMachineWorkflow.UpdateExpectedMachineInventory)
	w.RegisterWorkflow(expectedPowerShelfWorkflow.UpdateExpectedPowerShelfInventory)
	w.RegisterWorkflow(expectedRackWorkflow.UpdateExpectedRackInventory)
	w.RegisterWorkflow(expectedRackGroupWorkflow.UpdateExpectedRackGroupInventory)
	w.RegisterWorkflow(expectedSwitchWorkflow.UpdateExpectedSwitchInventory)
}

func (p *SiteProvider) RegisterActivities(w tsdkWorker.Worker) {
	wfScp := p.workflowSiteClientPool.(*sc.ClientPool)

	expectedMachineManager := expectedMachineActivity.NewManageExpectedMachine(p.dbSession, wfScp)
	w.RegisterActivity(&expectedMachineManager)

	expectedPowerShelfManager := expectedPowerShelfActivity.NewManageExpectedPowerShelf(p.dbSession, wfScp)
	w.RegisterActivity(&expectedPowerShelfManager)

	expectedRackManager := expectedRackActivity.NewManageExpectedRack(p.dbSession, wfScp)
	w.RegisterActivity(&expectedRackManager)

	expectedRackGroupManager := expectedRackGroupActivity.NewManageExpectedRackGroup(p.dbSession, wfScp)
	w.RegisterActivity(&expectedRackGroupManager)

	expectedSwitchManager := expectedSwitchActivity.NewManageExpectedSwitch(p.dbSession, wfScp)
	w.RegisterActivity(&expectedSwitchManager)
}
