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
	"errors"
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	validationis "github.com/go-ozzo/ozzo-validation/v4/is"
)

// APIVpcPeeringCreateRequest captures the request data for creating a new VPC peering
type APIVpcPeeringCreateRequest struct {
	// The order of VPCs is not important, the VPC peering is bidirectional.
	// Vpc1ID is the ID of one VPC in the peering
	Vpc1ID string `json:"vpc1Id"`
	// Vpc2ID is the ID of the other VPC in the peering
	Vpc2ID string `json:"vpc2Id"`
	// SiteID is the ID of the Site where the peering exists
	SiteID string `json:"siteId"`
}

// Validate ensures the values passed in create request are acceptable
func (vpcr APIVpcPeeringCreateRequest) Validate() error {
	err := validation.ValidateStruct(&vpcr,
		validation.Field(&vpcr.Vpc1ID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&vpcr.Vpc2ID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&vpcr.SiteID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
	)
	if err != nil {
		return err
	}

	// Validate that the VPCs are different
	if vpcr.Vpc1ID == vpcr.Vpc2ID {
		return validation.Errors{
			"vpc2Id": errors.New("Cannot be the same value as `vpc1Id`"),
		}
	}
	return nil
}

// APIVpcPeering represents a VPC peering connection
type APIVpcPeering struct {
	// ID is the unique UUID v4 identifier of the VPC peering in Forge Cloud
	ID string `json:"id"`
	// Vpc1ID is the ID of the first VPC in the peering
	Vpc1ID string `json:"vpc1Id"`
	// Vpc1 is the summary of the first VPC in the peering
	Vpc1 *APIVpcSummary `json:"vpc1,omitempty"`
	// Vpc2ID is the ID of the second VPC in the peering
	Vpc2ID string `json:"vpc2Id"`
	// Vpc2 is the summary of the second VPC in the peering
	Vpc2 *APIVpcSummary `json:"vpc2,omitempty"`
	// SiteID is the ID of the Site where the peering exists
	SiteID string `json:"siteId"`
	// Site is the summary of the site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// IsMultiTenant indicates if this is a multi-tenant peering
	IsMultiTenant bool `json:"isMultiTenant"`
	// Status is the status of the VPC peering
	Status string `json:"status"`
	// CreatedAt indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the VPC peering was last updated
	Updated time.Time `json:"updated"`
}

// APIVpcPeeringSummary represents a summary of a VPC peering connection
type APIVpcPeeringSummary struct {
	// ID is the unique UUID v4 identifier of the VPC peering in Forge Cloud
	ID string `json:"id"`
	// Vpc1ID is the ID of the first VPC in the peering
	Vpc1ID string `json:"vpc1Id"`
	// Vpc2ID is the ID of the second VPC in the peering
	Vpc2ID string `json:"vpc2Id"`
	// IsMultiTenant indicates if this is a multi-tenant peering
	IsMultiTenant bool `json:"isMultiTenant"`
	// Status is the status of the VPC peering
	Status string `json:"status"`
}
