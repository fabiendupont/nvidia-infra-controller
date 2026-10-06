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
	"errors"
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	validationis "github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	// MachineMaxLabelCount is the maximum number of Labels allowed per Machine
	MachineMaxLabelCount = 10

	validationErrorMachineMaintenanceStringLength = "must be at least 5 characters and maximum 256 characters"
)

// APIMachineUpdateRequest is the data structure to capture request to update a Machine
type APIMachineUpdateRequest struct {
	// InstanceTypeID is the ID of the InstanceType to set for the Machine
	InstanceTypeID *string `json:"instanceTypeId"`
	// ClearInstanceType indicates that the InstanceType should be cleared
	ClearInstanceType *bool `json:"clearInstanceType"`
	// SetMaintenanceMode enables or disables maintenance mode
	SetMaintenanceMode *bool `json:"setMaintenanceMode"`
	// MaintenanceMessage is the message to display during maintenance mode
	MaintenanceMessage *string `json:"maintenanceMessage"`
	// Labels allows setting a key value pair of arbitrary string metadata for the Machine
	Labels map[string]string `json:"labels"`
}

// Validate ensure the values passed in request are acceptable
func (mur APIMachineUpdateRequest) Validate() error {
	err := validation.ValidateStruct(&mur,
		validation.Field(&mur.InstanceTypeID,
			validationis.UUID.Error(rest.ValidationErrorInvalidUUID)),
		validation.Field(&mur.MaintenanceMessage,
			validation.When(mur.SetMaintenanceMode == nil || (mur.SetMaintenanceMode != nil && !*mur.SetMaintenanceMode), validation.Nil.Error("MaintenanceMessage cannot be specified unless SetMaintenanceMode is true")),
			validation.When(mur.SetMaintenanceMode != nil && *mur.SetMaintenanceMode && mur.MaintenanceMessage == nil, validation.Required.Error("MaintenanceMessage is required when SetMaintenanceMode is true")),
			validation.When(mur.SetMaintenanceMode != nil && *mur.SetMaintenanceMode && mur.MaintenanceMessage != nil, validation.Required.Error("MaintenanceMessage cannot be empty")),
			validation.When(mur.SetMaintenanceMode != nil && *mur.SetMaintenanceMode && mur.MaintenanceMessage != nil, validation.Match(rest.NotAllWhitespaceRegexp).Error("field consists only of whitespace")),
			validation.When(mur.SetMaintenanceMode != nil && *mur.SetMaintenanceMode && mur.MaintenanceMessage != nil, validation.Length(5, 256).Error(validationErrorMachineMaintenanceStringLength)),
		),
	)

	exclusiveOptionsCount := 0
	if mur.InstanceTypeID != nil {
		exclusiveOptionsCount++
	}

	if mur.ClearInstanceType != nil {
		exclusiveOptionsCount++
	}

	if mur.SetMaintenanceMode != nil {
		exclusiveOptionsCount++
	}

	if mur.Labels != nil {
		exclusiveOptionsCount++
	}

	if err == nil && exclusiveOptionsCount > 1 {
		err = validation.Errors{
			rest.ValidationCommonErrorField: errors.New("only one of setMaintenanceMode, instanceTypeId, clearInstanceType or labels can be set at a time"),
		}
	}

	if err == nil && exclusiveOptionsCount == 0 {
		err = validation.Errors{
			rest.ValidationCommonErrorField: errors.New("no updates specified. At least one of setMaintenanceMode, instanceTypeId, clearInstanceType or labels must be specified"),
		}
	}

	if err == nil && mur.ClearInstanceType != nil && !*mur.ClearInstanceType {
		err = validation.Errors{
			"clearInstanceType": errors.New("must be set to true to clear the Instance Type"),
		}
	}

	if err := rest.ValidateLabels(mur.Labels); err != nil {
		return err
	}

	return err
}

// APIMachine is the data structure to capture API representation of a Machine
type APIMachine struct {
	// ID is the unique UUID v4 identifier for the Machine
	ID string `json:"id"`
	// InfrastructureProviderID is the ID of the InfrastructureProvider
	InfrastructureProviderID string `json:"infrastructureProviderId"`
	// InfrastructureProvider is the summary of the InfrastructureProvider
	InfrastructureProvider *rest.APIInfrastructureProviderSummary `json:"infrastructureProvider,omitempty"`
	// SiteID is the ID of the Site
	SiteID string `json:"siteId"`
	// Site is the summary of the Site
	Site *rest.APISiteSummary `json:"site,omitempty"`
	// InstanceTypeID is the ID of the associated Instance Type
	InstanceTypeID *string `json:"instanceTypeId"`
	// InstanceType is the summary of the associated Instance Type
	InstanceType *APIInstanceTypeSummary `json:"instanceType,omitempty"`
	// InstanceID is the ID of the associated Instance (if any)
	InstanceID *string `json:"instanceId"`
	// Instance is the summary of the associated Instance (if any)
	Instance *rest.APIInstanceSummary `json:"instance,omitempty"`
	// TenantID is the ID of the Tenant that owns the Instance associated (if any)
	TenantID *string `json:"tenantId"`
	// Tenant is the summary of the Tenant that owns the Instance associated (if any)
	Tenant *rest.APITenantSummary `json:"tenant,omitempty"`
	// ControllerMachineID is the ID of the controllerMachine
	ControllerMachineID string `json:"controllerMachineId"`
	// ControllerMachineType is the type of the controller machine
	ControllerMachineType *string `json:"controllerMachineType"`
	// HwSkuDeviceType is the sku derived device type of the machine, e.g. cpu, gpu, cache, storage, etc.
	HwSkuDeviceType *string `json:"hwSkuDeviceType"`
	// Vendor is the vendor of the Machine
	Vendor *string `json:"vendor"`
	// ProductName is the product name of the Machine
	ProductName *string `json:"productName"`
	// SerialNumber is the serial number of the Machine
	SerialNumber *string `json:"serialNumber"`
	// Hostname is the hostname of the Machine
	Hostname *string `json:"hostname"`
	// MachineCapabilities is the list of capabilities of the machine
	MachineCapabilities []APIMachineCapability `json:"machineCapabilities"`
	// MachineInterfaces is the list of admin interfaces of the machine
	MachineInterfaces []APIMachineInterface `json:"machineInterfaces"`
	// MaintenanceMessage is the message to display during maintenance mode
	MaintenanceMessage *string `json:"maintenanceMessage"`
	// Metadata contains additional metadata about the machine
	Metadata *APIMachineMetadata `json:"metadata,omitempty"`
	// Health contains health information about the machine
	Health *APIMachineHealth `json:"health"`
	// Labels is VPC labels specified by user
	Labels map[string]string `json:"labels"`
	// Status represents the status of the machine
	Status string `json:"status"`
	// IsUsableByTenant indicates whether the machine is usable by or currently in use by a tenant.
	IsUsableByTenant bool `json:"isUsableByTenant"`
	// StatusHistory is the history of statuses for the Machine
	StatusHistory []rest.APIStatusDetail `json:"statusHistory"`
	// CreatedAt indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// UpdatedAt indicates the ISO datetime string for when the entity was last updated
	Updated time.Time `json:"updated"`
	// Deprecations is the list of deprecation messages denoting fields which are being deprecated
	Deprecations []APIDeprecation `json:"deprecations,omitempty"`
}

// APIDMIData is the data structure to capture API representation of a Machine's DMIData
type APIDMIData struct {
	// BoardName is the name of the Machine's board
	BoardName *string `json:"boardName"`
	// BoardVersion is the version of the Machine's board
	BoardVersion *string `json:"boardVersion"`
	// BiosDate is the date of the Machine's bios
	BiosDate *string `json:"biosDate"`
	// BiosVersion is the version of the Machine's bios
	BiosVersion *string `json:"biosVersion"`
	// ProductName is the name of the Machine's product
	ProductName *string `json:"productName"`
	// ProductSerial is searial number the Machine
	ProductSerial *string `json:"productSerial"`
	// BoardSerial is the searial number of the Machine's board
	BoardSerial *string `json:"boardSerial"`
	// ChassisSerial is searial number the Machine's Chassis
	ChassisSerial *string `json:"chassisSerial"`
	// SysVendor is the vendor of the Machine's system
	SysVendor *string `json:"sysVendor"`
}

// APIBMCInfo is the data structure to capture API representation of a Machine's BMC Info
type APIBMCInfo struct {
	// IP is the IP Address of the Machine's BMI
	IP *string `json:"ip"`
	// Mac is the Mac Address of the Machine's BMI
	Mac *string `json:"mac"`
	// Version is the version of the Machine's BMI
	Version *string `json:"version"`
	// FirmwareRevision is the firmware version revision of the Machine's BMI
	FirmwareRevision *string `json:"firmwareRevision"`
}

// APIMachineGPUInfo is the data structure to capture API representation of a Machine's GPU Info
type APIMachineGPUInfo struct {
	// Name of the Machine's GPU
	Name *string `json:"name"`
	// Serial is the serial number of the Machine's GPU
	Serial *string `json:"serial"`
	// DriverVersion is the version of the Machine's GPU driver
	DriverVersion *string `json:"driverVersion"`
	// VbiosVersion is the bios version of the Machine's GPU
	VbiosVersion *string `json:"vbiosVersion"`
	// InforomVersion is the info rom version of the Machine's GPU
	InforomVersion *string `json:"inforomVersion"`
	// TotalMemory is the total memory of the Machine's GPU
	TotalMemory *string `json:"totalMemory"`
	// Frequency is the frequency of the Machine's GPU
	Frequency *string `json:"frequency"`
	// PciBusId is the PCI BusId of the Machine's GPU
	PciBusId *string `json:"pciBusId"`
}

// APIMachineNetworkInterface is the data structure to capture API representation of a Machine's Network Interface Info
type APIMachineNetworkInterface struct {
	// MacAddress of the Machine's NetworkInterface
	MacAddress *string `json:"macAddress"`
	// Vendor is the serial number of the Machine's NetworkInterface
	Vendor *string `json:"vendor"`
	// Device is the device number of the Machine's NetworkInterface
	Device *string `json:"device"`
	// Path is the bios path of the Machine's NetworkInterface
	Path *string `json:"path"`
	// NumaNode is the info of numa node Machine's NetworkInterface
	NumaNode *int32 `json:"numaNode"`
	// Description is the description the Machine's NetworkInterface
	Description *string `json:"description"`
	// Slot is the slot number of the Machine's NetworkInterface
	Slot *string `json:"slot"`
}

// APIMachineInfiniBandInterface is the data structure to capture API representation of a Machine's InfiniBand Interface Info
type APIMachineInfiniBandInterface struct {
	// Guid of the Machine's InfiniBandInterface
	Guid *string `json:"guid"`
	// Vendor is the serial number of the Machine's InfiniBandInterface
	Vendor *string `json:"vendor"`
	// Device is the device number of the Machine's InfiniBandInterface
	Device *string `json:"device"`
	// Path is the bios path of the Machine's InfiniBandInterface
	Path *string `json:"path"`
	// NumaNode is the info of numa node Machine's InfiniBandInterface
	NumaNode *int32 `json:"numaNode"`
	// Description is the description the Machine's InfiniBandInterface
	Description *string `json:"description"`
	// Slot is the slot number of the Machine's InfiniBandInterface
	Slot *string `json:"slot"`
}

// APIMachineMetadata is the data structure to capture API representation of a Machine's Metadata Info
type APIMachineMetadata struct {
	// DMIData is the DMI data of the machine
	DMIData *APIDMIData `json:"dmiData,omitempty"`
	// BMCInfo is the BMC Info of the machine
	BMCInfo *APIBMCInfo `json:"bmcInfo,omitempty"`
	// GPUs is the list of GPUs for the machine
	GPUs []APIMachineGPUInfo `json:"gpus,omitempty"`
	// NetworkInterfaces is the list of Ethernet interfaces of the machine
	NetworkInterfaces []APIMachineNetworkInterface `json:"networkInterfaces,omitempty"`
	// InfiniBandInterfaces is the list of InfiniBand interfaces of the machine
	InfiniBandInterfaces []APIMachineInfiniBandInterface `json:"infinibandInterfaces,omitempty"`
}

// APIMachineHealth is the data structure to capture API representation of a Machine's health Info
type APIMachineHealth struct {
	Source               string                         `json:"source"`
	ObservedAt           *string                        `json:"observedAt"`
	ObservedAtDeprecated *string                        `json:"observed_at"`
	Successes            []APIMachineHealthProbeSuccess `json:"successes"`
	Alerts               []APIMachineHealthProbeAlert   `json:"alerts"`
}

// APIMachineHealthProbeSuccess is the data structure for a machine health probe success
type APIMachineHealthProbeSuccess struct {
	ID     string  `json:"id"`
	Target *string `json:"target"`
}

// APIMachineHealthProbeAlert is the data structure for a machine health probe alert
type APIMachineHealthProbeAlert struct {
	ID                      string   `json:"id"`
	Target                  *string  `json:"target"`
	InAlertSince            *string  `json:"inAlertSince"`
	InAlertSinceDeprecated  *string  `json:"in_alert_since"`
	Message                 string   `json:"message"`
	TenantMessage           *string  `json:"tenantMessage"`
	TenantMessageDeprecated *string  `json:"tenant_message"`
	Classifications         []string `json:"classifications"`
}

// APIMachineSummary is the data structure to provide a summary of a Machine
type APIMachineSummary struct {
	// ID of the Machine
	ID string `json:"id"`
	// ControllerMachineID is the ID of the controllerMachine
	ControllerMachineID string `json:"controllerMachineId"`
	// ControllerMachineType is the type of the controller machine
	ControllerMachineType *string `json:"controllerMachineType"`
	// HwSkuDeviceType is the sku derived device type of the machine, e.g. cpu, gpu, cache, storage, etc.
	HwSkuDeviceType *string `json:"hwSkuDeviceType"`
	// Vendor is the vendor of the Machine
	Vendor *string `json:"vendor"`
	// ProductName is the product name of the Machine
	ProductName *string `json:"productName"`
	// MaintenanceMessage is the message to display during maintenance mode
	MaintenanceMessage *string `json:"maintenanceMessage"`
	// Status represents the status of the machine
	Status string `json:"status"`
}

// APIMachineStats is a data structure to capture information about machine stats at the API layer
type APIMachineStats struct {
	// Total is the total number of the machine object
	Total int `json:"total"`
	// Initializing is the total number of initializing machine object
	Initializing int `json:"initializing"`
	// Reset is the total number of reset machine object
	Reset int `json:"reset"`
	// Ready is the total number of ready machine object
	Ready int `json:"ready"`
	// InUse is the total number of Machines in use by Tenant Instances
	InUse int `json:"inUse"`
	// Error is the total number of error machine object
	Error int `json:"error"`
	// Decommissioned is the total number of decommissioned machine object
	Decommissioned int `json:"decommissioned"`
	// Maintenance is the total number of machines in Maintenance
	Maintenance int `json:"maintenance"`
	// Unknown is the total number of unknown machine object
	Unknown int `json:"unknown"`
}
