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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	"github.com/NVIDIA/infra-controller/provider-api/rest/networking"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	validationis "github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	// MaxInterfaceCount is the maximum number of Interfaces allowed per Instance
	MaxInterfaceCount = 16

	// MachineIssueCategoryHardware is the category for hardware issues
	MachineIssueCategoryHardware = "Hardware"
	// MachineIssueCategoryNetwork is the category for network issues
	MachineIssueCategoryNetwork = "Network"
	// MachineIssueCategoryPerformance is the category for performance issues
	MachineIssueCategoryPerformance = "Performance"
	// MachineIssueCategoryOther is the category for other issues
	MachineIssueCategoryOther = "Other"
)

// ValidateInterfaces validates the Interfaces for the Instance
func ValidateInterfaces(ifcs *[]networking.APIInterfaceCreateOrUpdateRequest) error {
	// Validate Interfaces
	vpcPrefixInterfaceCount := 0
	subnetInterfaceCount := 0

	multiEthernetInterfaceCount := 0
	singleEthernetInterfaceCount := 0

	physicalInterfaceCount := 0

	for _, ifcr := range *ifcs {
		err := ifcr.Validate()

		if err != nil {
			return err
		}

		if ifcr.VpcPrefixID != nil {
			vpcPrefixInterfaceCount++
		} else {
			subnetInterfaceCount++
		}

		if ifcr.IsMultiEthernetInterface() {
			multiEthernetInterfaceCount++
		} else {
			singleEthernetInterfaceCount++
		}

		if ifcr.IsPhysical {
			physicalInterfaceCount++
		}
	}

	if vpcPrefixInterfaceCount > 0 && subnetInterfaceCount > 0 {
		return validation.Errors{
			rest.ValidationCommonErrorField: errors.New("either all interfaces must be VPC Prefix based or all of them must be Subnet based"),
		}
	}

	if multiEthernetInterfaceCount > 0 && singleEthernetInterfaceCount > 0 {
		return validation.Errors{
			rest.ValidationCommonErrorField: errors.New("either all interfaces must specify device/deviceInstance or none of them should specify those fields"),
		}
	}

	if singleEthernetInterfaceCount > 0 {
		if physicalInterfaceCount > 1 {
			return validation.Errors{
				"interface": errors.New("only one interface can be marked as physical for single-Ethernet interfaces"),
			}
		} else if physicalInterfaceCount == 0 {
			// Set first interface as physical if none of the interface is marked as physical
			(*ifcs)[0].IsPhysical = true
		}
	}

	return nil
}

// ValidateDpuExtensionServiceDeployments validates the DpuExtensionServiceDeployments for the Instance create/update request
func ValidateDpuExtensionServiceDeployments(desdrs []networking.APIDpuExtensionServiceDeploymentRequest) error {
	for _, desdr := range desdrs {
		err := desdr.Validate()
		if err != nil {
			return err
		}
	}

	desVersionMap := map[string]bool{}
	for _, desdr := range desdrs {
		desvID := fmt.Sprintf("%s:%s", desdr.DpuExtensionServiceID, desdr.Version)
		_, exists := desVersionMap[desvID]
		if exists {
			return validation.Errors{
				"dpuExtensionServiceDeployments": fmt.Errorf("duplicate deployment requests found for DPU Extension Service ID and version: %s", desvID),
			}
		}
		desVersionMap[desvID] = true
	}

	return nil
}

// APIInstanceCreateRequest is the data structure to capture request to create a new Instance
type APIInstanceCreateRequest struct {
	// Name is the name of the Instance
	Name string `json:"name"`
	// Description is the description of the Instance
	Description *string `json:"description"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// InstanceTypeID is the ID of the Instance Type. Only InstanceTypeID or MachineID can be present
	InstanceTypeID *string `json:"instanceTypeId"`
	// VpcID is the ID of the VPC containing the Instance
	VpcID string `json:"vpcId"`
	// SecondaryVpcIDs lists additional VPC UUIDs for prefix-backed, non-primary
	// network interfaces on the Instance.
	SecondaryVpcIDs []string `json:"secondaryVpcIds"`
	// OperatingSystemID is the ID of the Operating System
	OperatingSystemID *string `json:"operatingSystemId"`
	// IpxeScript is the iPXE script for the Operating System
	IpxeScript *string `json:"ipxeScript"`
	// AlwaysBootWithCustomIpxe is the flag to allow always boot with ipxe
	AlwaysBootWithCustomIpxe *bool `json:"alwaysBootWithCustomIpxe"`
	// PhoneHomeEnabled is the flag to allow enable phone home for the instance
	PhoneHomeEnabled *bool `json:"phoneHomeEnabled"`
	// UserData is the ID of the Operating System
	UserData *string `json:"userData"`
	// Interfaces is the list of Interfaces to create for the Instance
	Interfaces []networking.APIInterfaceCreateOrUpdateRequest `json:"interfaces"`
	// InfiniBandInterfaces is the list of InfiniBandInterface to create for the Instance
	InfiniBandInterfaces []networking.APIInfiniBandInterfaceCreateOrUpdateRequest `json:"infinibandInterfaces"`
	// DpuExtensionServiceDeployments is the list of DpuExtensionServiceDeployments to create for the Instance
	DpuExtensionServiceDeployments []networking.APIDpuExtensionServiceDeploymentRequest `json:"dpuExtensionServiceDeployments"`
	// NVLinkInterfaces is the list of NVLinkInterface to create for the Instance
	NVLinkInterfaces []networking.APINVLinkInterfaceCreateOrUpdateRequest `json:"nvLinkInterfaces"`
	// SSHKeyGroupIDs is a list of SSHKeyID objects
	SSHKeyGroupIDs []string `json:"sshKeyGroupIds"`
	// Labels is a key value objects
	Labels map[string]string `json:"labels"`
	// NetworkSecurityGroupID is the ID if a desired
	// NSG to attach to the instance
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
	// MachineID is the ID of the Machine. Only MachineID or InstanceTypeID can be present
	MachineID *string `json:"machineId"`
	// AllowUnhealthyMachine is a flag that can be used to target Machines are in maintenance or have health alerts preventing regular provision flow.
	AllowUnhealthyMachine *bool `json:"allowUnhealthyMachine"`
}

// Validate ensure the values passed in request are acceptable
func (icr APIInstanceCreateRequest) Validate() error {
	err := validation.ValidateStruct(&icr,
		validation.Field(&icr.Name,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		validation.Field(&icr.Description,
			validation.When(icr.Description != nil,
				validation.Length(0, 1024).Error(rest.ValidationErrorDescriptionStringLength)),
		),
		validation.Field(&icr.TenantID,
			validation.When(icr.TenantID != "", validationis.UUID.Error(rest.ValidationErrorInvalidUUID))),
		validation.Field(&icr.InstanceTypeID,
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&icr.VpcID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&icr.OperatingSystemID,
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&icr.Interfaces,
			validation.Required.Error("at least one Interface must be specified"),
			validation.Length(1, MaxInterfaceCount).Error(fmt.Sprintf("at most %v Interfaces can be specified", MaxInterfaceCount))),
	)

	if err != nil {
		return err
	}

	if icr.SecondaryVpcIDs != nil {
		for _, iface := range icr.Interfaces {
			if iface.VpcPrefixID == nil {
				return validation.Errors{
					"secondaryVpcIds": errors.New("`secondaryVpcIds` can only be specified when `vpcPrefixId` is specified within `interfaces`"),
				}
			}
		}
	}

	// ensure we have one and only one of InstanceTypeID or MachineID
	if icr.InstanceTypeID != nil && icr.MachineID != nil {
		return validation.Errors{"machineId": errors.New("only one of `instanceTypeId` or `machineId` can be specified in request, not both")}
	} else if icr.InstanceTypeID == nil && icr.MachineID == nil {
		return validation.Errors{"instanceTypeId": errors.New("either `instanceTypeId` or `machineId` must be specified")}
	}

	// make sure either of them provided
	if icr.OperatingSystemID == nil {
		if icr.IpxeScript == nil {
			return validation.Errors{
				"operatingSystemId": errors.New("either `operatingSystemId` or `ipxeScript` must be specified"),
			}
		} else if *icr.IpxeScript == "" {
			return validation.Errors{
				"ipxeScript": errors.New("cannot be empty when `operatingSystemId` is not specified"),
			}
		}
	}

	// Validate Interfaces
	err = ValidateInterfaces(&icr.Interfaces)
	if err != nil {
		return err
	}

	// Validate InfiniBand Interfaces
	for _, ibic := range icr.InfiniBandInterfaces {
		err = ibic.Validate()
		if err != nil {
			return err
		}
	}

	// Validate DpuExtensionServiceDeployments
	err = ValidateDpuExtensionServiceDeployments(icr.DpuExtensionServiceDeployments)
	if err != nil {
		return err
	}

	// Validate NVLink interfaces
	for _, nvlifc := range icr.NVLinkInterfaces {
		err = nvlifc.Validate()
		if err != nil {
			return err
		}
	}

	if err := rest.ValidateLabels(icr.Labels); err != nil {
		return err
	}

	return err
}

// APIBatchInstanceCreateRequest is the data structure to capture request to create multiple instances in a single request
// with rack-aware allocation logic to place instances on the same rack when possible
type APIBatchInstanceCreateRequest struct {
	// NamePrefix is the prefix for instance names (e.g., "worker" will create "worker-1", "worker-2", etc.)
	NamePrefix string `json:"namePrefix"`
	// Count is the number of instances to create
	Count int `json:"count"`
	// Description is the description for all instances
	Description *string `json:"description"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// InstanceTypeID is the ID of the Instance Type
	InstanceTypeID string `json:"instanceTypeId"`
	// VpcID is the ID of the VPC containing the Instances
	VpcID string `json:"vpcId"`
	// SecondaryVpcIDs lists additional VPC UUIDs for prefix-backed, non-primary
	// network interfaces on each Instance in the batch.
	SecondaryVpcIDs []string `json:"secondaryVpcIds"`
	// OperatingSystemID is the ID of the Operating System
	OperatingSystemID *string `json:"operatingSystemId"`
	// IpxeScript is the iPXE script for the Operating System
	IpxeScript *string `json:"ipxeScript"`
	// AlwaysBootWithCustomIpxe is the flag to allow always boot with ipxe
	AlwaysBootWithCustomIpxe *bool `json:"alwaysBootWithCustomIpxe"`
	// PhoneHomeEnabled is the flag to allow enable phone home for the instance
	PhoneHomeEnabled *bool `json:"phoneHomeEnabled"`
	// UserData is the user data for the instances
	UserData *string `json:"userData"`
	// Interfaces is the list of Interfaces to create for each instance (shared across all instances)
	Interfaces []networking.APIInterfaceCreateOrUpdateRequest `json:"interfaces"`
	// InfiniBandInterfaces is the list of InfiniBandInterface to create for each instance (shared across all instances)
	InfiniBandInterfaces []networking.APIInfiniBandInterfaceCreateOrUpdateRequest `json:"infinibandInterfaces"`
	// NVLinkInterfaces is the list of NVLinkInterface to create for each instance (shared across all instances)
	NVLinkInterfaces []networking.APINVLinkInterfaceCreateOrUpdateRequest `json:"nvLinkInterfaces"`
	// DpuExtensionServiceDeployments is the list of DpuExtensionServiceDeployments to create for each Instance (shared across all instances)
	DpuExtensionServiceDeployments []networking.APIDpuExtensionServiceDeploymentRequest `json:"dpuExtensionServiceDeployments"`
	// SSHKeyGroupIDs is a list of SSHKeyGroup IDs (shared across all instances)
	SSHKeyGroupIDs []string `json:"sshKeyGroupIds"`
	// Labels is a key value objects to be applied to all instances (shared across all instances)
	Labels map[string]string `json:"labels"`
	// NetworkSecurityGroupID is the ID of a desired NSG to attach to all instances (shared across all instances)
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
	// TopologyOptimized indicates whether to enforce rack-aware placement
	TopologyOptimized *bool `json:"topologyOptimized"`
}

// Validate ensure the values passed in batch instance create request are acceptable
func (bicr APIBatchInstanceCreateRequest) Validate() error {
	err := validation.ValidateStruct(&bicr,
		validation.Field(&bicr.NamePrefix,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 240).Error("Name prefix must contain at least 2 characters and a maximum of 240 characters")),
		validation.Field(&bicr.Count,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validation.Min(2).Error("Count must be at least 2"),
			validation.Max(18).Error("Count cannot exceed 18")),
		validation.Field(&bicr.Description,
			validation.When(bicr.Description != nil,
				validation.Length(0, 1024).Error(rest.ValidationErrorDescriptionStringLength)),
		),
		validation.Field(&bicr.TenantID,
			validation.When(bicr.TenantID != "", validationis.UUID.Error(rest.ValidationErrorInvalidUUID))),
		validation.Field(&bicr.InstanceTypeID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&bicr.VpcID,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&bicr.OperatingSystemID,
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&bicr.Interfaces,
			validation.Required.Error("at least one Interface must be specified"),
			validation.Length(1, MaxInterfaceCount).Error(fmt.Sprintf("at most %v Interfaces can be specified", MaxInterfaceCount))),
	)

	if err != nil {
		return err
	}

	if bicr.SecondaryVpcIDs != nil {
		for _, iface := range bicr.Interfaces {
			if iface.VpcPrefixID == nil {
				return validation.Errors{
					"secondaryVpcIds": errors.New("`secondaryVpcIds` can only be specified when `vpcPrefixId` is specified within `interfaces`"),
				}
			}
		}
	}

	// Validate that either OperatingSystemID or IpxeScript is specified
	if bicr.OperatingSystemID == nil {
		if bicr.IpxeScript == nil {
			return validation.Errors{
				"operatingSystemId": errors.New("either `operatingSystemId` or `ipxeScript` must be specified"),
			}
		} else if *bicr.IpxeScript == "" {
			return validation.Errors{
				"ipxeScript": errors.New("cannot be empty when `operatingSystemId` is not specified"),
			}
		}
	}

	// Validate Interfaces
	err = ValidateInterfaces(&bicr.Interfaces)
	if err != nil {
		return err
	}
	for _, ifc := range bicr.Interfaces {
		if ifc.IPAddress != nil {
			return validation.Errors{
				"interfaces": errors.New("batch instance create does not support `ipAddress` on interfaces"),
			}
		}
	}

	// Validate InfiniBand Interfaces
	for _, ibic := range bicr.InfiniBandInterfaces {
		err = ibic.Validate()
		if err != nil {
			return err
		}
	}

	// Validate DpuExtensionServiceDeployments
	err = ValidateDpuExtensionServiceDeployments(bicr.DpuExtensionServiceDeployments)
	if err != nil {
		return err
	}

	// Validate NVLink interfaces
	for _, nvlifc := range bicr.NVLinkInterfaces {
		err = nvlifc.Validate()
		if err != nil {
			return err
		}
	}

	if err := rest.ValidateLabels(bicr.Labels); err != nil {
		return err
	}

	// err should be nil at this point
	return err
}

// APIInstanceUpdateRequest is the data structure to capture request to update an Instance
type APIInstanceUpdateRequest struct {
	// Name is the name of the Instance
	Name *string `json:"name"`
	// Description is the description of the Instance
	Description *string `json:"description"`
	// Labels is a key value objects
	Labels map[string]string `json:"labels"`
	// TriggerReboot is the flag to trigger reboot
	TriggerReboot *bool `json:"triggerReboot"`
	// RebootWithCustomIpxe is the flag to allow reboot with ipxe
	RebootWithCustomIpxe *bool `json:"rebootWithCustomIpxe"`
	// ApplyUpdatesOnReboot is the flag to allow update first before reboot
	ApplyUpdatesOnReboot *bool `json:"applyUpdatesOnReboot"`
	// OperatingSystemID is the ID of the Operating System
	OperatingSystemID *string `json:"operatingSystemId"`
	// IpxeScript is the iPXE script for the Operating System
	IpxeScript *string `json:"ipxeScript"`
	// UserData is the user-data to be used when booting; e.g., a cloud-init config
	UserData *string `json:"userData"`
	// PhoneHomeEnabled is an attribute which is specified by user if Instance needs to be enabled for phone home or not
	PhoneHomeEnabled *bool `json:"phoneHomeEnabled"`
	// AlwaysBootWithCustomIpxe is an attribute which is specified by user if instance boot with ipxe or not
	AlwaysBootWithCustomIpxe *bool `json:"alwaysBootWithCustomIpxe"`
	// SecondaryVpcIDs lists additional VPC IDs for prefix-backed, non-primary
	// network interfaces on the Instance.
	SecondaryVpcIDs []string `json:"secondaryVpcIds"`
	// Interfaces is the list of Interfaces to update for the Instance
	Interfaces []networking.APIInterfaceCreateOrUpdateRequest `json:"interfaces"`
	// InfiniBandInterfaces is the list of InfiniBandInterface to update for the Instance
	InfiniBandInterfaces []networking.APIInfiniBandInterfaceCreateOrUpdateRequest `json:"infinibandInterfaces"`
	// DpuExtensionServiceDeployments is the list of DpuExtensionServiceDeployments to update for the Instance
	DpuExtensionServiceDeployments []networking.APIDpuExtensionServiceDeploymentRequest `json:"dpuExtensionServiceDeployments"`
	// NVLinkInterfaces is the list of NVLinkInterface to update for the Instance
	NVLinkInterfaces []networking.APINVLinkInterfaceCreateOrUpdateRequest `json:"nvLinkInterfaces"`
	// SSHKeyGroupIDs is a list of SSHKeyID objects
	SSHKeyGroupIDs []string `json:"sshKeyGroupIds"`
	// NetworkSecurityGroupID is the ID of Network Security Group to attach to the Instance
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
}

// IsUpdateRequest checks if the request is an instance config update request
func (iur *APIInstanceUpdateRequest) IsUpdateRequest() bool {
	return iur.Name != nil ||
		iur.Description != nil ||
		iur.Labels != nil ||
		iur.OperatingSystemID != nil ||
		iur.IpxeScript != nil ||
		iur.UserData != nil ||
		iur.PhoneHomeEnabled != nil ||
		iur.AlwaysBootWithCustomIpxe != nil ||
		iur.SecondaryVpcIDs != nil ||
		iur.Interfaces != nil ||
		iur.InfiniBandInterfaces != nil ||
		iur.NVLinkInterfaces != nil ||
		iur.SSHKeyGroupIDs != nil ||
		iur.NetworkSecurityGroupID != nil
}

// IsInterfaceUpdateRequest checks if the request is an instance interface update request
func (iur *APIInstanceUpdateRequest) IsInterfaceUpdateRequest() bool {
	return iur.Interfaces != nil || iur.InfiniBandInterfaces != nil || iur.NVLinkInterfaces != nil
}

// IsRebootRequest checks if the request is an instance reboot request
func (iur *APIInstanceUpdateRequest) IsRebootRequest() bool {
	return iur.TriggerReboot != nil && *iur.TriggerReboot
}

// Validate ensures the values passed in request are acceptable
func (iur APIInstanceUpdateRequest) Validate() error {
	err := validation.ValidateStruct(&iur,
		validation.Field(&iur.Name,
			// length validation rule accepts empty string as valid, hence, required is needed
			validation.When(iur.Name != nil, validation.Required.Error(rest.ValidationErrorStringLength), validation.By(rest.ValidateNameCharacters), validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		),
		validation.Field(&iur.Description,
			validation.When(iur.Description != nil, validation.Length(0, 1024).Error(rest.ValidationErrorDescriptionStringLength)),
		),
		validation.Field(&iur.OperatingSystemID,
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID),
		),
		validation.Field(&iur.Interfaces,
			validation.When(len(iur.Interfaces) > 0, validation.Length(1, MaxInterfaceCount).Error(fmt.Sprintf("at most %v Interfaces can be specified", MaxInterfaceCount))),
		),
	)

	if err != nil {
		return err
	}

	if iur.SecondaryVpcIDs != nil {
		if len(iur.Interfaces) == 0 {
			return validation.Errors{
				"secondaryVpcIds": errors.New("`secondaryVpcIds` can only be specified when `interfaces` is specified and non-empty"),
			}
		}

		for _, iface := range iur.Interfaces {
			if iface.VpcPrefixID == nil {
				return validation.Errors{
					"secondaryVpcIds": errors.New("`secondaryVpcIds` can only be specified when `vpcPrefixId` is specified within `interfaces`"),
				}
			}
		}
	}

	if iur.IsRebootRequest() && iur.IsUpdateRequest() {
		return validation.Errors{
			"triggerReboot": errors.New("reboot cannot be triggered if Instance attributes are being updated in the same request"),
		}
	}

	if !iur.IsRebootRequest() {
		if iur.RebootWithCustomIpxe != nil && *iur.RebootWithCustomIpxe {
			return validation.Errors{
				"rebootWithCustomIpxe": errors.New("`rebootWithCustomIpxe` can only be specified when `triggerReboot` is specified"),
			}
		}

		if iur.ApplyUpdatesOnReboot != nil && *iur.ApplyUpdatesOnReboot {
			return validation.Errors{
				"applyUpdatesOnReboot": errors.New("`applyUpdatesOnReboot` can only be specified when `triggerReboot` is specified"),
			}
		}
	}

	// Validate Interfaces if provided
	if len(iur.Interfaces) > 0 {
		err = ValidateInterfaces(&iur.Interfaces)
		if err != nil {
			return err
		}
	}

	// Validate InfiniBand Interfaces
	for _, ibifc := range iur.InfiniBandInterfaces {
		err = ibifc.Validate()
		if err != nil {
			return err
		}
	}

	// Validate DpuExtensionServiceDeployments
	err = ValidateDpuExtensionServiceDeployments(iur.DpuExtensionServiceDeployments)
	if err != nil {
		return err
	}

	// Validate NVLink interfaces
	for _, nvlic := range iur.NVLinkInterfaces {
		err = nvlic.Validate()
		if err != nil {
			return err
		}
	}

	if err := rest.ValidateLabels(iur.Labels); err != nil {
		return err
	}

	return err
}

// APIInstanceDeleteRequest is the data structure to capture request to delete an Instance
type APIInstanceDeleteRequest struct {
	// MachineHealthIssue is the report of a machine health issue
	MachineHealthIssue *APIMachineHealthIssueReport `json:"machineHealthIssue"`
	IsRepairTenant     *bool                        `json:"isRepairTenant"`
}

// APIMachineHealthIssueReport is the data structure to capture a machine health issue report
type APIMachineHealthIssueReport struct {
	// Category is the type of the issue
	Category string `json:"category"`
	// Summary is the summary of the issue
	Summary *string `json:"summary"`
	// Details is the message of the issue
	Details *string `json:"details"`
}

// Validate ensures the values passed in request are acceptable
func (idr APIInstanceDeleteRequest) Validate() error {
	if idr.MachineHealthIssue != nil {
		err := validation.ValidateStruct(idr.MachineHealthIssue,
			validation.Field(&idr.MachineHealthIssue.Category,
				validation.Required,
				validation.In(MachineIssueCategoryHardware, MachineIssueCategoryNetwork, MachineIssueCategoryPerformance, MachineIssueCategoryOther),
			),
			validation.Field(&idr.MachineHealthIssue.Summary,
				validation.Required,
				validation.Length(0, 1024).Error(rest.ValidationErrorStringLength)),
			validation.Field(&idr.MachineHealthIssue.Details,
				validation.Length(0, 1024).Error(rest.ValidationErrorStringLength)),
		)

		if err != nil {
			return err
		}
	}

	return nil
}

// SSHKeyGroupsSummaryDeprecated ensures we keep returning empty array until deprecation time even with omitempty
type SSHKeyGroupsSummaryDeprecated struct {
	SSHKeyGroups []APISSHKeyGroupSummary
}

// MarshalJSON provides custom JSON marshaling for SSHKeyGroupsSummaryDeprecated
func (skgsd *SSHKeyGroupsSummaryDeprecated) MarshalJSON() ([]byte, error) {
	return json.Marshal(skgsd.SSHKeyGroups)
}

// UnmarshalJSON provides custom JSON unmarshaling for SSHKeyGroupsSummaryDeprecated
func (skgsd *SSHKeyGroupsSummaryDeprecated) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &skgsd.SSHKeyGroups)
}

// APIInstance is the data structure to capture API representation of an Instance
type APIInstance struct {
	// ID is the unique UUID v4 identifier for the Instance
	ID string `json:"id"`
	// Name of the Instance
	Name string `json:"name"`
	// Description is the description of the Instance
	Description *string `json:"description"`
	// ControllerInstanceID is the ID of the Instance in Site Controller
	ControllerInstanceID string `json:"controllerInstanceId"`
	// TenantID is the ID of the Tenant
	TenantID string `json:"tenantId"`
	// Tenant is the summary of the tenant
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// InfrastructureProviderID is the ID of the Infrastructure Provider
	InfrastructureProviderID string `json:"infrastructureProviderId"`
	// InfrastructureProvider is the summary of the Infrastructure Provider
	InfrastructureProvider *rest.APIInfrastructureProviderSummary `json:"infrastructureProvider,omitempty"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// InstanceTypeID is the ID of the InstanceType
	InstanceTypeID *string `json:"instanceTypeId"`
	// InstanceType is the summary of the InstanceType
	InstanceType *APIInstanceTypeSummary `json:"instanceType,omitempty"`
	// VpcID is the ID of the VPC
	VpcID string `json:"vpcId"`
	// Vpc is the summary of the VPC
	Vpc *networking.APIVpcSummary `json:"vpc,omitempty"`
	// SecondaryVpcIDs lists non-primary VPC UUIDs derived from prefix-backed
	// interfaces attached to the Instance.
	SecondaryVpcIDs []string `json:"secondaryVpcIds"`
	// MachineID is the ID of the Machine
	MachineID *string `json:"machineId"`
	// Machine is the summary of the Machine
	Machine *APIMachineSummary `json:"machine,omitempty"`
	// OperatingSystemID is the ID of the OperatingSystem
	OperatingSystemID *string `json:"operatingSystemId"`
	// OperatingSystem is the summary of the OperatingSystem
	OperatingSystem *APIOperatingSystemSummary `json:"operatingSystem,omitempty"`
	// IpxeScript is an attribute which is inherited from Operating System
	IpxeScript *string `json:"ipxeScript"`
	// AlwaysBootWithCustomIpxe is an attribute which is specified by user if instance boot with ipxe or not
	AlwaysBootWithCustomIpxe bool `json:"alwaysBootWithCustomIpxe"`
	// PhoneHomeEnabled is an attribute which is specified by user if instance needs to be enabled for phone home or not
	PhoneHomeEnabled bool `json:"phoneHomeEnabled"`
	// UserData is inherited from Operating System or specified by user if allowed
	UserData *string `json:"userData"`
	// Labels is Instance labels specified by user
	Labels map[string]string `json:"labels"`
	// IsUpdatePending is an attribute suggest if instance update pending or not
	IsUpdatePending bool `json:"isUpdatePending"`
	// SerialConsoleURL is the ssh serial console URL associated with the instance
	SerialConsoleURL *string `json:"serialConsoleUrl"`
	// NetworkSecurityGroupID is the ID of attached NSG, if any
	NetworkSecurityGroupID *string `json:"networkSecurityGroupId"`
	// NetworkSecurityGroup holds the summary for attached NSG, if requested via includeRelation
	NetworkSecurityGroup *networking.APINetworkSecurityGroupSummary `json:"networkSecurityGroup,omitempty"`
	// NetworkSecurityGroupPropagationDetails is the propagation details for the attached NSG, if any
	NetworkSecurityGroupPropagationDetails *networking.APINetworkSecurityGroupPropagationDetails `json:"networkSecurityGroupPropagationDetails"`
	// NetworkSecurityGroupInherited indicates if the Instance is inheriting Network Security Group rules from parent VPC
	NetworkSecurityGroupInherited bool `json:"networkSecurityGroupInherited"`
	// TPM EK Cert
	TpmEkCertificate *string `json:"tpmEkCertificate"`
	// Status is the status of the Instance
	Status string `json:"status"`
	// Interfaces are list of the subnet associated with the Instance
	Interfaces []networking.APIInterface `json:"interfaces"`
	// InfiniBandInterfaces are list of the InfiniBandInterface associated with the Instance
	InfiniBandInterfaces []networking.APIInfiniBandInterface `json:"infinibandInterfaces"`
	// DpuExtensionServiceDeployments are list of the DpuExtensionServiceDeployments associated with the Instance
	DpuExtensionServiceDeployments []networking.APIDpuExtensionServiceDeployment `json:"dpuExtensionServiceDeployments"`
	// NVLinkInterfaces are list of the NVLinkInterface associated with the Instance
	NVLinkInterfaces []networking.APINVLinkInterface `json:"nvLinkInterfaces"`
	// SSHKeyGroupIDs are list of the ssh key group IDs associated with the Instance
	SSHKeyGroupIDs []string `json:"sshKeyGroupIds"`
	// SSHKeyGroups are list of the ssh key group associated with the Instance
	SSHKeyGroups []APISSHKeyGroupSummary `json:"sshKeyGroups"`
	// StatusHistory is the history of statuses for the Instance
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// Created indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the entity was last updated
	Updated time.Time `json:"updated"`
	// Deprecations is the list of deprecation messages denoting fields which are being deprecated
	Deprecations []APIDeprecation `json:"deprecations,omitempty"`
}

// APIInstanceStats is a data structure to capture information about Instance stats at the API layer
type APIInstanceStats struct {
	// Total is the total number of the Instances
	Total int `json:"total"`
	// Pending is the total number of pending Instances
	Pending int `json:"pending"`
	// Terminating is the total number of provisioning Instances
	Terminating int `json:"terminating"`
	// Ready is the total number of ready Instances
	Ready int `json:"ready"`
	// Updating is the total number of Instances receiving system updates
	Updating int `json:"updating"`
	// Registering is the total number of registering Instances
	Registering int `json:"registering"`
	// Error is the total number of error Instances
	Error int `json:"error"`
}
