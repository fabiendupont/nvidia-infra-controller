// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package spectrumfabric implements a NICo provider that manages Spectrum-X
// switch fabric directly via the NVUE REST API, without an intermediary
// orchestration layer like Ansible/AAP or Netris Controller.
//
// This provider communicates with Cumulus Linux switches running NVUE by
// sending JSON PATCH requests to the NVUE declarative API. Each NICo
// networking event (VPC create/delete, Subnet create/delete) is translated
// into NVUE configuration changes:
//
//   - VPC lifecycle maps to VRF objects on Spectrum switches
//   - Subnet lifecycle maps to VxLAN VNI + bridge VLAN configuration
//
// The provider is hook-driven: it registers reactions on VPC and Subnet
// lifecycle events and a pre-create subnet sync hook for validation.
package spectrumfabric
