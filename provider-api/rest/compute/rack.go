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

package compute

import "fmt"

// RackFilter specifies which racks to target in a batch operation.
// If nil or empty, the operation targets all racks in the site.
type RackFilter struct {
	Names []string `json:"names,omitempty"`
}

// APIRackGetRequest captures query parameters for getting a single rack.
type APIRackGetRequest struct {
	SiteID            string `query:"siteId"`
	IncludeComponents bool   `query:"includeComponents"`
}

// Validate checks required fields for a single rack get request.
func (r *APIRackGetRequest) Validate() error {
	if r.SiteID == "" {
		return fmt.Errorf("siteId query parameter is required")
	}
	return nil
}

// APIRackGetAllRequest captures query parameters for listing racks.
type APIRackGetAllRequest struct {
	SiteID            string   `query:"siteId"`
	IncludeComponents bool     `query:"includeComponents"`
	Name              []string `query:"name"`
	Manufacturer      []string `query:"manufacturer"`
	PageNumber        string   `query:"pageNumber"`
	PageSize          string   `query:"pageSize"`
	OrderBy           string   `query:"orderBy"`
}

// Validate checks required fields for listing racks.
func (r *APIRackGetAllRequest) Validate() error {
	if r.SiteID == "" {
		return fmt.Errorf("siteId query parameter is required")
	}
	return nil
}

// APIRackValidateAllRequest captures query parameters for validating racks.
type APIRackValidateAllRequest struct {
	SiteID       string   `query:"siteId"`
	Name         []string `query:"name"`
	Manufacturer []string `query:"manufacturer"`
}

// Validate checks required fields for validating racks.
func (r *APIRackValidateAllRequest) Validate() error {
	if r.SiteID == "" {
		return fmt.Errorf("siteId query parameter is required")
	}
	return nil
}

// APIRack is the API representation of a Rack from RLA
type APIRack struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Manufacturer string              `json:"manufacturer"`
	Model        string              `json:"model"`
	SerialNumber string              `json:"serialNumber"`
	Description  string              `json:"description"`
	Location     *APIRackLocation    `json:"location,omitempty"`
	Components   []*APIRackComponent `json:"components,omitempty"`
}

// APIRackLocation represents the location of a rack
type APIRackLocation struct {
	Region     string `json:"region"`
	Datacenter string `json:"datacenter"`
	Room       string `json:"room"`
	Position   string `json:"position"`
}

// APIBMC represents a BMC (Baseboard Management Controller) entry
type APIBMC struct {
	Type       string `json:"type"`
	MacAddress string `json:"macAddress"`
	IPAddress  string `json:"ipAddress"`
}

// APIRackComponent represents a component within a rack
type APIRackComponent struct {
	ID              string    `json:"id"`
	ComponentID     string    `json:"componentId"`
	RackID          string    `json:"rackId"`
	Type            string    `json:"type"`
	Name            string    `json:"name"`
	SerialNumber    string    `json:"serialNumber"`
	Manufacturer    string    `json:"manufacturer"`
	Model           string    `json:"model"`
	Description     string    `json:"description"`
	FirmwareVersion string    `json:"firmwareVersion"`
	SlotID          int32     `json:"slotId"`
	TrayIdx         int32     `json:"trayIdx"`
	HostID          int32     `json:"hostId"`
	BMCs            []*APIBMC `json:"bmcs"`
	PowerState      string    `json:"powerState"`
}

// APIFieldDiff represents a single field difference
type APIFieldDiff struct {
	FieldName     string `json:"fieldName"`
	ExpectedValue string `json:"expectedValue"`
	ActualValue   string `json:"actualValue"`
}

// APIComponentDiff represents a single component difference found during validation
type APIComponentDiff struct {
	Type        string            `json:"type"`
	ID          string            `json:"id,omitempty"`
	ComponentID string            `json:"componentId,omitempty"`
	Expected    *APIRackComponent `json:"expected,omitempty"`
	Actual      *APIRackComponent `json:"actual,omitempty"`
	FieldDiffs  []*APIFieldDiff   `json:"fieldDiffs,omitempty"`
}

// APIRackValidationResult is the API representation of a rack validation result
type APIRackValidationResult struct {
	Diffs           []*APIComponentDiff `json:"diffs"`
	TotalDiffs      int32               `json:"totalDiffs"`
	MissingCount    int32               `json:"missingCount"`
	UnexpectedCount int32               `json:"unexpectedCount"`
	DriftCount      int32               `json:"driftCount"`
	MatchCount      int32               `json:"matchCount"`
}

// APIBringUpRackRequest is the request body for bring up operations on a single rack
type APIBringUpRackRequest struct {
	SiteID      string `json:"siteId"`
	Description string `json:"description,omitempty"`
}

// Validate validates the bring up request
func (r *APIBringUpRackRequest) Validate() error {
	if r.SiteID == "" {
		return fmt.Errorf("siteId is required")
	}
	return nil
}

// APIBringUpRackResponse is the API response for bring up operations
type APIBringUpRackResponse struct {
	TaskIDs []string `json:"taskIds"`
}

// APIBatchBringUpRackRequest is the JSON body for batch rack bring up.
type APIBatchBringUpRackRequest struct {
	SiteID      string      `json:"siteId"`
	Filter      *RackFilter `json:"filter,omitempty"`
	Description string      `json:"description,omitempty"`
}

// Validate checks required fields.
func (r *APIBatchBringUpRackRequest) Validate() error {
	if r.SiteID == "" {
		return fmt.Errorf("siteId is required")
	}
	return nil
}
