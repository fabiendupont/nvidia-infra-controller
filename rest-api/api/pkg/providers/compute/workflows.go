// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package compute

import (
	tsdkWorker "go.temporal.io/sdk/worker"

	sc "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/client/site"
	wfconfig "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/config"

	instanceActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/instance"
	instanceTypeActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/instancetype"
	ipxeTemplateActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/ipxetemplate"
	machineActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/machine"
	osImageActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/operatingsystem"
	skuActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/sku"
	sshKeyGroupActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/sshkeygroup"

	instanceWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/instance"
	instanceTypeWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/instancetype"
	ipxeTemplateWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/ipxetemplate"
	machineWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/machine"
	osImageWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/operatingsystem"
	skuWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/sku"
	sshKeyGroupWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/sshkeygroup"
)

const computeTaskQueue = "nico-compute-task-queue"

func (p *ComputeProvider) TaskQueue() string { return computeTaskQueue }

func (p *ComputeProvider) RegisterWorkflows(w tsdkWorker.Worker) {
	w.RegisterWorkflow(instanceWorkflow.DeleteInstanceByID)
	w.RegisterWorkflow(instanceWorkflow.RebootInstanceByID)
	w.RegisterWorkflow(instanceWorkflow.UpdateInstanceInventory)
	w.RegisterWorkflow(machineWorkflow.UpdateMachineInventory)
	w.RegisterWorkflow(sshKeyGroupWorkflow.SyncSSHKeyGroup)
	w.RegisterWorkflow(sshKeyGroupWorkflow.DeleteSSHKeyGroup)
	w.RegisterWorkflow(sshKeyGroupWorkflow.UpdateSSHKeyGroupInventory)
	w.RegisterWorkflow(instanceTypeWorkflow.UpdateInstanceTypeInventory)
	w.RegisterWorkflow(osImageWorkflow.UpdateOsImageInventory)
	w.RegisterWorkflow(osImageWorkflow.UpdateOperatingSystemInventory)
	w.RegisterWorkflow(skuWorkflow.UpdateSkuInventory)
	w.RegisterWorkflow(ipxeTemplateWorkflow.UpdateIpxeTemplateInventory)
}

func (p *ComputeProvider) RegisterActivities(w tsdkWorker.Worker) {
	wfScp := p.workflowSiteClientPool.(*sc.ClientPool)
	cfg := p.workflowConfig.(*wfconfig.Config)

	machineManager := machineActivity.NewManageMachine(p.dbSession, wfScp)
	w.RegisterActivity(&machineManager)

	instanceManager := instanceActivity.NewManageInstance(p.dbSession, wfScp, p.tc, cfg)
	w.RegisterActivity(&instanceManager)

	instanceTypeManager := instanceTypeActivity.NewManageInstanceType(p.dbSession, wfScp)
	w.RegisterActivity(&instanceTypeManager)

	sshKeyGroupManager := sshKeyGroupActivity.NewManageSSHKeyGroup(p.dbSession, wfScp)
	w.RegisterActivity(&sshKeyGroupManager)

	skuManager := skuActivity.NewManageSku(p.dbSession, wfScp)
	w.RegisterActivity(&skuManager)

	osImageManager := osImageActivity.NewManageOsImage(p.dbSession, wfScp)
	w.RegisterActivity(&osImageManager)

	ipxeTemplateManager := ipxeTemplateActivity.NewManageIpxeTemplate(p.dbSession, wfScp)
	w.RegisterActivity(&ipxeTemplateManager)
}
