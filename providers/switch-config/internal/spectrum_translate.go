// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"fmt"

	"github.com/fabiendupont/nvidia-nvue-client-go/pkg/nvue"
	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// TranslateToNVUEPatches converts NICo switch state to a slice of NVUE PatchOps
// suitable for nvue.Client.ConfigureAndApply.
//
// The translation follows the same patterns as the spectrum-fabric provider:
//   - VRFs: /vrf/nico-{vpcID} with BGP + EVPN enabled
//   - L2 VNIs: /bridge/domain/br_default/vlan/{vid}/vni
//   - L3 VNIs: /vrf/nico-{vrfName}/evpn/vni
//   - Loopback: /interface/lo with management IP
//
// Fields marked [gap] in the proto (BGP peers, IS-IS, EVPN fabric params) are
// skipped until NICo's data model is extended to persist them.
func TranslateToNVUEPatches(sw *providerv1.NicoSwitch, routing *providerv1.SwitchRoutingConfig) []nvue.PatchOp {
	var patches []nvue.PatchOp

	// ── Loopback / management interface ──────────────────────────────────────
	if ip := sw.GetManagementIp(); ip != "" {
		patches = append(patches, nvue.PatchOp{
			Path: "/interface/lo",
			Payload: map[string]any{
				"ip": map[string]any{
					"address": map[string]any{
						ip + "/32": map[string]any{},
					},
				},
			},
		})
	}

	if routing == nil {
		return patches
	}

	// ── VRFs (from VPC routing profiles) ─────────────────────────────────────
	// Pattern mirrors spectrum-fabric.SyncVPCToFabric: /vrf/nico-{vpcID}
	for _, vrf := range routing.GetVrfs() {
		vrfPath := fmt.Sprintf("/vrf/%s", vrf.GetName())
		patches = append(patches, nvue.PatchOp{
			Path: vrfPath,
			Payload: map[string]any{
				"router": map[string]any{
					"bgp": map[string]any{
						"enable":    "on",
						"router-id": "auto",
						"address-family": map[string]any{
							"ipv4-unicast": map[string]any{
								"enable":       "on",
								"redistribute": map[string]any{"connected": map[string]any{"enable": "on"}},
								"route-export": map[string]any{"to-evpn": map[string]any{"enable": "on"}},
							},
							"l2vpn-evpn": map[string]any{"enable": "on"},
						},
					},
				},
				"evpn": map[string]any{"enable": "on"},
			},
		})

		// L3 VNI binding per VRF
		if vrf.GetL3Vni() != 0 {
			patches = append(patches, nvue.PatchOp{
				Path: fmt.Sprintf("%s/evpn/vni/%d", vrfPath, vrf.GetL3Vni()),
				Payload: map[string]any{
					"advertise": map[string]any{
						"ipv4": map[string]any{"unicast": map[string]any{"enable": "on"}},
					},
				},
			})
		}
	}

	// ── L2 VNIs (VLAN ↔ VNI, from subnet network_virtualization_type) ────────
	for _, l2 := range routing.GetL2Vnis() {
		vid := l2.GetVlan()
		vni := l2.GetVni()
		if vid == 0 || vni == 0 {
			continue
		}
		patches = append(patches,
			nvue.PatchOp{
				Path:    fmt.Sprintf("/bridge/domain/br_default/vlan/%d", vid),
				Payload: map[string]any{"vni": map[string]any{fmt.Sprintf("%d", vni): map[string]any{}}},
			},
			nvue.PatchOp{
				Path:    fmt.Sprintf("/interface/vlan%d", vid),
				Payload: map[string]any{"ip": map[string]any{"vrr": map[string]any{"enable": "on"}}},
			},
		)
	}

	// ── BGP peers [gap: not yet stored in NICo] ───────────────────────────────
	// Skipped — SwitchRoutingConfig.bgp_peers returns empty until NICo persists
	// BGP topology. The VRF patches above enable BGP; peer config follows.

	// ── EVPN fabric params [gap: not yet stored in NICo] ─────────────────────
	if evpn := routing.GetEvpn(); evpn.GetEsiBaseMac() != "" || evpn.GetFabricMac() != "" {
		patches = append(patches, nvue.PatchOp{
			Path:    "/evpn",
			Payload: map[string]any{"enable": "on"},
		})
	}

	return patches
}
