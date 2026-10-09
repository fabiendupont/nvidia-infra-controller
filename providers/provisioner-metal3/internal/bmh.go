// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package metal3 implements a NICo provider that provisions bare-metal
// machines via Metal3 Bare Metal Operator (BMO) and its BareMetalHost CRD.
// BMO reconciles BareMetalHost objects against physical hardware using Ironic
// (IPMI / Redfish / PXE); NICo creates and watches the CRs and delegates all
// low-level provisioning to BMO.
package metal3

import "k8s.io/apimachinery/pkg/runtime/schema"

// BareMetalHostGVR is the GroupVersionResource for the metal3.io BareMetalHost CRD.
var BareMetalHostGVR = schema.GroupVersionResource{
	Group:    "metal3.io",
	Version:  "v1alpha1",
	Resource: "baremetalhosts",
}

// BMHState mirrors the BareMetalHost provisioning state machine.
type BMHState string

const (
	BMHStateAvailable     BMHState = "available"
	BMHStateProvisioning  BMHState = "provisioning"
	BMHStateProvisioned   BMHState = "provisioned"
	BMHStateDeprovisioning BMHState = "deprovisioning"
	BMHStateError         BMHState = "error"
	BMHStateDeleting      BMHState = "deleting"
)

// BareMetalHostSpec is the portion of the BareMetalHost spec that NICo writes.
type BareMetalHostSpec struct {
	BMC            BMCDetails `json:"bmc"`
	BootMACAddress string     `json:"bootMACAddress,omitempty"`
	// Online drives power state. true = powered on.
	Online bool       `json:"online"`
	Image  *ImageSpec `json:"image,omitempty"`
}

// BMCDetails identifies the out-of-band management interface.
type BMCDetails struct {
	// Address is the BMC endpoint, e.g. "ipmi://192.168.1.2" or "redfish+https://192.168.1.2/redfish/v1/Systems/1"
	Address string `json:"address"`
	// CredentialsName is the name of a Secret in the same namespace with keys
	// "username" and "password".
	CredentialsName                string `json:"credentialsName"`
	DisableCertificateVerification bool   `json:"disableCertificateVerification,omitempty"`
}

// ImageSpec describes the OS image to deploy.
type ImageSpec struct {
	URL          string `json:"url"`
	Checksum     string `json:"checksum,omitempty"`
	ChecksumType string `json:"checksumType,omitempty"` // "md5", "sha256", "sha512"
}

// BareMetalHostStatus is the subset of status that NICo reads.
type BareMetalHostStatus struct {
	Provisioning ProvisionStatus `json:"provisioning"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	ErrorType    string          `json:"errorType,omitempty"`
}

// ProvisionStatus holds the current provisioning state.
type ProvisionStatus struct {
	State BMHState `json:"state"`
}

// ProvisionRequest carries everything NICo knows about a machine being provisioned.
type ProvisionRequest struct {
	MachineID       string
	BMCURL          string
	BMCUsername     string
	BMCPassword     string
	BootMACAddress  string
	ImageURL        string
	ImageChecksum   string
	ImageChecksumType string
	Hostname        string
}

// PowerAction controls machine power.
type PowerAction string

const (
	PowerActionOn    PowerAction = "on"
	PowerActionOff   PowerAction = "off"
	PowerActionCycle PowerAction = "cycle"
)
