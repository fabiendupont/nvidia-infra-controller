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

package health

import (
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
)

// APIFabric is the data structure to capture API representation of a Fabric
type APIFabric struct {
	// ID is the guid identifier for the Fabric
	ID string `json:"id"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// InfrastructureProviderID is the ID of the InfrastructureProvider
	InfrastructureProviderID string `json:"infrastructureProviderId"`
	// InfrastructureProvider is the summary of the InfrastructureProvider
	InfrastructureProvider *rest.APIInfrastructureProviderSummary `json:"infrastructureProvider,omitempty"`
	// Status represents the status of the machine
	Status string `json:"status"`
	// StatusHistory is the history of statuses for the Fabric
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// CreatedAt indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// UpdatedAt indicates the ISO datetime string for when the entity was last updated
	Updated time.Time `json:"updated"`
}

// APIFabricSummary is the data structure to capture API summary of a Fabric
type APIFabricSummary struct {
	// ID of the Fabric
	ID string `json:"id"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Status is the status of the Fabric
	Status string `json:"status"`
}
