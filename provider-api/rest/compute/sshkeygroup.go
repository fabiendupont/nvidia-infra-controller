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
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// APISSHKeyGroupCreateRequest is the data structure to capture instance request to create a new SSHKeyGroup
type APISSHKeyGroupCreateRequest struct {
	// Name is the name of the SSHKeyGroup
	Name string `json:"name"`
	// Description is the description of the SSHKeyGroup
	Description *string `json:"description"`
	// SiteIDs is a list of Site objects
	SiteIDs []string `json:"siteIds"`
	// SSHKeyIDs is a list of SSHKeyID objects
	SSHKeyIDs []string `json:"sshKeyIds"`
}

// Validate ensures that the values passed in request are acceptable
func (sgcr APISSHKeyGroupCreateRequest) Validate() error {
	err := validation.ValidateStruct(&sgcr,
		validation.Field(&sgcr.Name,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
	)
	if err != nil {
		return err
	}
	return nil
}

// APISSHKeyGroupUpdateRequest is the data structure to capture user request to update a SSHKeyGroup
type APISSHKeyGroupUpdateRequest struct {
	// Name is the name of the SSHKeyGroup
	Name *string `json:"name"`
	// Description is the description of the SSHKeyGroup
	Description *string `json:"description"`
	// SiteIDs is a list of Site objects
	SiteIDs []string `json:"siteIds"`
	// SSHKeyIDs is a list of SSHKeyID objects
	SSHKeyIDs []string `json:"sshKeyIds"`
	// Version is the keyset version of the SSHKeyGroup
	Version *string `json:"version"`
}

// Validate ensure the values passed in request are acceptable
func (sgur APISSHKeyGroupUpdateRequest) Validate() error {
	return validation.ValidateStruct(&sgur,
		validation.Field(&sgur.Name,
			validation.When(sgur.Name != nil, validation.Required.Error(rest.ValidationErrorStringLength)),
			validation.When(sgur.Name != nil, validation.By(rest.ValidateNameCharacters)),
			validation.When(sgur.Name != nil, validation.Length(2, 256).Error(rest.ValidationErrorStringLength))),
		validation.Field(&sgur.Version,
			validation.Required.Error(rest.ValidationErrorValueRequired)),
	)
}

// APISSHKeyGroup is the data structure to capture API representation of a SSHKeyGroup
type APISSHKeyGroup struct {
	// ID is the unique UUID v4 identifier for the SSHKeyGroup
	ID string `json:"id"`
	// Name is the name of the SSHKeyGroup
	Name string `json:"name"`
	// Description is the description of the SSHKeyGroup
	Description *string `json:"description"`
	// Org is the organization the SSHKeyGroup belongs to
	Org string `json:"org"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// Version is the keyset version for the SSHKeyGroups
	Version *string `json:"version"`
	// Status is the status of the SSHKeyGroups
	Status string `json:"status"`
	// StatusHistory is the status detail records for the SSHKeyGroups over time
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// SSHKeys is the list of sshkeys associated with the sshkey group
	SSHKeys []APISSHKey `json:"sshKeys"`
	// SiteAssociations is the list of sites associated with the sshkey group
	SiteAssociations []APISSHKeyGroupSiteAssociation `json:"siteAssociations"`
	// Created indicates the ISO datetime string for when the SSHKeyGroup was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the SSHKeyGroup was last updated
	Updated time.Time `json:"updated"`
}

// APISSHKeyGroupSummary is the data structure to capture API summary of a SSHKeyGroup
type APISSHKeyGroupSummary struct {
	// ID is the unique UUID v4 identifier for the SSHKeyGroup
	ID string `json:"id"`
	// Name of the Site, only lowercase characters, digits, hyphens and cannot begin/end with hyphen
	Name string `json:"name"`
	// Description is the description of the SSHKeyGroup
	Description *string `json:"description"`
	// Version is the keyset version for the SSHKeyGroups
	Version *string `json:"version"`
	// Status is the status of the site
	Status string `json:"status"`
	// Created indicates the ISO datetime string for when the SSHKeyGroup was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the SSHKeyGroup was last updated
	Updated time.Time `json:"updated"`
}

// APISSHKeyGroupSiteAssociation is the data structure to capture API representation of a SSHKeyGroup site association
type APISSHKeyGroupSiteAssociation struct {
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site"`
	// ControllerKeySetVersion is the version of corresponding keyset on Site
	ControllerKeySetVersion *string `json:"version"`
	// Status is the status of the SSHKeyGroupSiteAssociation
	Status string `json:"status"`
	// Created indicates the ISO datetime string for when the site was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the site was last updated
	Updated time.Time `json:"updated"`
}
