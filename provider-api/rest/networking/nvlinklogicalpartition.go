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

package networking

import (
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	validationis "github.com/go-ozzo/ozzo-validation/v4/is"
)

// APINVLinkLogicalPartitionCreateRequest is the data structure to capture instance request to create a new NVLinkLogicalPartition
type APINVLinkLogicalPartitionCreateRequest struct {
	// Name is the name of the NVLinkLogicalPartition
	Name string `json:"name"`
	// Description is the description of the NVLinkLogicalPartition
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
}

// Validate ensure the values passed in request are acceptable
func (anlpcr APINVLinkLogicalPartitionCreateRequest) Validate() error {
	err := validation.ValidateStruct(&anlpcr,
		validation.Field(&anlpcr.Name,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		validation.Field(&anlpcr.SiteID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
	)
	if err != nil {
		return err
	}
	return nil
}

// APINVLinkLogicalPartitionUpdateRequest is the data structure to capture user request to update a NVLinkLogicalPartition
type APINVLinkLogicalPartitionUpdateRequest struct {
	// Name is the name of the NVLinkLogicalPartition
	Name *string `json:"name"`
	// Description is the description of the NVLinkLogicalPartition
	Description *string `json:"description"`
}

// Validate ensure the values passed in request are acceptable
func (anlpur APINVLinkLogicalPartitionUpdateRequest) Validate() error {
	return validation.ValidateStruct(&anlpur,
		validation.Field(&anlpur.Name,
			validation.When(anlpur.Name != nil, validation.Required.Error(rest.ValidationErrorStringLength)),
			validation.When(anlpur.Name != nil, validation.By(rest.ValidateNameCharacters)),
			validation.When(anlpur.Name != nil, validation.Length(2, 256).Error(rest.ValidationErrorStringLength))),
	)
}

// APINVLinkLogicalPartition is the data structure to capture API representation of a NVLinkLogicalPartition
type APINVLinkLogicalPartition struct {
	// ID is the unique UUID v4 identifier for the NVLinkLogicalPartition
	ID string `json:"id"`
	// Name is the name of the NVLinkLogicalPartition
	Name string `json:"name"`
	// Description is the description of the NVLinkLogicalPartition
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// Vpcs is the list of VPCs associated with the NVLinkLogicalPartition
	Vpcs []APIVpcSummary `json:"vpcs,omitempty"`
	// NVLinkInterfaces is the list of NVLinkInterfaces associated with the NVLinkLogicalPartition
	NVLinkInterfaces []APINVLinkInterfaceSummary `json:"nvLinkInterfaces,omitempty"`
	// NVLinkLogicalPartitionStats holds GPU and instance counts for a NVLinkLogicalPartition
	NVLinkLogicalPartitionStats *APINVLinkLogicalPartitionStats `json:"nvLinkLogicalPartitionStats"`
	// Status is the status o the NVLinkLogicalPartition
	Status string `json:"status"`
	// StatusHistory is the status detail records for the NVLinkLogicalPartition over time
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// Created indicates the ISO datetime string for when the NVLinkLogicalPartition was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the NVLinkLogicalPartition was last updated
	Updated time.Time `json:"updated"`
}

// APINVLinkLogicalPartitionSummary is the data structure to capture API summary of a NVLinkLogicalPartition
type APINVLinkLogicalPartitionSummary struct {
	// ID of the NVLinkLogicalPartition
	ID string `json:"id"`
	// Name of the NVLinkLogicalPartition
	Name string `json:"name"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Status is the status of the NVLinkLogicalPartition
	Status string `json:"status"`
}

// APINVLinkLogicalPartitionStats holds GPU and instance counts for a NVLinkLogicalPartition
type APINVLinkLogicalPartitionStats struct {
	TotalGpus              int `json:"totalGpus"`
	TotalDistinctInstances int `json:"totalDistinctInstances"`
}
