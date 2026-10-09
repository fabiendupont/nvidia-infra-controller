// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metal3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IronicInspectorClient fetches node introspection data from Ironic Inspector.
// Ironic Inspector exposes GET /v1/introspection/{node_uuid}/data.
type IronicInspectorClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewIronicInspectorClient creates a client for the Ironic Inspector API.
// baseURL should be the inspector endpoint, e.g. http://ironic-inspector.metal3-system:5050.
// token is an optional Bearer token; pass empty string if not required.
func NewIronicInspectorClient(baseURL, token string) *IronicInspectorClient {
	return &IronicInspectorClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// IntrospectionExtra holds the fields written by the nico-ipa-extensions
// collectors during IPA inspection. Standard Ironic Inspector fields occupy
// the top-level IntrospectionData object; our custom fields live in `extra`.
type IntrospectionExtra struct {
	// TPMEKCert is the base64-encoded PEM EK certificate from the TPM.
	TPMEKCert string `json:"tpm_ekcert"`
	// TPMVersion is the TPM specification version, typically "2.0".
	TPMVersion string `json:"tpm_version"`
	// PlatformSerial is the platform serial number extracted from the EK cert
	// SubjectAltName, used to correlate with the BMC SPDM DevID identity.
	PlatformSerial string `json:"platform_serial"`
	// PCR0 is the hex-encoded PCR[0] value (BIOS/UEFI code measurement).
	PCR0 string `json:"pcr0"`
	// PCR2 is the hex-encoded PCR[2] value (option ROM / GPU firmware measurement).
	PCR2 string `json:"pcr2"`
	// PCR7 is the hex-encoded PCR[7] value (Secure Boot state measurement).
	PCR7 string `json:"pcr7"`
}

// IntrospectionData is the response from Ironic Inspector's introspection data endpoint.
// Only the fields relevant to NICo attestation are decoded; the full response
// contains many additional fields (CPU, memory, disks, NICs, etc.).
type IntrospectionData struct {
	Extra IntrospectionExtra `json:"extra"`
}

// GetIntrospectionData fetches introspection data for an Ironic node by UUID.
// Returns nil, nil if the node exists but has no introspection data yet (HTTP 404).
func (c *IronicInspectorClient) GetIntrospectionData(ctx context.Context, nodeUUID string) (*IntrospectionData, error) {
	url := fmt.Sprintf("%s/v1/introspection/%s/data", c.baseURL, nodeUUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Node exists but introspection data not yet available.
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("Ironic Inspector returned %d: %s", resp.StatusCode, body)
	}

	var data IntrospectionData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode introspection data: %w", err)
	}
	return &data, nil
}
