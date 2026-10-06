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
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cdbm "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db/model"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/providers/compute/computesvc"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// computeServiceServer implements NicoComputeServiceServer by delegating
// to the computesvc.Service interface.
type computeServiceServer struct {
	providerv1.UnimplementedNicoComputeServiceServer
	svc computesvc.Service
}

func (s *computeServiceServer) GetInstanceByID(ctx context.Context, req *providerv1.GetInstanceByIDRequest) (*providerv1.InstanceResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid instance id: %v", err)
	}

	instance, err := s.svc.GetInstanceByID(ctx, nil, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "instance not found: %v", err)
	}

	return &providerv1.InstanceResponse{Instance: instanceToProto(instance)}, nil
}

func (s *computeServiceServer) GetMachineByID(ctx context.Context, req *providerv1.GetMachineByIDRequest) (*providerv1.MachineResponse, error) {
	machineID := req.GetId()
	if machineID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "machine id is required")
	}

	machine, err := s.svc.GetMachineByID(ctx, nil, machineID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "machine not found: %v", err)
	}

	return &providerv1.MachineResponse{Machine: machineToProto(machine)}, nil
}

func instanceToProto(i *cdbm.Instance) *providerv1.Instance {
	pb := &providerv1.Instance{
		Id:                       i.ID.String(),
		Name:                     i.Name,
		TenantId:                 i.TenantID.String(),
		InfrastructureProviderId: i.InfrastructureProviderID.String(),
		SiteId:                   i.SiteID.String(),
		VpcId:                    i.VpcID.String(),
		Status:                   i.Status,
		Labels:                   i.Labels,
	}
	if i.MachineID != nil {
		pb.MachineId = *i.MachineID
	}
	if i.Hostname != nil {
		pb.Hostname = *i.Hostname
	}
	return pb
}

func machineToProto(m *cdbm.Machine) *providerv1.Machine {
	pb := &providerv1.Machine{
		Id:                       m.ID,
		InfrastructureProviderId: m.InfrastructureProviderID.String(),
		SiteId:                   m.SiteID.String(),
		Status:                   m.Status,
		IsAssigned:               m.IsAssigned,
		Labels:                   m.Labels,
	}
	if m.InstanceTypeID != nil {
		pb.InstanceTypeId = m.InstanceTypeID.String()
	}
	if m.Hostname != nil {
		pb.Hostname = *m.Hostname
	}
	return pb
}
