// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)


// SwitchConfigActivities is the unified activity set for the switch-config
// provider. It consolidates list/push/upgrade activities alongside the
// existing CableValidationActivities.
type SwitchConfigActivities struct {
	CableValidationActivities // embed for cable-validation activities

	configurator SwitchConfigurator
}

// NewSwitchConfigActivities creates the unified activity struct.
func NewSwitchConfigActivities(sc *SwitchClient, store *ConfigStore, configurator SwitchConfigurator) *SwitchConfigActivities {
	return &SwitchConfigActivities{
		CableValidationActivities: CableValidationActivities{switchClient: sc, store: store},
		configurator:              configurator,
	}
}

// ListSwitchesForSiteActivity returns all switches for a site via NicoSwitchService.
func (a *SwitchConfigActivities) ListSwitchesForSiteActivity(ctx context.Context, siteID string) ([]SwitchRef, error) {
	activity.RecordHeartbeat(ctx, "listing switches")
	switches, err := a.switchClient.ListSwitches(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list switches for site %s: %w", siteID, err)
	}
	refs := make([]SwitchRef, 0, len(switches))
	for _, sw := range switches {
		provider := sw.GetLabels()["nico.nvidia.com/switch-config-provider"]
		if provider == "" {
			provider = "spectrum" // default when annotation absent
		}
		refs = append(refs, SwitchRef{SwitchID: sw.GetId(), Name: sw.GetName(), Provider: provider})
	}
	return refs, nil
}

// PushConfigActivity pushes NVUE configuration to a single switch.
func (a *SwitchConfigActivities) PushConfigActivity(ctx context.Context, switchID string) error {
	activity.RecordHeartbeat(ctx, "pushing config")
	_, err := a.configurator.Push(ctx, switchID)
	return err
}

// UpgradeOSActivity triggers an OS upgrade on a single switch.
func (a *SwitchConfigActivities) UpgradeOSActivity(ctx context.Context, switchID, targetVersion string) error {
	activity.RecordHeartbeat(ctx, "upgrading OS")
	return a.configurator.UpgradeOS(ctx, switchID, targetVersion)
}

// SwitchConfigDeployWorkflow pushes NICo-derived configuration to all fabric
// switches in the given site that are assigned to the given providerName.
// Switches with a different nico.nvidia.com/switch-config-provider annotation
// are skipped — they are handled by their own provider's workflow.
//
// providerName must match ConfiguratorOptions.Platform, e.g. "spectrum".
// Pass "" to use the default "spectrum".
func SwitchConfigDeployWorkflow(ctx workflow.Context, siteID, providerName string) error {
	if providerName == "" {
		providerName = "spectrum"
	}
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: 5 * time.Second,
			MaximumInterval: 2 * time.Minute,
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var switches []SwitchRef
	if err := workflow.ExecuteActivity(ctx, (*SwitchConfigActivities).ListSwitchesForSiteActivity, siteID).Get(ctx, &switches); err != nil {
		return fmt.Errorf("list switches: %w", err)
	}

	var lastErr error
	for _, sw := range switches {
		// Skip switches assigned to a different switch-config provider.
		if sw.Provider != providerName {
			workflow.GetLogger(ctx).Info("skipping switch assigned to different provider",
				"switchID", sw.SwitchID, "switchProvider", sw.Provider, "thisProvider", providerName)
			continue
		}
		if err := workflow.ExecuteActivity(ctx, (*SwitchConfigActivities).PushConfigActivity, sw.SwitchID).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to push config", "switchID", sw.SwitchID, "error", err)
			lastErr = err // continue to next switch; report last error
		}
	}
	return lastErr
}

// OSUpgradeWorkflow upgrades a single switch's OS to the target version.
// No retry — upgrades are not safe to repeat automatically.
func OSUpgradeWorkflow(ctx workflow.Context, switchID, targetVersion string) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    60 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	return workflow.ExecuteActivity(ctx, (*SwitchConfigActivities).UpgradeOSActivity, switchID, targetVersion).Get(ctx, nil)
}
