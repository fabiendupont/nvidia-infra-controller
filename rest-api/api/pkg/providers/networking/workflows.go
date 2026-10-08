// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package networking

import (
	tsdkWorker "go.temporal.io/sdk/worker"

	dpuExtensionServiceActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/dpuextensionservice"
	ibpActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/infinibandpartition"
	networkSecurityGroupActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/networksecuritygroup"
	nvLinkLogicalPartitionActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/nvlinklogicalpartition"
	sxpActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/spectrumxpartition"
	subnetActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/subnet"
	vpcActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/vpc"
	vpcPeeringActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/vpcpeering"
	vpcPrefixActivity "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/activity/vpcprefix"

	dpuExtensionServiceWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/dpuextensionservice"
	ibpWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/infinibandpartition"
	networkSecurityGroupWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/networksecuritygroup"
	nvLinkLogicalPartitionWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/nvlinklogicalpartition"
	sxpWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/spectrumxpartition"
	subnetWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/subnet"
	vpcWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/vpc"
	vpcPeeringWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/vpcpeering"
	vpcPrefixWorkflow "github.com/NVIDIA/infra-controller/rest-api/workflow/pkg/workflow/vpcprefix"
)

const networkingTaskQueue = "nico-networking-task-queue"

func (p *NetworkingProvider) TaskQueue() string { return networkingTaskQueue }

func (p *NetworkingProvider) RegisterWorkflows(w tsdkWorker.Worker) {
	w.RegisterWorkflow(vpcWorkflow.DeleteVpcByID)
	w.RegisterWorkflow(subnetWorkflow.DeleteSubnetByID)
	w.RegisterWorkflow(ibpWorkflow.DeleteInfiniBandPartitionByID)
	w.RegisterWorkflow(vpcWorkflow.UpdateVpcInventory)
	w.RegisterWorkflow(subnetWorkflow.UpdateSubnetInventory)
	w.RegisterWorkflow(ibpWorkflow.UpdateInfiniBandPartitionInventory)
	w.RegisterWorkflow(sxpWorkflow.UpdateSpectrumXPartitionInventory)
	w.RegisterWorkflow(networkSecurityGroupWorkflow.UpdateNetworkSecurityGroupInventory)
	w.RegisterWorkflow(vpcPrefixWorkflow.UpdateVpcPrefixInventory)
	w.RegisterWorkflow(vpcPeeringWorkflow.UpdateVpcPeeringInventory)
	w.RegisterWorkflow(dpuExtensionServiceWorkflow.UpdateDpuExtensionServiceInventory)
	w.RegisterWorkflow(nvLinkLogicalPartitionWorkflow.UpdateNVLinkLogicalPartitionInventory)
}

func (p *NetworkingProvider) RegisterActivities(w tsdkWorker.Worker) {
	// p.workflowSiteClientPool is typed as *workflow/pkg/client/site.ClientPool
	wfScp := p.workflowSiteClientPool

	vpcManager := vpcActivity.NewManageVpc(p.dbSession, wfScp, p.tc)
	w.RegisterActivity(&vpcManager)

	subnetManager := subnetActivity.NewManageSubnet(p.dbSession, wfScp, p.tc)
	w.RegisterActivity(&subnetManager)

	ibpManager := ibpActivity.NewManageInfiniBandPartition(p.dbSession, wfScp)
	w.RegisterActivity(&ibpManager)

	sxpManager := sxpActivity.NewManageSpectrumXPartition(p.dbSession, wfScp)
	w.RegisterActivity(&sxpManager)

	nsgManager := networkSecurityGroupActivity.NewManageNetworkSecurityGroup(p.dbSession, wfScp)
	w.RegisterActivity(&nsgManager)

	vpcPrefixManager := vpcPrefixActivity.NewManageVpcPrefix(p.dbSession, wfScp)
	w.RegisterActivity(&vpcPrefixManager)

	vpcPeeringManager := vpcPeeringActivity.NewManageVpcPeering(p.dbSession, wfScp)
	w.RegisterActivity(&vpcPeeringManager)

	dpuManager := dpuExtensionServiceActivity.NewManageDpuExtensionService(p.dbSession, wfScp)
	w.RegisterActivity(&dpuManager)

	nvLinkManager := nvLinkLogicalPartitionActivity.NewManageNVLinkLogicalPartition(p.dbSession, wfScp)
	w.RegisterActivity(&nvLinkManager)
}
