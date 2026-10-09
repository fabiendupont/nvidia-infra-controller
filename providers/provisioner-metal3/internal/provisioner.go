// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metal3

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	provisionTimeout   = 30 * time.Minute
	pollInterval       = 10 * time.Second
	bmcSecretSuffix    = "-bmc-creds"
	// AnnotationProvisioner is the machine annotation that selects this backend.
	AnnotationProvisioner = "nico.nvidia.com/provisioner"
	ProvisionerName       = "metal3"
)

// Metal3Provisioner delegates machine provisioning to Metal3 Bare Metal Operator
// by managing BareMetalHost CRs in the management cluster.
type Metal3Provisioner struct {
	dynamic   dynamic.Interface
	k8s       kubernetes.Interface
	namespace string // Kubernetes namespace for BareMetalHost CRs
}

// NewMetal3Provisioner builds a Metal3Provisioner from the given k8s config.
func NewMetal3Provisioner(kubeconfigPath, namespace string) (*Metal3Provisioner, error) {
	var cfg *rest.Config
	var err error
	if kubeconfigPath != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("build k8s config: %w", err)
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}

	return &Metal3Provisioner{dynamic: dyn, k8s: k8s, namespace: namespace}, nil
}

// Name satisfies MachineProvisioner.
func (p *Metal3Provisioner) Name() string { return ProvisionerName }

// Provision creates (or updates) a BareMetalHost CR for the given machine and
// waits until BMO reports it as provisioned.
func (p *Metal3Provisioner) Provision(ctx context.Context, req ProvisionRequest) error {
	bmhName := "nico-" + req.MachineID
	secretName := bmhName + bmcSecretSuffix

	logger := log.With().
		Str("machine_id", req.MachineID).
		Str("bmh", bmhName).
		Logger()

	// 1. Ensure the BMC credential Secret exists.
	if err := p.ensureBMCSecret(ctx, secretName, req.BMCUsername, req.BMCPassword); err != nil {
		return fmt.Errorf("ensure BMC secret: %w", err)
	}

	// 2. Create or update the BareMetalHost CR.
	spec := BareMetalHostSpec{
		BMC: BMCDetails{
			Address:         req.BMCURL,
			CredentialsName: secretName,
		},
		BootMACAddress: req.BootMACAddress,
		Online:         true,
		Image: &ImageSpec{
			URL:          req.ImageURL,
			Checksum:     req.ImageChecksum,
			ChecksumType: req.ImageChecksumType,
		},
	}
	if err := p.applyBMH(ctx, bmhName, spec); err != nil {
		return fmt.Errorf("apply BareMetalHost: %w", err)
	}

	// 3. Poll until provisioned or error.
	logger.Info().Msg("waiting for BareMetalHost to reach provisioned state")
	deadline := time.Now().Add(provisionTimeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for BareMetalHost %s to provision", bmhName)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}

		state, errMsg, err := p.getBMHState(ctx, bmhName)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to get BareMetalHost state, retrying")
			continue
		}
		logger.Debug().Str("state", string(state)).Msg("BareMetalHost state")
		switch state {
		case BMHStateProvisioned:
			logger.Info().Msg("BareMetalHost provisioned")
			return nil
		case BMHStateError:
			return fmt.Errorf("BareMetalHost %s entered error state: %s", bmhName, errMsg)
		}
	}
}

// Deprovision sets the BareMetalHost offline+imageless and waits for
// BMO to wipe the machine, then deletes the CR and BMC Secret.
func (p *Metal3Provisioner) Deprovision(ctx context.Context, machineID string) error {
	bmhName := "nico-" + machineID
	secretName := bmhName + bmcSecretSuffix

	logger := log.With().Str("machine_id", machineID).Str("bmh", bmhName).Logger()

	// Patch to trigger deprovisioning: online=false, image=nil.
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"online": false,
			"image":  nil,
		},
	}
	patchBytes, _ := json.Marshal(patch)
	_, err := p.dynamic.Resource(BareMetalHostGVR).Namespace(p.namespace).
		Patch(ctx, bmhName, "merge-patch+json", patchBytes, metav1.PatchOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("patch BareMetalHost for deprovision: %w", err)
	}

	// Wait for available state.
	logger.Info().Msg("waiting for BareMetalHost to reach available state")
	deadline := time.Now().Add(provisionTimeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for BareMetalHost %s to deprovision", bmhName)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}

		state, _, err := p.getBMHState(ctx, bmhName)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // already gone
			}
			continue
		}
		if state == BMHStateAvailable {
			break
		}
	}

	// Delete BareMetalHost and Secret.
	_ = p.dynamic.Resource(BareMetalHostGVR).Namespace(p.namespace).
		Delete(ctx, bmhName, metav1.DeleteOptions{})
	_ = p.k8s.CoreV1().Secrets(p.namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	logger.Info().Msg("BareMetalHost deprovisioned and deleted")
	return nil
}

// PowerControl changes machine power state via the BareMetalHost online field.
func (p *Metal3Provisioner) PowerControl(ctx context.Context, machineID string, action PowerAction) error {
	bmhName := "nico-" + machineID
	switch action {
	case PowerActionOn:
		return p.patchOnline(ctx, bmhName, true)
	case PowerActionOff:
		return p.patchOnline(ctx, bmhName, false)
	case PowerActionCycle:
		if err := p.patchOnline(ctx, bmhName, false); err != nil {
			return err
		}
		// Brief pause to let BMO act before powering back on.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
		return p.patchOnline(ctx, bmhName, true)
	default:
		return fmt.Errorf("unknown power action: %s", action)
	}
}

// --- helpers ---

func (p *Metal3Provisioner) ensureBMCSecret(ctx context.Context, name, username, password string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.namespace},
		StringData: map[string]string{"username": username, "password": password},
	}
	_, err := p.k8s.CoreV1().Secrets(p.namespace).Create(ctx, secret, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = p.k8s.CoreV1().Secrets(p.namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	return err
}

func (p *Metal3Provisioner) applyBMH(ctx context.Context, name string, spec BareMetalHostSpec) error {
	specMap, err := toUnstructured(spec)
	if err != nil {
		return err
	}
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "metal3.io/v1alpha1",
			"kind":       "BareMetalHost",
			"metadata":   map[string]interface{}{"name": name, "namespace": p.namespace},
			"spec":       specMap,
		},
	}
	_, createErr := p.dynamic.Resource(BareMetalHostGVR).Namespace(p.namespace).
		Create(ctx, obj, metav1.CreateOptions{})
	if errors.IsAlreadyExists(createErr) {
		data, _ := json.Marshal(map[string]interface{}{"spec": specMap})
		_, err = p.dynamic.Resource(BareMetalHostGVR).Namespace(p.namespace).
			Patch(ctx, name, "merge-patch+json", data, metav1.PatchOptions{})
		return err
	}
	return createErr
}

func (p *Metal3Provisioner) getBMHState(ctx context.Context, name string) (BMHState, string, error) {
	obj, err := p.dynamic.Resource(BareMetalHostGVR).Namespace(p.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	statusRaw, _, _ := unstructured.NestedMap(obj.Object, "status")
	data, _ := json.Marshal(statusRaw)
	var status BareMetalHostStatus
	_ = json.Unmarshal(data, &status)
	return status.Provisioning.State, status.ErrorMessage, nil
}

func (p *Metal3Provisioner) patchOnline(ctx context.Context, name string, online bool) error {
	data, _ := json.Marshal(map[string]interface{}{"spec": map[string]interface{}{"online": online}})
	_, err := p.dynamic.Resource(BareMetalHostGVR).Namespace(p.namespace).
		Patch(ctx, name, "merge-patch+json", data, metav1.PatchOptions{})
	return err
}

func toUnstructured(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	return m, json.Unmarshal(b, &m)
}
