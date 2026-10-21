// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fabiendupont/nvidia-nvue-client-go/pkg/nvue"
	"github.com/rs/zerolog/log"
)

// SpectrumConfigurator applies switch configuration to NVIDIA Spectrum switches
// (NVOS or Cumulus Linux 5.x) via the NVUE REST API.
//
// It uses github.com/fabiendupont/nvidia-nvue-client-go — the same client used
// by the spectrum-fabric provider — which implements the NVUE revision workflow:
// create revision → PATCH desired state → apply → poll until applied.
//
// Credential resolution order:
//  1. NicoSwitch.credential_ref looked up via Vault (not yet implemented; TODO).
//  2. SWITCH_USERNAME / SWITCH_PASSWORD environment variables (dev fallback).
//  3. opts.SwitchUsername / opts.SwitchPassword (programmatic override).
type SpectrumConfigurator struct {
	opts ConfiguratorOptions
}

// NewSpectrumConfigurator creates a SpectrumConfigurator.
func (s *SpectrumConfigurator) ConfiguratorProvider() string { return "spectrum" }

func NewSpectrumConfigurator(opts ConfiguratorOptions) (*SpectrumConfigurator, error) {
	return &SpectrumConfigurator{opts: opts}, nil
}

// Push reads desired state from NicoSwitchService, translates it to NVUE
// PatchOps, and applies them via ConfigureAndApply. Stores the resulting NVUE
// JSON in the config store for versioning and deduplication.
func (s *SpectrumConfigurator) Push(ctx context.Context, switchID string) (*ConfigRecord, error) {
	if s.opts.SwitchService == nil {
		return nil, fmt.Errorf("switch service not available")
	}

	sw, err := s.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return nil, fmt.Errorf("switch %s has no management_ip (data model gap — not yet stored in NICo)", switchID)
	}

	routing, err := s.opts.SwitchService.GetRoutingConfig(ctx, switchID)
	if err != nil {
		// Routing config gap: proceed with whatever is available.
		log.Warn().Err(err).Str("switch_id", switchID).Msg("routing config unavailable, pushing partial state")
		routing = nil
	}

	patches := TranslateToNVUEPatches(sw, routing)
	if len(patches) == 0 {
		return nil, fmt.Errorf("no NVUE patches generated for switch %s (check data model gaps)", switchID)
	}

	username, password := s.resolveCredentials(sw.GetCredentialRef())
	client := nvue.NewClient(sw.GetManagementIp(), username, password,
		nvue.WithInsecureSkipVerify())

	log.Info().Str("switch_id", switchID).Str("management_ip", sw.GetManagementIp()).
		Int("patch_count", len(patches)).Msg("applying NVUE config to Spectrum switch")

	if _, err := client.ConfigureAndApply(ctx, patches); err != nil {
		return nil, fmt.Errorf("NVUE ConfigureAndApply switch %s: %w", switchID, err)
	}

	if s.opts.ConfigStore == nil {
		return nil, nil
	}
	// Store the applied patches as NVUE JSON for versioning.
	payload := patchesToMap(patches)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal NVUE payload: %w", err)
	}
	return s.opts.ConfigStore.Store(ctx, switchID, sw.GetSiteId(), "nvue-json",
		"nico-switch-config", "pushed via NVUE API", jsonBytes)
}

// Validate compares the switch's running configuration against the desired
// NVUE state derived from NICo. Returns a ValidationResult with mismatches.
func (s *SpectrumConfigurator) Validate(ctx context.Context, switchID string) (*ValidationResult, error) {
	if s.opts.SwitchService == nil {
		return nil, fmt.Errorf("switch service not available")
	}

	sw, err := s.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return nil, fmt.Errorf("switch %s has no management_ip", switchID)
	}

	routing, _ := s.opts.SwitchService.GetRoutingConfig(ctx, switchID)
	desired := patchesToMap(TranslateToNVUEPatches(sw, routing))

	username, password := s.resolveCredentials(sw.GetCredentialRef())
	client := nvue.NewClient(sw.GetManagementIp(), username, password,
		nvue.WithInsecureSkipVerify())

	// Fetch running config from the switch. Empty rev = applied/running state.
	runningRaw, err := client.Get(ctx, "/", "applied")
	if err != nil {
		return nil, fmt.Errorf("get running config from switch %s: %w", switchID, err)
	}
	var runningMap map[string]interface{}
	if err := json.Unmarshal(runningRaw, &runningMap); err != nil {
		return nil, fmt.Errorf("parse running config: %w", err)
	}
	mismatches := diffNVUE(desired, runningMap, "")

	// Return a single summary result; per-port results are stored separately.
	result := &ValidationResult{
		SwitchID: switchID,
		SiteID:   sw.GetSiteId(),
		Match:    len(mismatches) == 0,
	}
	if len(mismatches) > 0 {
		result.ObservedPeer = fmt.Sprintf("mismatches: %v", mismatches)
	}
	return result, nil
}

// UpgradeOS triggers a firmware/OS upgrade to targetVersion via NVUE.
// NVUE path: PATCH /platform/firmware {target: targetVersion} + apply.
func (s *SpectrumConfigurator) UpgradeOS(ctx context.Context, switchID, targetVersion string) error {
	sw, err := s.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return fmt.Errorf("switch %s has no management_ip", switchID)
	}
	username, password := s.resolveCredentials(sw.GetCredentialRef())
	client := nvue.NewClient(sw.GetManagementIp(), username, password, nvue.WithInsecureSkipVerify())

	log.Info().Str("switch_id", switchID).Str("target_version", targetVersion).
		Msg("initiating OS upgrade on Spectrum switch via NVUE")

	patches := []nvue.PatchOp{{
		Path:    "/platform/firmware",
		Payload: map[string]interface{}{"target": targetVersion},
	}}
	if _, err := client.ConfigureAndApply(ctx, patches); err != nil {
		return fmt.Errorf("NVUE firmware upgrade switch %s to %s: %w", switchID, targetVersion, err)
	}
	return nil
}

// ValidateHardware checks PSUs, fans, and transceivers via NVUE GET /platform/environment.
func (s *SpectrumConfigurator) ValidateHardware(ctx context.Context, switchID string) (*HardwareHealthResult, error) {
	sw, err := s.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return nil, fmt.Errorf("switch %s has no management_ip", switchID)
	}
	username, password := s.resolveCredentials(sw.GetCredentialRef())
	client := nvue.NewClient(sw.GetManagementIp(), username, password, nvue.WithInsecureSkipVerify())

	raw, err := client.Get(ctx, "/platform/environment", "applied")
	if err != nil {
		return nil, fmt.Errorf("GET /platform/environment switch %s: %w", switchID, err)
	}

	var env map[string]interface{}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("parse /platform/environment: %w", err)
	}

	result := &HardwareHealthResult{Healthy: true}
	parseComponents := func(key string) []ComponentHealth {
		raw, ok := env[key].(map[string]interface{})
		if !ok {
			return nil
		}
		var out []ComponentHealth
		for name, v := range raw {
			item, _ := v.(map[string]interface{})
			state, _ := item["state"].(string)
			if state == "" {
				state = "unknown"
			}
			if state != "ok" && state != "present" {
				result.Healthy = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s/%s: %s", key, name, state))
			}
			out = append(out, ComponentHealth{Name: name, State: state})
		}
		return out
	}
	result.PSUs = parseComponents("psu")
	result.Fans = parseComponents("fan")
	result.Transceivers = parseComponents("port-module")
	return result, nil
}

// GetDiagnostics reads BGP neighbors, MAC table, and MLAG state from the switch.
func (s *SpectrumConfigurator) GetDiagnostics(ctx context.Context, switchID string) (*SwitchDiagnostics, error) {
	sw, err := s.opts.SwitchService.GetSwitch(ctx, switchID)
	if err != nil {
		return nil, fmt.Errorf("get switch %s: %w", switchID, err)
	}
	if sw.GetManagementIp() == "" {
		return nil, fmt.Errorf("switch %s has no management_ip", switchID)
	}
	username, password := s.resolveCredentials(sw.GetCredentialRef())
	client := nvue.NewClient(sw.GetManagementIp(), username, password, nvue.WithInsecureSkipVerify())

	diag := &SwitchDiagnostics{SwitchID: switchID}

	// BGP neighbors: GET /vrf?include=/*/router/bgp/neighbor
	if bgpRaw, err := client.Get(ctx, "/vrf", "applied"); err == nil {
		var vrfs map[string]interface{}
		if json.Unmarshal(bgpRaw, &vrfs) == nil {
			for vrfName, vrf := range vrfs {
				vMap, _ := vrf.(map[string]interface{})
				router, _ := vMap["router"].(map[string]interface{})
				bgp, _ := router["bgp"].(map[string]interface{})
				neighbors, _ := bgp["neighbor"].(map[string]interface{})
				for peerIP, n := range neighbors {
					nMap, _ := n.(map[string]interface{})
					state, _ := nMap["state"].(string)
					asStr, _ := nMap["remote-as"].(float64)
					diag.BGPNeighbors = append(diag.BGPNeighbors, BGPNeighbor{
						PeerIP: peerIP, VRF: vrfName,
						State: state, RemoteAS: uint32(asStr),
					})
				}
			}
		}
	} else {
		diag.Errors = append(diag.Errors, "bgp: "+err.Error())
	}

	// MAC table: GET /bridge/domain/br_default/mac-table
	if macRaw, err := client.Get(ctx, "/bridge/domain/br_default/mac-table", "applied"); err == nil {
		var macs map[string]interface{}
		if json.Unmarshal(macRaw, &macs) == nil {
			for mac, entry := range macs {
				e, _ := entry.(map[string]interface{})
				vid, _ := e["vlan"].(float64)
				iface, _ := e["interface"].(string)
				diag.MACEntries = append(diag.MACEntries, MACEntry{
					MAC: mac, VID: int(vid), Interface: iface,
				})
			}
		}
	} else {
		diag.Errors = append(diag.Errors, "mac-table: "+err.Error())
	}

	// MLAG: GET /mlag
	if mlagRaw, err := client.Get(ctx, "/mlag", "applied"); err == nil {
		var mlag map[string]interface{}
		if json.Unmarshal(mlagRaw, &mlag) == nil {
			if state, ok := mlag["anycast-ip"].(string); ok {
				diag.MLAGState = state
			}
		}
	}

	return diag, nil
}

// resolveCredentials returns the switch's NVUE credentials.
// TODO: resolve via Vault using credentialRef (credential_ref format:
// "<site_id>/<serial_number>/nvos" as documented in NicoSwitch.credential_ref).
func (s *SpectrumConfigurator) resolveCredentials(credentialRef string) (string, string) {
	_ = credentialRef
	username := os.Getenv("SWITCH_USERNAME")
	if username == "" {
		username = s.opts.SwitchUsername
	}
	password := os.Getenv("SWITCH_PASSWORD")
	if password == "" {
		password = s.opts.SwitchPassword
	}
	return username, password
}

// patchesToMap merges a slice of PatchOps into a single nested map for
// storage and comparison. Keys are the NVUE path segments.
func patchesToMap(patches []nvue.PatchOp) map[string]interface{} {
	merged := make(map[string]interface{})
	for _, p := range patches {
		merged[p.Path] = p.Payload
	}
	return merged
}

// diffNVUE returns human-readable mismatches between desired and running NVUE
// config maps. Only keys present in desired are checked (permissive diff).
func diffNVUE(desired, running map[string]interface{}, prefix string) []string {
	var mismatches []string
	for k, dv := range desired {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		rv, ok := running[k]
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("missing key %s", key))
			continue
		}
		dm, isMap := dv.(map[string]interface{})
		rm, runMap := rv.(map[string]interface{})
		if isMap && runMap {
			mismatches = append(mismatches, diffNVUE(dm, rm, key)...)
		} else if fmt.Sprintf("%v", dv) != fmt.Sprintf("%v", rv) {
			mismatches = append(mismatches, fmt.Sprintf("%s: want %v got %v", key, dv, rv))
		}
	}
	return mismatches
}
