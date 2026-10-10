// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

const activityTimeout = 5 * time.Minute

// CableValidationWorkflow orchestrates the four cable-validation activities for
// all switches in a site.
func CableValidationWorkflow(ctx workflow.Context, siteID string) error {
	ao := workflow.ActivityOptions{StartToCloseTimeout: activityTimeout}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var topology ExpectedTopology
	if err := workflow.ExecuteActivity(ctx, (*CableValidationActivities).FetchExpectedTopologyActivity, siteID).Get(ctx, &topology); err != nil {
		return fmt.Errorf("fetch expected topology: %w", err)
	}

	var allResults []*ValidationResult
	for _, sw := range topology.Switches {
		var observed ObservedLLDP
		if err := workflow.ExecuteActivity(ctx, (*CableValidationActivities).FetchObservedLLDPActivity, siteID, sw.GetId()).Get(ctx, &observed); err != nil {
			continue
		}
		var summary ValidationSummary
		if err := workflow.ExecuteActivity(ctx, (*CableValidationActivities).ComputeMismatchesActivity, sw.GetId(), siteID, topology, observed).Get(ctx, &summary); err != nil {
			return fmt.Errorf("compute mismatches for %s: %w", sw.GetId(), err)
		}
		allResults = append(allResults, summary.Results...)
	}

	return workflow.ExecuteActivity(ctx, (*CableValidationActivities).ReportMismatchesActivity, siteID, allResults).Get(ctx, nil)
}

// ExpectedTopology holds switch list and expected LLDP topology from NicoSwitchService.
type ExpectedTopology struct {
	Switches  []*providerv1.NicoSwitch
	Neighbors map[string][]*providerv1.CableLink // keyed by switch_id
}

// ObservedLLDP holds LLDP CableLinks observed on a switch.
type ObservedLLDP struct {
	SwitchID string
	Links    []*providerv1.CableLink
}

// ValidationSummary holds per-switch mismatch results.
type ValidationSummary struct {
	SwitchID string
	Results  []*ValidationResult
}

// CableValidationActivities holds the dependencies injected at worker startup.
type CableValidationActivities struct {
	switchClient *SwitchClient
	store        *ConfigStore
}

// NewCableValidationActivities creates the activity set with required clients.
func NewCableValidationActivities(sc *SwitchClient, store *ConfigStore) *CableValidationActivities {
	return &CableValidationActivities{switchClient: sc, store: store}
}

// FetchExpectedTopologyActivity calls NicoSwitchService for switches and their
// expected neighbors.
func (a *CableValidationActivities) FetchExpectedTopologyActivity(ctx context.Context, siteID string) (ExpectedTopology, error) {
	activity.RecordHeartbeat(ctx, "fetching switches")
	switches, err := a.switchClient.ListSwitches(ctx, siteID)
	if err != nil {
		return ExpectedTopology{}, fmt.Errorf("list switches: %w", err)
	}
	topology := ExpectedTopology{
		Switches:  switches,
		Neighbors: make(map[string][]*providerv1.CableLink),
	}
	for _, sw := range switches {
		neighbors, err := a.switchClient.ListExpectedNeighbors(ctx, siteID, sw.GetId())
		if err != nil {
			log.Ctx(ctx).Warn().Str("switch_id", sw.GetId()).Err(err).Msg("could not fetch expected neighbors")
			continue
		}
		topology.Neighbors[sw.GetId()] = neighbors
	}
	return topology, nil
}

// FetchObservedLLDPActivity returns LLDP neighbors observed on a switch.
// Returns empty ObservedLLDP until Core adds GetSwitchLLDPNeighbors RPC
// or the provider adds SSH/NETCONF discovery.
func (a *CableValidationActivities) FetchObservedLLDPActivity(ctx context.Context, siteID, switchID string) (ObservedLLDP, error) {
	activity.RecordHeartbeat(ctx, "fetching LLDP for "+switchID)
	// TODO: implement live LLDP discovery.
	return ObservedLLDP{SwitchID: switchID}, nil
}

// ComputeMismatchesActivity diffs expected vs. observed LLDP per port.
func (a *CableValidationActivities) ComputeMismatchesActivity(ctx context.Context, switchID, siteID string, expected ExpectedTopology, observed ObservedLLDP) (ValidationSummary, error) {
	activity.RecordHeartbeat(ctx, "computing mismatches for "+switchID)

	// Build map: local port → observed remote device
	observedByPort := make(map[string]string)
	for _, link := range observed.Links {
		if link.GetLocal() != nil && link.GetRemote() != nil {
			observedByPort[link.GetLocal().GetPortName()] = link.GetRemote().GetDeviceId()
		}
	}

	var results []*ValidationResult
	for _, link := range expected.Neighbors[switchID] {
		if link.GetLocal() == nil || link.GetRemote() == nil {
			continue
		}
		port := link.GetLocal().GetPortName()
		expectedPeer := link.GetRemote().GetDeviceId()
		observedPeer := observedByPort[port]
		match := observedPeer != "" && observedPeer == expectedPeer
		results = append(results, &ValidationResult{
			SiteID:       siteID,
			SwitchID:     switchID,
			Port:         port,
			ExpectedPeer: expectedPeer,
			ObservedPeer: observedPeer,
			Match:        match,
		})
	}
	return ValidationSummary{SwitchID: switchID, Results: results}, nil
}

// ReportMismatchesActivity reports observed neighbors to NICo and stores results.
func (a *CableValidationActivities) ReportMismatchesActivity(ctx context.Context, siteID string, results []*ValidationResult) error {
	activity.RecordHeartbeat(ctx, "reporting mismatches")

	// Group CableLinks by switch and report back.
	bySwitchID := make(map[string][]*providerv1.CableLink)
	for _, r := range results {
		bySwitchID[r.SwitchID] = append(bySwitchID[r.SwitchID], &providerv1.CableLink{
			Local:  &providerv1.CableEndpoint{PortName: r.Port},
			Remote: &providerv1.CableEndpoint{DeviceId: r.ObservedPeer},
			Status: map[bool]string{true: "match", false: "mismatch"}[r.Match],
		})
	}
	for switchID, links := range bySwitchID {
		if err := a.switchClient.ReportObservedNeighbors(ctx, siteID, switchID, links); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("switch_id", switchID).Msg("failed to report observed neighbors")
		}
	}

	if a.store != nil {
		return a.store.StoreValidationResults(ctx, results)
	}
	return nil
}
