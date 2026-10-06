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
)

const MaxNetworkSecurityGroupRules = 200
const NetworkSecurityGroupRulePriorityMin = 0
const NetworkSecurityGroupRulePriorityMax = 60000

// Action constants
const (
	APINetworkSecurityGroupRuleActionPermit = "PERMIT"
	APINetworkSecurityGroupRuleActionDeny   = "DENY"
)

// Direction constants
const (
	APINetworkSecurityGroupRuleDirectionIngress = "INGRESS"
	APINetworkSecurityGroupRuleActionEgress     = "EGRESS"
)

// Protocol constants
const (
	APINetworkSecurityGroupRuleProtocolAny   = "ANY"
	APINetworkSecurityGroupRuleProtocolIcmp  = "ICMP"
	APINetworkSecurityGroupRuleProtocolIcmp6 = "ICMP6"
	APINetworkSecurityGroupRuleProtocolTcp   = "TCP"
	APINetworkSecurityGroupRuleProtocolUdp   = "UDP"
)

// Propagation status constants
const (
	APINetworkSecurityGroupPropagationDetailedStatusNone    = "None"
	APINetworkSecurityGroupPropagationDetailedStatusPartial = "Partial"
	APINetworkSecurityGroupPropagationDetailedStatusFull    = "Full"
	APINetworkSecurityGroupPropagationDetailedStatusUnknown = "Unknown"
	APINetworkSecurityGroupPropagationDetailedStatusError   = "Error"

	APINetworkSecurityGroupPropagationStatusError         = "Error"
	APINetworkSecurityGroupPropagationStatusSynchronizing = "Synchronizing"
	APINetworkSecurityGroupPropagationStatusSynchronized  = "Synchronized"
)

// APINetworkSecurityGroupCreateRequest is the data structure to capture instance request to create a new NetworkSecurityGroup
type APINetworkSecurityGroupCreateRequest struct {
	// Name is the name of the NetworkSecurityGroup
	Name string `json:"name"`
	// Description is the description of the NetworkSecurityGroup
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Rules is the list of NetworkSecurityGroupRuleAttributes for the NetworkSecurityGroup
	Rules []APINetworkSecurityGroupRule `json:"rules"`
	// StatefulEgress defines whether a NetworkSecurityGroup's egress rules will be automatically stateful
	StatefulEgress bool `json:"statefulEgress"`
	// Labels to be associted with the NetworkSecurityGroup
	Labels map[string]string `json:"labels"`
}

// APINetworkSecurityGroupUpdateRequest is the data structure to capture user request to update a NetworkSecurityGroup
type APINetworkSecurityGroupUpdateRequest struct {
	// Name is the name of the NetworkSecurityGroup
	Name *string `json:"name"`
	// Description is the description of the NetworkSecurityGroup
	Description *string `json:"description"`
	// StatefulEgress defines whether a NetworkSecurityGroup's egress rules will be automatically stateful
	StatefulEgress *bool `json:"statefulEgress"`
	// Rules is the list of NetworkSecurityGroupRuleAttributes for the NetworkSecurityGroup
	Rules []APINetworkSecurityGroupRule `json:"rules"`
	// Labels to be associted with the NetworkSecurityGroup
	Labels map[string]string `json:"labels"`
}

// APINetworkSecurityGroup is the data structure to capture API representation of a NetworkSecurityGroup
type APINetworkSecurityGroup struct {
	// ID is the unique UUID v4 identifier for the NetworkSecurityGroup
	ID string `json:"id"`
	// Name is the name of the NetworkSecurityGroup
	Name string `json:"name"`
	// Description is the description of the NetworkSecurityGroup
	Description *string `json:"description"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// Status is the status of the NetworkSecurityGroup
	Status string `json:"status"`
	// StatusHistory is the status detail records for the site over time
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// StatefulEgress defines whether a NetworkSecurityGroup's egress rules will be automatically stateful
	StatefulEgress bool `json:"statefulEgress"`
	// Rules is the list of NetworkSecurityGroupRuleAttributes for the NetworkSecurityGroup
	Rules []*APINetworkSecurityGroupRule `json:"rules"`
	// Labels is the set of labels/tags for the NetworkSecurityGroup
	Labels map[string]string `json:"labels"`
	// Created indicates the ISO datetime string for when the NetworkSecurityGroup was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the NetworkSecurityGroup was last updated
	Updated time.Time `json:"updated"`
	// AttachmentStats holds the counts for objects that have
	// Attached the NSG if requested.
	AttachmentStats *APINetworkSecurityGroupStats `json:"attachmentStats"`
	// RuleCount hold the count of the number of rules in the NetworkSecurityGroup
	RuleCount int `json:"ruleCount"`
}

// APINetworkSecurityGroupRule is the data structure for a single NSG rule
type APINetworkSecurityGroupRule struct {
	Name                 *string `json:"name"`
	Direction            string  `json:"direction"`
	SourcePortRange      *string `json:"sourcePortRange"`
	DestinationPortRange *string `json:"destinationPortRange"`
	Protocol             string  `json:"protocol"`
	Action               string  `json:"action"`
	Priority             int     `json:"priority"`
	SourcePrefix         *string `json:"sourcePrefix"`
	DestinationPrefix    *string `json:"destinationPrefix"`
}

// APINetworkSecurityGroupStats holds detailed usage stats for an NSG
type APINetworkSecurityGroupStats struct {
	// InUse is a convenience field that will be true
	// if TotalAttachmentCount > 0
	InUse bool `json:"inUse"`
	// VpcAttachmentCount holds the count of the number of VPCs that have the NSG directly attached.
	VpcAttachmentCount int `json:"directVpcAttachmentCount"`
	// InstanceAttachmentCount holds the count of the number of instances that have the NSG directly attached.
	InstanceAttachmentCount int `json:"directInstanceAttachmentCount"`
	// TotalAttachmentCount holds the total count of all objects that have
	// the NSG directly attached.
	TotalAttachmentCount int `json:"totalDirectAttachmentCount"`
}

// APINetworkSecurityGroupSummary is the data structure to capture API summary of a NetworkSecurityGroup
type APINetworkSecurityGroupSummary struct {
	// ID of the NetworkSecurityGroup
	ID string `json:"id"`
	// Name of the NetworkSecurityGroup
	Name string `json:"name"`
	// Description of the NetworkSecurityGroup
	Description *string `json:"description"`
	// Status is the status of the NetworkSecurityGroup
	Status string `json:"status"`
	// StatefulEgress defines whether a NetworkSecurityGroup's egress rules will be automatically stateful
	StatefulEgress bool `json:"statefulEgress"`
	// RuleCount hold the count of the number of rules in the NetworkSecurityGroup
	RuleCount int `json:"ruleCount"`
}

// APINetworkSecurityGroupPropagationDetails holds propagation status details
type APINetworkSecurityGroupPropagationDetails struct {
	// The ID of the object (VPC/Instance/etc) for these details
	ObjectID string `json:"object_id"`
	// The detailed propagation status that was
	// actually returned from Carbide
	DetailedStatus string `json:"detailedStatus"`
	// The simplified propagation status
	// that reduces the actual status to just
	// a few values.
	Status string `json:"status"`
	// Additional details for the status
	Details *string `json:"details"`
	// IDs of the instances involved in determining the
	// propagation status
	RelatedInstanceIds []string `json:"relatedInstanceIds"`
	// IDs of any instances associated with the ObjectID that have
	// not yet updated their NSG rules.
	UnpropagatedInstanceIds []string `json:"unpropagatedInstanceIds"`
}
