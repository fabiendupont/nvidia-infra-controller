// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package metal3

import "os"

// Config holds runtime configuration for the Metal3 provider, read from
// environment variables at Init time.
type Config struct {
	// KubeconfigPath is the path to a kubeconfig file for the management cluster.
	// When empty, in-cluster config is used (default for pod deployments).
	KubeconfigPath string

	// Metal3Namespace is the Kubernetes namespace in which BareMetalHost CRs
	// and BMC credential Secrets are created. Defaults to "metal3-system".
	Metal3Namespace string

	// IronicInspectorURL is the base URL for the Ironic Inspector API,
	// e.g. http://ironic-inspector.metal3-system:5050. When empty, Ironic
	// Inspector integration is disabled and post-machine-inspect does no
	// TPM EK correlation (logs a warning and passes through).
	IronicInspectorURL string

	// IronicToken is an optional Bearer token for authenticating to Ironic Inspector.
	IronicToken string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() Config {
	ns := os.Getenv("METAL3_NAMESPACE")
	if ns == "" {
		ns = "metal3-system"
	}
	return Config{
		KubeconfigPath:     os.Getenv("KUBECONFIG"),
		Metal3Namespace:    ns,
		IronicInspectorURL: os.Getenv("IRONIC_INSPECTOR_URL"),
		IronicToken:        os.Getenv("IRONIC_TOKEN"),
	}
}
