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
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	validationis "github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
)

// VPC network virtualization types
const (
	VpcEthernetVirtualizer = "ETHERNET_VIRTUALIZER"
	VpcFNN                 = "FNN"
)

const (
	APIVpcRoutingProfileExternal           = "external"
	APIVpcRoutingProfileInternal           = "internal"
	APIVpcRoutingProfilePrivilegedInternal = "privileged-internal"
)

var (
	vpcRoutingProfileStartsWithLetterRegexp = regexp.MustCompile(`^[A-Za-z]`)
	vpcRoutingProfileAllowedCharsRegexp     = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
)

var apiVpcRoutingProfileToSiteMap = map[string]string{
	APIVpcRoutingProfileExternal:           "EXTERNAL",
	APIVpcRoutingProfileInternal:           "INTERNAL",
	APIVpcRoutingProfilePrivilegedInternal: "PRIVILEGED_INTERNAL",
}

// NormalizeAPIVpcRoutingProfileForSite converts REST routing profile values to the
// current site-controller wire format when a known mapping exists.
func NormalizeAPIVpcRoutingProfileForSite(routingProfile string) string {
	if mapped, ok := apiVpcRoutingProfileToSiteMap[routingProfile]; ok {
		return mapped
	}
	return routingProfile
}

// APIVpcCreateRequest captures the request data for creating a new VPC
type APIVpcCreateRequest struct {
	// ID is the user-specified UUID of the VPC.
	ID *uuid.UUID `json:"id"`
	// Name is the name of the VPC
	Name string `json:"name"`
	// Description is the description of the VPC
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// NetworkVirtualizationType is a VPC virtualization type
	NetworkVirtualizationType *string `json:"networkVirtualizationType"`
	// Labels is a key value objects
	Labels map[string]string `json:"labels"`
	// NetworkSecurityGroupID is the ID if a desired
	// NSG to attach to the VPC
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
	// NVLinkLogicalPartitionID is the ID of the NVLinkLogicalPartition
	NVLinkLogicalPartitionID *string `json:"nvLinkLogicalPartitionId"`
	// Vni is an optional, explicitly requested VPC VNI.
	// The request will be rejected by the site if the VNI
	// is not within a VNI range allowed for explicit requests.
	Vni *int `json:"vni"`
	// RoutingProfile specifies the routing profile for the VPC.
	// This is only supported when `networkVirtualizationType` is `FNN`, or when
	// `networkVirtualizationType` is omitted and the Site has native networking enabled.
	// This requires the Tenant to have elevated privileges. Current accepted values
	// are `privileged-internal`, `internal`, and `external`.
	RoutingProfile *string `json:"routingProfile"`
}

// Validate ensure the values passed in create request are acceptable
func (ascr APIVpcCreateRequest) Validate() error {
	err := validation.ValidateStruct(&ascr,
		validation.Field(&ascr.Name,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		validation.Field(&ascr.Description,
			validation.When(ascr.Description != nil,
				validation.Length(0, 1024).Error(rest.ValidationErrorDescriptionStringLength)),
		),
		validation.Field(&ascr.RoutingProfile,
			validation.When(ascr.RoutingProfile != nil,
				validation.Length(3, 64).Error("`routingProfile` must contain at least 3 characters and a maximum of 64 characters"),
				validation.Match(vpcRoutingProfileStartsWithLetterRegexp).Error("`routingProfile` must start with a letter"),
				validation.Match(vpcRoutingProfileAllowedCharsRegexp).Error("`routingProfile` may only contain letters, numbers, or dashes"),
			),
		),
		validation.Field(&ascr.SiteID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&ascr.ID,
			validation.When(ascr.ID != nil, validationis.UUID.Error(rest.ValidationErrorInvalidUUID))),
	)

	if err != nil {
		return err
	}

	// NetworkVirtualizationType validation
	if ascr.NetworkVirtualizationType != nil {
		if (*ascr.NetworkVirtualizationType != VpcEthernetVirtualizer) && (*ascr.NetworkVirtualizationType != VpcFNN) {
			return validation.Errors{
				"networkVirtualizationType": errors.New("either ETHERNET_VIRTUALIZER or FNN are currently supported"),
			}
		}
	}

	if ascr.RoutingProfile != nil {
		if _, ok := apiVpcRoutingProfileToSiteMap[*ascr.RoutingProfile]; !ok {
			return validation.Errors{
				"routingProfile": fmt.Errorf("`routingProfile` must be one of %s, %s, or %s", APIVpcRoutingProfilePrivilegedInternal, APIVpcRoutingProfileInternal, APIVpcRoutingProfileExternal),
			}
		}

		if ascr.NetworkVirtualizationType != nil && *ascr.NetworkVirtualizationType != VpcFNN {
			return validation.Errors{
				"routingProfile": errors.New("`routingProfile` is only supported when `networkVirtualizationType` is FNN"),
			}
		}
	}

	if ascr.Vni != nil && (*ascr.Vni < 0 || *ascr.Vni > math.MaxUint16) {
		return validation.Errors{
			"vni": fmt.Errorf("VNI must be an integer between 0 and %d", math.MaxUint16),
		}
	}

	if err := rest.ValidateLabels(ascr.Labels); err != nil {
		return err
	}

	return err
}

// APIVpcUpdateRequest captures the request data for updating a new VPC
type APIVpcUpdateRequest struct {
	// Name is the name of the VPC
	Name *string `json:"name"`
	// Description is the description of the VPC
	Description *string `json:"description"`
	// Labels is a key value objects
	Labels map[string]string `json:"labels"`
	// NetworkSecurityGroupID is the ID if a desired
	// NSG to attach to the VPC
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
	// NVLinkLogicalPartitionID is the ID of the NVLinkLogicalPartition
	NVLinkLogicalPartitionID *string `json:"nvLinkLogicalPartitionId"`
}

// Validate ensure the values passed in update request are acceptable
func (asur APIVpcUpdateRequest) Validate() error {
	err := validation.ValidateStruct(&asur,
		validation.Field(&asur.Name,
			validation.When(asur.Name != nil, validation.Required.Error(rest.ValidationErrorStringLength)),
			validation.When(asur.Name != nil, validation.By(rest.ValidateNameCharacters)),
			validation.When(asur.Name != nil, validation.Length(2, 256).Error(rest.ValidationErrorStringLength))),
		validation.Field(&asur.Description,
			validation.When(asur.Description != nil, validation.Length(0, 1024).Error(rest.ValidationErrorDescriptionStringLength)),
		),
	)

	if err != nil {
		return err
	}

	if err := rest.ValidateLabels(asur.Labels); err != nil {
		return err
	}

	return err
}

// APIVpcVirtualizationUpdateRequest captures the request data for updating virtualization type for a given VPC
type APIVpcVirtualizationUpdateRequest struct {
	// NetworkVirtualizationType is a VPC virtualization type
	NetworkVirtualizationType string `json:"networkVirtualizationType"`
}

// APIVpc is a data structure to capture information about VPC at the API layer
type APIVpc struct {
	// ID is the unique UUID v4 identifier of the VPC in Forge Cloud
	ID string `json:"id"`
	// Name is the name of the VPC
	Name string `json:"name"`
	// Description is the description of the VPC
	Description *string `json:"description"`
	// Org is the NGC organization ID of the infrastructure provider and the org the VPC belongs to
	Org string `json:"org"`
	// InfrastructureProviderID is the ID of the infrastructure provider who owns the site
	InfrastructureProviderID *string `json:"infrastructureProviderId"`
	// InfrastructureProvider is the summary of the InfrastructureProvider
	InfrastructureProvider *rest.APIInfrastructureProviderSummary `json:"infrastructureProvider,omitempty"`
	// TenantID is the ID of the Tenant
	TenantID *string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// SiteID is the ID of the Site
	SiteID *string `json:"siteId"`
	// Site is the summary of the site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// NetworkVirtualizationType is a VPC virtualization type
	NetworkVirtualizationType *string `json:"networkVirtualizationType"`
	// ControllerVpcID is the ID of the corresponding VPC in Site Controller
	ControllerVpcID *string `json:"controllerVpcId"`
	// Labels is VPC labels specified by user
	Labels map[string]string `json:"labels"`
	// NVLinkLogicalPartitionID is the ID of the NVLinkLogicalPartition
	NVLinkLogicalPartitionID *string `json:"nvLinkLogicalPartitionId"`
	// NVLinkLogicalPartitionSummary is the summary of the NVLinkLogicalPartition
	NVLinkLogicalPartitionSummary *APINVLinkLogicalPartitionSummary `json:"nvLinkLogicalPartitionSummary,omitempty"`
	// NetworkSecurityGroupID is the ID of attached NSG, if any
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
	// NetworkSecurityGroup holds the summary for attached NSG, if requested via includeRelation
	NetworkSecurityGroup *APINetworkSecurityGroupSummary `json:"networkSecurityGroup,omitempty"`
	// NetworkSecurityGroupPropagationDetails is the propagation details for the attched NSG, if any
	NetworkSecurityGroupPropagationDetails *APINetworkSecurityGroupPropagationDetails `json:"networkSecurityGroupPropagationDetails"`
	// RoutingProfile is the applied routing profile for the VPC, when known.
	RoutingProfile *string `json:"routingProfile"`
	// RequestedVni is the explicitly requested VPC VNI at creation time _if_ one was requested.
	RequestedVni *int `json:"requestedVni"`
	// Vni is the active/actual VNI of the VPC, regardless of whether it was
	// explicitly requested or auto-allocated.
	Vni *int `json:"vni"`
	// Status is the status of the VPC
	Status string `json:"status"`
	// StatusHistory is the status detail records for the VPC over time
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// CreatedAt indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the VPC was last updated
	Updated time.Time `json:"updated"`
}

// APIVpcStats is a data structure to capture information about VPC stats at the API layer
type APIVpcStats struct {
	// Total is the total number of the VPC object in Forge Cloud
	Total int `json:"total"`
	// Pending is the total number of pending VPC object in Forge Cloud
	Pending int `json:"pending"`
	// Provisioning is the total number of provisioning VPC object in Forge Cloud
	Provisioning int `json:"provisioning"`
	// Ready is the total number of ready VPC object in Forge Cloud
	Ready int `json:"ready"`
	// Deleting is the total number of deleting VPC object in Forge Cloud
	Deleting int `json:"deleting"`
	// Error is the total number of error VPC object in Forge Cloud
	Error int `json:"error"`
}

// APIVpcSummary is the data structure to capture API representation of a Vpc Summary
type APIVpcSummary struct {
	// ID is the unique UUID v4 identifier of the VPC in Forge Cloud
	ID string `json:"id"`
	// Name of the Vpc, only lowercase characters, digits, hyphens and cannot begin/end with hyphen
	Name string `json:"name"`
	// ControllerVpcID is the ID of the corresponding VPC in Site Controller
	ControllerVpcID *string `json:"controllerVpcId"`
	// Network virtualization type is a VPC virtualization type
	NetworkVirtualizationType *string `json:"networkVirtualizationType"`
	// Status is the status of the VPC
	Status string `json:"status"`
}
