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

// TrayFilter specifies which trays to target in a batch operation.
// If nil or empty, the operation targets all trays in the site.
type TrayFilter struct {
	RackID       *string  `json:"rackId,omitempty"`
	RackName     *string  `json:"rackName,omitempty"`
	Type         *string  `json:"type,omitempty"`
	ComponentIDs []string `json:"componentIds,omitempty"`
	IDs          []string `json:"ids,omitempty"`
}

// APITrayGetAllRequest captures query parameters for listing trays from RLA.
type APITrayGetAllRequest struct {
	SiteID       string   `query:"siteId"`
	RackID       *string  `query:"rackId"`
	RackName     *string  `query:"rackName"`
	Type         *string  `query:"type"`
	ComponentIDs []string `query:"componentId"`
	IDs          []string `query:"id"`
}

// APITrayValidateAllRequest captures query parameters for validating trays.
type APITrayValidateAllRequest struct {
	SiteID       string   `query:"siteId"`
	RackID       *string  `query:"rackId"`
	RackName     *string  `query:"rackName"`
	Name         []string `query:"name"`
	Manufacturer []string `query:"manufacturer"`
	Type         *string  `query:"type"`
	ComponentIDs []string `query:"componentId"`
}

// APITrayPosition represents the position of a tray within a rack
type APITrayPosition struct {
	SlotID  int32 `json:"slotId"`
	TrayIdx int32 `json:"trayIdx"`
	HostID  int32 `json:"hostId"`
}

// APITray is the API representation of a Tray (Component) from RLA
type APITray struct {
	ID              string           `json:"id"`
	ComponentID     string           `json:"componentId"`
	Type            string           `json:"type"`
	Name            string           `json:"name"`
	Manufacturer    string           `json:"manufacturer"`
	Model           string           `json:"model"`
	SerialNumber    string           `json:"serialNumber"`
	Description     string           `json:"description"`
	FirmwareVersion string           `json:"firmwareVersion"`
	PowerState      string           `json:"powerState"`
	Position        *APITrayPosition `json:"position"`
	BMCs            []*APIBMC        `json:"bmcs"`
	RackID          string           `json:"rackId"`
}
