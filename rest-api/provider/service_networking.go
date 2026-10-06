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
	"github.com/NVIDIA/infra-controller/rest-api/db/pkg/db/paginator"
	"github.com/NVIDIA/infra-controller/rest-api/api/pkg/providers/networking/networkingsvc"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// networkingServiceServer implements NicoNetworkingServiceServer by delegating
// to the networkingsvc.Service interface.
type networkingServiceServer struct {
	providerv1.UnimplementedNicoNetworkingServiceServer
	svc networkingsvc.Service
}

func (s *networkingServiceServer) GetVpcByID(ctx context.Context, req *providerv1.GetVpcByIDRequest) (*providerv1.VpcResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vpc id: %v", err)
	}

	vpc, err := s.svc.GetVpcByID(ctx, nil, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "vpc not found: %v", err)
	}

	return &providerv1.VpcResponse{Vpc: vpcToProto(vpc)}, nil
}

func (s *networkingServiceServer) GetSubnets(ctx context.Context, req *providerv1.GetSubnetsRequest) (*providerv1.SubnetListResponse, error) {
	vpcID, err := uuid.Parse(req.GetVpcId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vpc id: %v", err)
	}

	filter := cdbm.SubnetFilterInput{VpcIDs: []uuid.UUID{vpcID}}
	subnets, total, err := s.svc.GetSubnets(ctx, nil, filter, paginator.PageInput{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get subnets: %v", err)
	}

	pbSubnets := make([]*providerv1.Subnet, len(subnets))
	for i := range subnets {
		pbSubnets[i] = subnetToProto(&subnets[i])
	}

	return &providerv1.SubnetListResponse{
		Subnets: pbSubnets,
		Total:   int32(total),
	}, nil
}

func vpcToProto(v *cdbm.Vpc) *providerv1.Vpc {
	pb := &providerv1.Vpc{
		Id:                       v.ID.String(),
		Name:                     v.Name,
		Org:                      v.Org,
		InfrastructureProviderId: v.InfrastructureProviderID.String(),
		TenantId:                 v.TenantID.String(),
		SiteId:                   v.SiteID.String(),
		Status:                   v.Status,
		Labels:                   v.Labels,
	}
	if v.NetworkVirtualizationType != nil {
		pb.NetworkVirtualizationType = *v.NetworkVirtualizationType
	}
	if v.RoutingProfile != nil {
		pb.RoutingProfile = *v.RoutingProfile
	}
	if v.NetworkSecurityGroupID != nil {
		pb.NetworkSecurityGroupId = *v.NetworkSecurityGroupID
	}
	return pb
}

func subnetToProto(su *cdbm.Subnet) *providerv1.Subnet {
	pb := &providerv1.Subnet{
		Id:           su.ID.String(),
		Name:         su.Name,
		Org:          su.Org,
		SiteId:       su.SiteID.String(),
		VpcId:        su.VpcID.String(),
		TenantId:     su.TenantID.String(),
		PrefixLength: int32(su.PrefixLength),
		Status:       su.Status,
	}
	if su.IPv4Prefix != nil {
		pb.Ipv4Prefix = *su.IPv4Prefix
	}
	if su.IPv4Gateway != nil {
		pb.Ipv4Gateway = *su.IPv4Gateway
	}
	if su.IPv6Prefix != nil {
		pb.Ipv6Prefix = *su.IPv6Prefix
	}
	if su.IPv6Gateway != nil {
		pb.Ipv6Gateway = *su.IPv6Gateway
	}
	return pb
}
