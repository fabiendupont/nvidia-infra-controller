// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cdb "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db"
	cdbm "github.com/NVIDIA/infra-controller/rest-api/db/pkg/db/model"
	"github.com/NVIDIA/infra-controller/rest-api/db/pkg/db/paginator"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// switchServiceServer implements NicoSwitchServiceServer by reading from
// NICo's expected_switch table. LLDP neighbor data is not currently stored
// in NICo; ReportObservedNeighbors is a no-op stub until Core adds a
// GetSwitchLLDPNeighbors RPC.
type switchServiceServer struct {
	providerv1.UnimplementedNicoSwitchServiceServer
	db *cdb.Session
}

func (s *switchServiceServer) ListSwitches(ctx context.Context, req *providerv1.ListSwitchesRequest) (*providerv1.ListSwitchesResponse, error) {
	siteID, err := uuid.Parse(req.GetSiteId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid site_id: %v", err)
	}

	dao := cdbm.NewExpectedSwitchDAO(s.db)
	filter := cdbm.ExpectedSwitchFilterInput{SiteIDs: []uuid.UUID{siteID}}
	switches, _, err := dao.GetAll(ctx, nil, filter, paginator.PageInput{}, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list switches: %v", err)
	}

	result := make([]*providerv1.NicoSwitch, len(switches))
	for i := range switches {
		result[i] = expectedSwitchToProto(&switches[i])
	}
	return &providerv1.ListSwitchesResponse{Switches: result, Total: int32(len(result))}, nil
}

func (s *switchServiceServer) GetSwitch(ctx context.Context, req *providerv1.GetSwitchRequest) (*providerv1.SwitchResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid switch_id: %v", err)
	}

	dao := cdbm.NewExpectedSwitchDAO(s.db)
	sw, err := dao.Get(ctx, nil, id, nil, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "switch not found: %v", err)
	}
	return &providerv1.SwitchResponse{Switch: expectedSwitchToProto(sw)}, nil
}

func (s *switchServiceServer) GetSwitchRoutingConfig(ctx context.Context, req *providerv1.GetSwitchRoutingConfigRequest) (*providerv1.SwitchRoutingConfig, error) {
	// Routing config (VRFs, BGP peers, EVPN) is not yet persisted in NICo's
	// data model. Return a minimal response with what is known.
	// TODO: populate from VPC/Subnet model once the mapping is defined.
	return &providerv1.SwitchRoutingConfig{SwitchId: req.GetSwitchId()}, nil
}

// ListExpectedNeighbors returns expected LLDP neighbors.
// NICo does not currently model per-port cabling topology; returns empty list.
// TODO: populate from a future cabling/topology table.
func (s *switchServiceServer) ListExpectedNeighbors(ctx context.Context, req *providerv1.ListExpectedNeighborsRequest) (*providerv1.ListExpectedNeighborsResponse, error) {
	return &providerv1.ListExpectedNeighborsResponse{}, nil
}

// ReportObservedNeighbors accepts LLDP-observed neighbor data from providers.
// NICo does not yet store switch LLDP data; the report is acknowledged and
// discarded. TODO: persist once Core adds GetSwitchLLDPNeighbors RPC.
func (s *switchServiceServer) ReportObservedNeighbors(ctx context.Context, req *providerv1.ReportObservedNeighborsRequest) (*providerv1.ReportObservedNeighborsResponse, error) {
	return &providerv1.ReportObservedNeighborsResponse{MismatchCount: 0, DiffSummary: "not yet implemented"}, nil
}

// expectedSwitchToProto converts a DB ExpectedSwitch to the provider proto.
func expectedSwitchToProto(es *cdbm.ExpectedSwitch) *providerv1.NicoSwitch {
	sw := &providerv1.NicoSwitch{
		Id:                 es.ID.String(),
		SiteId:             es.SiteID.String(),
		SerialNumber:       es.SwitchSerialNumber,
		BmcMac:             es.BmcMacAddress,
	}
	if es.Name != nil {
		sw.Name = *es.Name
	}
	if es.Model != nil {
		sw.Model = *es.Model
	}
	if es.Manufacturer != nil {
		sw.Manufacturer = *es.Manufacturer
	}
	if es.BmcIpAddress != nil {
		sw.BmcIp = *es.BmcIpAddress
	}
	if es.RackID != nil {
		sw.RackId = *es.RackID
	}
	return sw
}
