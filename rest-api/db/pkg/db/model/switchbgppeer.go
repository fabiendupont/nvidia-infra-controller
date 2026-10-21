// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// SwitchBGPPeer represents one BGP neighbor a fabric switch should establish.
// Typically used for leaf-spine underlay BGP and MLAG peer links.
type SwitchBGPPeer struct {
	bun.BaseModel `bun:"table:switch_bgp_peers"`

	ID               uuid.UUID `bun:"switch_bgp_peer_id,pk,type:uuid,default:gen_random_uuid()"`
	ExpectedSwitchID uuid.UUID `bun:"expected_switch_id,notnull"`
	PeerIP           string    `bun:"peer_ip,notnull"` // inet stored as string
	PeerASN          int32     `bun:"peer_asn,notnull"`
	VRF              string    `bun:"vrf,notnull,default:'default'"`
	PeerGroup        *string   `bun:"peer_group"`
	TTL              int32     `bun:"ttl,notnull,default:0"`
	SourceInterface  *string   `bun:"source_interface"`
	RouteRole        *string   `bun:"route_role"`
	EVPNEnabled      bool      `bun:"evpn_enabled,notnull,default:false"`
	BFDEnabled       bool      `bun:"bfd_enabled,notnull,default:false"`
	CreatedAt        time.Time `bun:"created_at,nullzero,default:now()"`
	UpdatedAt        time.Time `bun:"updated_at,nullzero,default:now()"`
}
