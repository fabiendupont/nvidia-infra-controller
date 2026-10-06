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

// APIInfiniBandPartitionCreateRequest is the data structure to capture instance request to create a new InfiniBandPartition
type APIInfiniBandPartitionCreateRequest struct {
	// Name is the name of the InfiniBand Partition
	Name string `json:"name"`
	// Description is the description of the InfiniBand Partition
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Labels is the labels of the InfiniBand Partition
	Labels map[string]string `json:"labels"`
}

// Validate ensure the values passed in request are acceptable
func (ibpcr APIInfiniBandPartitionCreateRequest) Validate() error {
	err := validation.ValidateStruct(&ibpcr,
		validation.Field(&ibpcr.Name,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		validation.Field(&ibpcr.SiteID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
	)

	if err != nil {
		return err
	}

	return rest.ValidateLabels(ibpcr.Labels)
}

// APIInfiniBandPartitionUpdateRequest is the data structure to capture user request to update a InfiniBandPartition
type APIInfiniBandPartitionUpdateRequest struct {
	// Name is the name of the InfiniBand Partition
	Name *string `json:"name"`
	// Description is the description of the InfiniBand Partition
	Description *string `json:"description"`
	// Labels is the labels of the InfiniBand Partition
	Labels map[string]string `json:"labels"`
}

// Validate ensure the values passed in request are acceptable
func (ibpur APIInfiniBandPartitionUpdateRequest) Validate() error {
	err := validation.ValidateStruct(&ibpur,
		validation.Field(&ibpur.Name,
			validation.When(ibpur.Name != nil, validation.Required.Error(rest.ValidationErrorStringLength)),
			validation.When(ibpur.Name != nil, validation.By(rest.ValidateNameCharacters)),
			validation.When(ibpur.Name != nil, validation.Length(2, 256).Error(rest.ValidationErrorStringLength))),
	)

	if err != nil {
		return err
	}

	return rest.ValidateLabels(ibpur.Labels)
}

// APIInfiniBandPartition is the data structure to capture API representation of a InfiniBand Partition
type APIInfiniBandPartition struct {
	// ID is the unique UUID v4 identifier for the InfiniBand Partition
	ID string `json:"id"`
	// Name is the name of the InfiniBand Partition
	Name string `json:"name"`
	// Description is the description of the InfiniBand Partition
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// Controller IB Partition ID is the ID of the Site Controller IB partition
	ControllerIBPartitionID *string `json:"controllerIBPartitionId"`
	// Partition Key is the key of IB partition
	PartitionKey *string `json:"partitionKey"`
	// Partition Name is the name of IB partition
	PartitionName *string `json:"partitionName"`
	// Service Level is the service level of IB partition
	ServiceLevel *int `json:"serviceLevel"`
	// Rate Limit is the rate limit of IB partition
	RateLimit *float32 `json:"rateLimit"`
	// Mtu of the IB partition
	Mtu *int `json:"mtu"`
	// EnableSharp indicates if sharp enable on the IB partition or not
	EnableSharp *bool `json:"enableSharp"`
	// Labels is the labels of the InfiniBand Partition
	Labels map[string]string `json:"labels"`
	// Status is the status o the InfiniBand Partition
	Status string `json:"status"`
	// StatusHistory is the status detail records for the InfiniBand Partition over time
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// Created indicates the ISO datetime string for when the InfiniBand Partition was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the InfiniBand Partition was last updated
	Updated time.Time `json:"updated"`
}

// APIInfiniBandPartitionSummary is the data structure to capture API summary of a InfiniBandPartition
type APIInfiniBandPartitionSummary struct {
	// ID of the InfiniBand Partition
	ID string `json:"id"`
	// Name of the InfiniBand Partition
	Name string `json:"name"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Controller IB Partition is the ID of the Site Controller Partition corresponding to the InfiniBand Partition
	ControllerIBPartitionID *string `json:"controllerIBPartitionId"`
	// Status is the status of the InfiniBand Partition
	Status string `json:"status"`
}
