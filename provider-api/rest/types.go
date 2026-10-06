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

package rest

import "time"

// APIStatusDetail captures API representation of a status detail
type APIStatusDetail struct {
	// Status denotes the state of the associated entity at a particular time
	Status string `json:"status"`
	// Message contains the description of the state and cause/remedy in case of error
	Message *string `json:"message"`
	// Created indicates the ISO datetime string for when the associated entity assumed the status
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the associated entity was last found to have this status
	Updated time.Time `json:"updated"`
}

// APIInfrastructureProviderSummary is the data structure to capture API representation of an Infrastructure Provider
type APIInfrastructureProviderSummary struct {
	// Org contains the name of the org the Infrastructure Provider belongs to
	Org string `json:"org"`
	// OrgDisplayName contains the display name of the org this Infrastructure Provider belongs to
	OrgDisplayName *string `json:"orgDisplayName"`
}

// APITenantCapabilities holds the model of tenant capabilities
type APITenantCapabilities struct {
	TargetedInstanceCreation bool `json:"targetedInstanceCreation"`
}

// APITenantSummary is the data structure to capture API representation of a Tenant Summary
type APITenantSummary struct {
	// Org contains the name of the org this tenant belongs to
	Org string `json:"org"`
	// OrgDisplayName contains the display name of the org the Tenant belongs to
	OrgDisplayName *string `json:"orgDisplayName"`
	// Capabilities hold the capabilities, currently for use as tenant-level feature flagging
	Capabilities *APITenantCapabilities `json:"capabilities"`
}

// APISiteCapabilities holds the model of site capabilities
type APISiteCapabilities struct {
	NativeNetworking          bool `json:"nativeNetworking"`
	NetworkSecurityGroup      bool `json:"networkSecurityGroup"`
	NVLinkPartition           bool `json:"nvLinkPartition"`
	RackLevelAdministration   bool `json:"rackLevelAdministration"`
	ImageBasedOperatingSystem bool `json:"imageBasedOperatingSystem"`
}

// APISiteSummary is the data structure to capture API summary of a Site
type APISiteSummary struct {
	// ID is the unique UUID v4 identifier for the Site
	ID string `json:"id"`
	// Name of the Site, only lowercase characters, digits, hyphens and cannot begin/end with hyphen
	Name string `json:"name"`
	// InfrastructureProviderID is the ID of the infrastructure provider who owns the site
	InfrastructureProviderID string `json:"infrastructureProviderId"`
	// InfrastructureProvider is the summary of the InfrastructureProvider
	InfrastructureProvider *APIInfrastructureProviderSummary `json:"infrastructureProvider,omitempty"`
	// IsSerialConsoleEnabled is a flag to indicate if serial console is enabled
	IsSerialConsoleEnabled bool `json:"isSerialConsoleEnabled"`
	// IsOnline is the connection status attribute for Site
	IsOnline bool `json:"isOnline"`
	// Status is the status of the site
	Status string `json:"status"`
	// Capabilities holds the capabilities, currently for use as site-level feature flagging
	Capabilities *APISiteCapabilities `json:"capabilities"`
}

// APIInstanceSummary is the data structure to capture API summary of an Instance
type APIInstanceSummary struct {
	// ID of the Instance
	ID string `json:"id"`
	// Name of the Instance, only lowercase characters, digits, hyphens and cannot begin/end with hyphen
	Name string `json:"name"`
	// InfrastructureProviderID is the ID of the Infrastructure Provider
	InfrastructureProviderID string `json:"infrastructureProviderId"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// InstanceTypeID is the ID of the InstanceType
	InstanceTypeID string `json:"instanceTypeId"`
	// ControllerInstanceID is the ID of the Instance in Site Controller
	ControllerInstanceID string `json:"controllerInstanceId"`
	// Status is the status of the Instance
	Status string `json:"status"`
}
