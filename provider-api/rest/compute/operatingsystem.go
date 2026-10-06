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

import (
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
)

// APIOperatingSystemCreateRequest is the data structure to capture user request to create a new OperatingSystem
type APIOperatingSystemCreateRequest struct {
	// Name is the name of the OperatingSystem
	Name string `json:"name"`
	// Description is the description of the OperatingSystem
	Description *string `json:"description"`
	// InfrastructureProviderID is the ID of the InfrastructureProvider creating the Operating System
	InfrastructureProviderID *string `json:"infrastructureProviderId"`
	// SiteIDs is a list of Site objects
	SiteIDs []string `json:"siteIds"`
	// TenantID is the ID of the Tenant creating the Operating System
	TenantID *string `json:"tenantId"`
	// IpxeScript is the iPXE script for the Operating System
	IpxeScript *string `json:"ipxeScript"`
	// ImageURL is the image path for the Operating System
	ImageURL *string `json:"imageUrl"`
	// ImageSHA is SHA for the Operating System image type
	ImageSHA *string `json:"imageSha"`
	// ImageAuthType is auth type for the Operating System type
	ImageAuthType *string `json:"imageAuthType"`
	// ImageAuthToken is auth token for for the Operating System image type
	ImageAuthToken *string `json:"imageAuthToken"`
	// ImageDisk is disk for the Operating System image type
	ImageDisk *string `json:"imageDisk"`
	// RootFsID is root fs id for the Operating System image type
	RootFsID *string `json:"rootFsId"`
	// RootFsLabel is root fs label for the Operating System image type
	RootFsLabel *string `json:"rootFsLabel"`
	// PhoneHomeEnabled is the flag to allow enable phone home
	PhoneHomeEnabled *bool `json:"phoneHomeEnabled"`
	// UserData is the user data for the Operating System
	UserData *string `json:"userData"`
	// IsCloudInit indicates if the Operating System needs cloud init
	IsCloudInit bool `json:"isCloudInit"`
	// AllowOverride indicates if overrides are allowed
	AllowOverride bool `json:"allowOverride"`
	// EnableBlockStorage indicates whether the Operating System image will be stored remotely via block storage
	EnableBlockStorage bool `json:"enableBlockStorage"`
}

// APIOperatingSystemUpdateRequest is the data structure to capture user request to update an OperatingSystem
type APIOperatingSystemUpdateRequest struct {
	// Name is the name of the OperatingSystem
	Name *string `json:"name"`
	// Description is the description of the Operating System
	Description *string `json:"description"`
	// IpxeScript is the ipxe script for the Operating System
	IpxeScript *string `json:"ipxeScript"`
	// ImageURL is the image path for the Operating System
	ImageURL *string `json:"imageUrl"`
	// ImageSHA is SHA for the Operating System image type
	ImageSHA *string `json:"imageSha"`
	// ImageAuthType is auth type for the Operating System type
	ImageAuthType *string `json:"imageAuthType"`
	// ImageAuthToken is auth token for for the Operating System image type
	ImageAuthToken *string `json:"imageAuthToken"`
	// ImageDisk is disk for the Operating System image type
	ImageDisk *string `json:"imageDisk"`
	// RootFsID is root fs id for the Operating System image type
	RootFsID *string `json:"rootFsId"`
	// RootFsLabel is root fs label for the Operating System image type
	RootFsLabel *string `json:"rootFsLabel"`
	// PhoneHomeEnabled is the flag to allow enable phone home
	PhoneHomeEnabled *bool `json:"phoneHomeEnabled"`
	// UserData is the user data for the Operating System
	UserData *string `json:"userData"`
	// IsCloudInit indicates if the Operating System needs cloud init
	IsCloudInit *bool `json:"isCloudInit"`
	// AllowOverride indicates if overrides are allowed
	AllowOverride *bool `json:"allowOverride"`
	// IsActive indicates if the Operating System is active
	IsActive *bool `json:"isActive"`
	// DeactivationNote is the deactivation note if any
	DeactivationNote *string `json:"deactivationNote"`
}

// APIOperatingSystem is the data structure to capture API representation of an OS
type APIOperatingSystem struct {
	// ID is the unique UUID v4 identifier for the Operating System
	ID string `json:"id"`
	// Name is the name of the Operating System
	Name string `json:"name"`
	// Description is the description of the Operating System
	Description *string `json:"description"`
	// InfrastructureProviderID is the ID of the InfrastructureProvider creating the OS
	InfrastructureProviderID *string `json:"infrastructureProviderId"`
	// InfrastructureProvider is the summary of the InfrastructureProvider
	InfrastructureProvider *rest.APIInfrastructureProviderSummary `json:"infrastructureProvider,omitempty"`
	// TenantID is the ID of the tenant creating the Operating System
	TenantID *string `json:"tenantId"`
	// Tenant is the summary of the Tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// Type is which type of Operating System
	Type *string `json:"type"`
	// ImageUrl is url path for the Operating System
	ImageURL *string `json:"imageUrl"`
	// ImageSHA is SHA for the Operating System image type
	ImageSHA *string `json:"imageSha"`
	// ImageAuthType is auth type for the Operating System type
	ImageAuthType *string `json:"imageAuthType"`
	// ImageAuthToken is auth token for for the Operating System image type
	ImageAuthToken *string `json:"imageAuthToken"`
	// ImageDisk is disk for the Operating System image type
	ImageDisk *string `json:"imageDisk"`
	// RootFsID is root fs id for the Operating System image type
	RootFsID *string `json:"rootFsId"`
	// RootFsLabel is root fs id for the Operating System image type
	RootFsLabel *string `json:"rootFsLabel"`
	// IpxeScript is the ipxe script for the Operating System
	IpxeScript *string `json:"ipxeScript"`
	// PhoneHomeEnabled is an attribute which is specified by user if Operating System needs to be enabled for phone home or not
	PhoneHomeEnabled bool `json:"phoneHomeEnabled"`
	// UserData is the user data for the Operating System
	UserData *string `json:"userData"`
	// IsCloudInit indicates if the Operating System needs cloud init
	IsCloudInit bool `json:"isCloudInit"`
	// AllowOverride indicates if overrides are allowed
	AllowOverride bool `json:"allowOverride"`
	// EnableBlockStorage indicates whether the Operating System image will be stored remotely via block storage
	EnableBlockStorage bool `json:"enableBlockStorage"`
	// IsActive indicates if the Operating System is active
	IsActive bool `json:"isActive"`
	// DeactivationNote is the deactivation note if any
	DeactivationNote *string `json:"deactivationNote"`
	// Status is the status of the Operating System
	Status string `json:"status"`
	// StatusHistory is the history of statuses for the Operating System
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// SiteAssociations is the list of Sites associated with the Operating System
	SiteAssociations []APIOperatingSystemSiteAssociation `json:"siteAssociations"`
	// CreatedAt indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// UpdatedAt indicates the ISO datetime string for when the entity was last updated
	Updated time.Time `json:"updated"`
}

// APIOperatingSystemSummary is the data structure to capture API summary of an OperatingSystem
type APIOperatingSystemSummary struct {
	// ID of the OperatingSystem
	ID string `json:"id"`
	// Name of the OperatingSystem, only lowercase characters, digits, hyphens and cannot begin/end with hyphen
	Name string `json:"name"`
	// Type is which type of Operating System
	Type *string `json:"type"`
	// Status is the status of the Operating System
	Status string `json:"status"`
}
