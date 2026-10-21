// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

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
	switchID, err := uuid.Parse(req.GetSwitchId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid switch_id: %v", err)
	}

	// Resolve site_id from the expected switch record.
	esDAO := cdbm.NewExpectedSwitchDAO(s.db)
	sw, err := esDAO.Get(ctx, nil, switchID, nil, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "switch not found: %v", err)
	}

	// Query all VPCs for this site that have an active L3 VNI allocated.
	// These map to VRFs on the switch: one VRF per VPC.
	vpcDAO := cdbm.NewVpcDAO(s.db)
	vpcs, _, err := vpcDAO.GetAll(ctx, nil,
		cdbm.VpcFilterInput{SiteIDs: []uuid.UUID{sw.SiteID}},
		paginator.PageInput{}, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list VPCs: %v", err)
	}

	cfg := &providerv1.SwitchRoutingConfig{SwitchId: req.GetSwitchId()}

	for i := range vpcs {
		vpc := &vpcs[i]
		if vpc.ActiveVni == nil {
			continue // skip VPCs without an allocated VNI
		}

		vni := uint32(*vpc.ActiveVni)
		vrfName := "nico-" + vpc.Name // matches spectrum-fabric SyncVPCToFabric convention

		// Build route target strings (ASN:VNI format) from the effective
		// routing profile when available.
		var importRTs, exportRTs []string
		if p := vpc.EffectiveRoutingProfile; p != nil {
			for _, rt := range p.RouteTargetImports {
				importRTs = append(importRTs, fmt.Sprintf("%d:%d", rt.ASN, rt.VNI))
			}
			for _, rt := range p.RouteTargetsOnExports {
				exportRTs = append(exportRTs, fmt.Sprintf("%d:%d", rt.ASN, rt.VNI))
			}
		}

		cfg.Vrfs = append(cfg.Vrfs, &providerv1.Vrf{
			Name:               vrfName,
			L3Vni:              vni,
			ImportRouteTargets: importRTs,
			ExportRouteTargets: exportRTs,
		})

		cfg.L3Vnis = append(cfg.L3Vnis, &providerv1.L3Vni{
			Vni:              vni,
			Vrf:              vrfName,
			RouteTargets:     importRTs, // use import RTs as the combined RT list
		})
	}

	// BGP peers: query from switch_bgp_peers table.
	var bgpPeers []cdbm.SwitchBGPPeer
	if err := s.db.DB.NewSelect().
		Model(&bgpPeers).
		Where("expected_switch_id = ?", switchID).
		Scan(ctx); err != nil {
		// Non-fatal: log and continue with empty peer list.
		_ = err
	}
	for _, p := range bgpPeers {
		proto := &providerv1.BgpPeer{
			PeerIp:          p.PeerIP,
			PeerAsn:         uint32(p.PeerASN),
			Vrf:             p.VRF,
			Ttl:             uint32(p.TTL),
			EvpnEnabled:     p.EVPNEnabled,
		}
		if p.PeerGroup != nil {
			proto.PeerGroup = *p.PeerGroup
		}
		if p.SourceInterface != nil {
			proto.SourceInterface = *p.SourceInterface
		}
		if p.RouteRole != nil {
			proto.RouteRole = *p.RouteRole
		}
		cfg.BgpPeers = append(cfg.BgpPeers, proto)
	}

	// Build a VPC UUID → VRF name map for the L2 VNI lookup below.
	vpcToVRF := make(map[uuid.UUID]string, len(vpcs))
	for i := range vpcs {
		vpcToVRF[vpcs[i].ID] = "nico-" + vpcs[i].Name
	}

	// Populate L2 VNIs from subnets that have a VLAN ID assigned.
	subnetDAO := cdbm.NewSubnetDAO(s.db)
	subnets, _, err := subnetDAO.GetAll(ctx, nil,
		cdbm.SubnetFilterInput{SiteIDs: []uuid.UUID{sw.SiteID}},
		paginator.PageInput{}, nil)
	if err == nil {
		for i := range subnets {
			sn := &subnets[i]
			if sn.VlanID == nil {
				continue // VLAN not yet assigned
			}
			cfg.L2Vnis = append(cfg.L2Vnis, &providerv1.L2Vni{
				Vlan: uint32(*sn.VlanID),
				Vrf:  vpcToVRF[sn.VpcID], // empty string if VPC not in this site
			})
		}
	}

	// Populate EVPN fabric params from the switch record when present.
	if sw.EsiBaseMac != nil || sw.DfPreference != nil {
		evpn := &providerv1.EvpnConfig{}
		if sw.EsiBaseMac != nil {
			evpn.EsiBaseMac = *sw.EsiBaseMac
		}
		if sw.DfPreference != nil {
			evpn.DfPreference = uint32(*sw.DfPreference)
		}
		cfg.Evpn = evpn
	}

	// IS-IS interfaces and EVPN fabric_mac remain empty until the data model
	// is extended.

	return cfg, nil
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
