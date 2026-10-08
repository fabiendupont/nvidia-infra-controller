// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package spectrumfabric

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// FeatureConfig controls which NICo lifecycle events trigger NVUE configuration
// changes on the Spectrum switches.
type FeatureConfig struct {
	SyncVPC    bool `json:"sync_vpc"`
	SyncSubnet bool `json:"sync_subnet"`
}

// ProviderConfig holds the full configuration for the spectrum-fabric provider.
type ProviderConfig struct {
	NVUEURL              string
	NVUEUsername         string
	NVUEPassword         string
	TLSSkipVerify        bool
	Features             FeatureConfig
	RevisionPollInterval time.Duration
	RevisionTimeout      time.Duration
}

func ConfigFromEnv() ProviderConfig {
	cfg := ProviderConfig{
		NVUEURL:       os.Getenv("NVUE_URL"),
		NVUEUsername:  os.Getenv("NVUE_USERNAME"),
		NVUEPassword:  os.Getenv("NVUE_PASSWORD"),
		TLSSkipVerify: envBool("NVUE_TLS_SKIP_VERIFY"),
		Features: FeatureConfig{
			SyncVPC:    envBoolDefault("NVUE_SYNC_VPC", true),
			SyncSubnet: envBoolDefault("NVUE_SYNC_SUBNET", true),
		},
	}
	if d := os.Getenv("NVUE_REVISION_POLL_INTERVAL"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			cfg.RevisionPollInterval = parsed
		}
	}
	if d := os.Getenv("NVUE_REVISION_TIMEOUT"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			cfg.RevisionTimeout = parsed
		}
	}
	return cfg
}

func (c *ProviderConfig) Validate() error {
	if c.NVUEURL == "" {
		return fmt.Errorf("NVUE_URL is required")
	}
	if c.NVUEUsername == "" {
		return fmt.Errorf("NVUE_USERNAME is required")
	}
	if c.NVUEPassword == "" {
		return fmt.Errorf("NVUE_PASSWORD is required")
	}
	return nil
}

func envBool(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	b, _ := strconv.ParseBool(v)
	return b
}

func envBoolDefault(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
