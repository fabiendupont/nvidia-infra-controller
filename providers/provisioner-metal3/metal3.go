// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package metal3 exposes the Metal3 Bare Metal Operator provisioner as an
// adapter that implements the rest-api/api/pkg/provisioner.MachineProvisioner
// interface. The implementation lives in internal/ and is wrapped here to
// bridge the two packages' PowerAction and ProvisionRequest types.
package metal3

import (
	"context"

	mp "github.com/NVIDIA/infra-controller/rest-api/api/pkg/provisioner"
	"github.com/NVIDIA/infra-controller/providers/provisioner-metal3/internal"
)

// Adapter wraps the internal Metal3Provisioner and implements
// rest-api/api/pkg/provisioner.MachineProvisioner.
type Adapter struct {
	inner *metal3.Metal3Provisioner
}

func (a *Adapter) Name() string { return a.inner.Name() }

func (a *Adapter) Provision(ctx context.Context, req mp.ProvisionRequest) error {
	return a.inner.Provision(ctx, metal3.ProvisionRequest{
		MachineID:         req.MachineID,
		BMCURL:            req.BMCURL,
		BMCUsername:       req.BMCUsername,
		BMCPassword:       req.BMCPassword,
		ImageURL:          req.ImageURL,
		ImageChecksum:     req.ImageChecksum,
		ImageChecksumType: req.ImageChecksumType,
		Hostname:          req.Hostname,
	})
}

func (a *Adapter) Deprovision(ctx context.Context, machineID string) error {
	return a.inner.Deprovision(ctx, machineID)
}

func (a *Adapter) PowerControl(ctx context.Context, machineID string, action mp.PowerAction) error {
	return a.inner.PowerControl(ctx, machineID, metal3.PowerAction(action))
}

// New constructs a Metal3 provisioner adapter that implements MachineProvisioner.
//   - kubeconfigPath: path to a kubeconfig file; empty means in-cluster config.
//   - namespace: the Kubernetes namespace where BareMetalHost CRs are created.
func New(kubeconfigPath, namespace string) (*Adapter, error) {
	inner, err := metal3.NewMetal3Provisioner(kubeconfigPath, namespace)
	if err != nil {
		return nil, err
	}
	return &Adapter{inner: inner}, nil
}
