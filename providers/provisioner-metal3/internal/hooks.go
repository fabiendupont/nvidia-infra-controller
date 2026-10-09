// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metal3

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

// provisionHookPayload is the expected shape of the pre-machine-provision hook payload.
// Fields match what the compute provider emits when starting provisioning.
type provisionHookPayload struct {
	MachineID         string `json:"machine_id"`
	Provisioner       string `json:"provisioner"` // annotation value, e.g. "metal3"
	BMCURL            string `json:"bmc_url"`
	BMCUsername       string `json:"bmc_username"`
	BMCPassword       string `json:"bmc_password"`
	BootMACAddress    string `json:"boot_mac_address,omitempty"`
	ImageURL          string `json:"image_url"`
	ImageChecksum     string `json:"image_checksum,omitempty"`
	ImageChecksumType string `json:"image_checksum_type,omitempty"`
	Hostname          string `json:"hostname,omitempty"`
}

// handlePreMachineProvision is the sync hook handler for pre-machine-provision.
// It intercepts provisioning for machines annotated with nico.nvidia.com/provisioner=metal3
// and drives BMO directly. Machines without the annotation are passed through (success).
func (s *Server) handlePreMachineProvision(ctx context.Context, rawPayload interface{}) error {
	raw, err := json.Marshal(rawPayload)
	if err != nil {
		return fmt.Errorf("marshal hook payload: %w", err)
	}
	var payload provisionHookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse hook payload: %w", err)
	}

	logger := log.With().
		Str("hook", "pre-machine-provision").
		Str("machine_id", payload.MachineID).
		Str("provisioner_annotation", payload.Provisioner).
		Logger()

	// Only act if the machine is annotated for Metal3.
	if payload.Provisioner != ProvisionerName {
		logger.Debug().Msg("provisioner annotation not metal3, passing through")
		return nil
	}

	if s.provisioner == nil {
		return fmt.Errorf("Metal3 provisioner not initialized")
	}

	bmhName := "nico-" + payload.MachineID
	logger.Info().Str("bmc_url", payload.BMCURL).Str("image_url", payload.ImageURL).
		Str("bmh_name", bmhName).
		Msg("Metal3 provisioner taking ownership of machine provisioning")

	req := ProvisionRequest{
		MachineID:         payload.MachineID,
		BMCURL:            payload.BMCURL,
		BMCUsername:       payload.BMCUsername,
		BMCPassword:       payload.BMCPassword,
		BootMACAddress:    payload.BootMACAddress,
		ImageURL:          payload.ImageURL,
		ImageChecksum:     payload.ImageChecksum,
		ImageChecksumType: payload.ImageChecksumType,
		Hostname:          payload.Hostname,
	}
	if err := s.provisioner.Provision(ctx, req); err != nil {
		return err
	}
	// Gap 3: record machineID → bmhName so post-machine-inspect can look up the
	// Ironic node UUID without needing a NICo API write-back.
	s.machineToNode.Store(payload.MachineID, bmhName)
	return nil
}

// inspectHookPayload is the expected shape of the post-machine-inspect hook payload.
// NICo emits this after Ironic/IPA inspection completes for a Metal3-managed machine.
type inspectHookPayload struct {
	MachineID      string `json:"machine_id"`
	IronicNodeUUID string `json:"ironic_node_uuid"` // Ironic's internal node UUID
	// BMCDevIDSerial is the platform serial extracted from the BMC's SPDM DevID
	// certificate during NICo's initial BMC discovery. Empty if SPDM is not supported
	// by the BMC or if discovery has not yet run.
	BMCDevIDSerial string `json:"bmc_devid_serial"`
}

// handlePostMachineInspect fires after Ironic/IPA inspection completes.
// It reads the TPM EK certificate and PCR baseline from Ironic Inspector's
// introspection data and cross-correlates the platform serial with the BMC
// SPDM DevID identity to detect hardware substitution.
func (s *Server) handlePostMachineInspect(ctx context.Context, rawPayload interface{}) error {
	raw, err := json.Marshal(rawPayload)
	if err != nil {
		return fmt.Errorf("marshal hook payload: %w", err)
	}
	var payload inspectHookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse hook payload: %w", err)
	}

	logger := log.With().
		Str("hook", "post-machine-inspect").
		Str("machine_id", payload.MachineID).
		Str("ironic_node_uuid", payload.IronicNodeUUID).
		Logger()

	// Gap 3: if IronicNodeUUID is absent from the hook payload, fall back to the
	// in-memory map populated by pre-machine-provision (bmhName == Ironic node UUID).
	if payload.IronicNodeUUID == "" {
		if v, ok := s.machineToNode.Load(payload.MachineID); ok {
			payload.IronicNodeUUID = v.(string)
			logger.Debug().Str("ironic_node_uuid", payload.IronicNodeUUID).
				Msg("resolved Ironic node UUID from pre-provision map")
		} else {
			logger.Debug().Msg("no Ironic node UUID in payload or map, skipping Inspector lookup")
			return nil
		}
	}

	if s.inspector == nil {
		logger.Warn().Msg("Ironic Inspector not configured (IRONIC_INSPECTOR_URL unset), skipping TPM correlation")
		return nil
	}

	data, err := s.inspector.GetIntrospectionData(ctx, payload.IronicNodeUUID)
	if err != nil {
		// Inspector unavailable is non-fatal: log and proceed.
		logger.Warn().Err(err).Msg("could not fetch Ironic introspection data, skipping TPM correlation")
		return nil
	}
	if data == nil {
		logger.Warn().Msg("no introspection data available yet for node")
		return nil
	}

	extra := data.Extra

	// Cross-correlate TPM EK platform serial with BMC SPDM DevID serial.
	// A mismatch indicates the TPM or BMC was swapped — possible hardware substitution.
	if payload.BMCDevIDSerial != "" && extra.PlatformSerial != "" {
		tpmSerial := normalizeSerial(extra.PlatformSerial)
		bmcSerial := normalizeSerial(payload.BMCDevIDSerial)
		if tpmSerial != bmcSerial {
			return fmt.Errorf(
				"TPM EK platform serial %q does not match BMC SPDM DevID serial %q for machine %s — possible hardware substitution",
				extra.PlatformSerial, payload.BMCDevIDSerial, payload.MachineID,
			)
		}
		logger.Info().
			Str("platform_serial", extra.PlatformSerial).
			Msg("TPM EK and BMC SPDM identity corroborated")
	} else {
		logger.Debug().
			Str("tpm_serial", extra.PlatformSerial).
			Str("bmc_spdm_serial", payload.BMCDevIDSerial).
			Msg("serial correlation skipped (one or both values absent)")
	}

	// Log PCR baseline for operator visibility. NICo Core uses these as reference
	// values when comparing future SPDM firmware measurements.
	logger.Info().
		Str("tpm_version", extra.TPMVersion).
		Str("pcr0", extra.PCR0).
		Str("pcr2", extra.PCR2).
		Str("pcr7", extra.PCR7).
		Msg("TPM PCR baseline recorded at inspection time")

	return nil
}

// normalizeSerial strips hyphens and lowercases a serial number for comparison.
func normalizeSerial(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "-", ""))
}

// postProvisionPayload is the expected shape of the post-machine-provision hook payload.
type postProvisionPayload struct {
	MachineID   string `json:"machine_id"`
	Provisioner string `json:"provisioner"` // "metal3"
	OSType      string `json:"os_type"`     // "nico-managed" | "user-provisioned"
	// AttestationState is set by NICo Core from its SPDM polling of the BMC.
	// Values: "hardware_verified" | "hardware_failed" | "unknown" | ""
	AttestationState string `json:"attestation_state"`
}

// handlePostMachineProvision gates the provisioning→running transition on hardware
// attestation state. It blocks machines whose SPDM attestation has explicitly
// failed; machines with unknown or missing attestation state are allowed through
// with a warning (SPDM polling may not have run yet, or the BMC does not support it).
//
// This hook does not contact Keylime — runtime TPM attestation for NICo-managed
// OS is handled by the Keylime provider (NEP-0010, planned).
func (s *Server) handlePostMachineProvision(ctx context.Context, rawPayload interface{}) error {
	raw, err := json.Marshal(rawPayload)
	if err != nil {
		return fmt.Errorf("marshal hook payload: %w", err)
	}
	var payload postProvisionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("parse hook payload: %w", err)
	}

	if payload.Provisioner != ProvisionerName {
		return nil
	}

	logger := log.With().
		Str("hook", "post-machine-provision").
		Str("machine_id", payload.MachineID).
		Str("os_type", payload.OSType).
		Str("attestation_state", payload.AttestationState).
		Logger()

	// Gap 4: if AttestationState is not in the hook payload, query NicoComputeService
	// for the nico.nvidia.com/attestation-state label set by NICo Core's SPDM polling.
	if payload.AttestationState == "" && s.computeClient != nil {
		resp, err := s.computeClient.GetMachineByID(ctx, &providerv1.GetMachineByIDRequest{
			Id: payload.MachineID,
		})
		if err != nil {
			logger.Warn().Err(err).Msg("could not fetch machine from NicoComputeService for attestation label; defaulting to unknown")
		} else if resp.GetMachine() != nil {
			if state, ok := resp.GetMachine().GetLabels()["nico.nvidia.com/attestation-state"]; ok && state != "" {
				payload.AttestationState = state
				logger.Debug().Str("attestation_state", state).Msg("resolved attestation state from machine label")
			}
		}
	}

	switch payload.AttestationState {
	case "hardware_failed":
		return fmt.Errorf(
			"hardware attestation failed for machine %s (SPDM or PCR mismatch) — cannot transition to running",
			payload.MachineID,
		)
	case "hardware_verified":
		logger.Info().Msg("hardware attestation verified, allowing provisioning→running")
		return nil
	default:
		// "unknown", "" — SPDM not yet run or BMC does not support SPDM.
		// Allow the transition with a warning rather than blocking indefinitely.
		logger.Warn().
			Str("attestation_state", payload.AttestationState).
			Msg("hardware attestation state unknown; allowing provisioning→running without SPDM confirmation")
		return nil
	}
}

// deprovisionHookPayload is the expected shape of the post-machine-deprovision payload.
type deprovisionHookPayload struct {
	MachineID   string `json:"machine_id"`
	Provisioner string `json:"provisioner"`
}

// handlePostMachineDeprovision cleans up BMO resources after NICo releases a machine.
func (s *Server) handlePostMachineDeprovision(ctx context.Context, rawPayload interface{}) error {
	raw, err := json.Marshal(rawPayload)
	if err != nil {
		return nil // non-fatal: deprovision cleanup is best-effort
	}
	var payload deprovisionHookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	if payload.Provisioner != ProvisionerName || s.provisioner == nil {
		return nil
	}

	log.Info().Str("machine_id", payload.MachineID).Msg("Metal3: deprovisioning machine")
	return s.provisioner.Deprovision(ctx, payload.MachineID)
}
