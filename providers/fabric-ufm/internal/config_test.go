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

package ufmfabric

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnvBool_True(t *testing.T) {
	t.Setenv("TEST_ENVBOOL", "true")
	assert.True(t, envBool("TEST_ENVBOOL"))
}

func TestEnvBool_False(t *testing.T) {
	t.Setenv("TEST_ENVBOOL", "false")
	assert.False(t, envBool("TEST_ENVBOOL"))
}

func TestEnvBool_One(t *testing.T) {
	t.Setenv("TEST_ENVBOOL", "1")
	assert.True(t, envBool("TEST_ENVBOOL"))
}

func TestEnvBool_Unset(t *testing.T) {
	assert.False(t, envBool("TEST_ENVBOOL_UNSET_KEY"))
}

func TestEnvBool_Invalid(t *testing.T) {
	t.Setenv("TEST_ENVBOOL", "notabool")
	assert.False(t, envBool("TEST_ENVBOOL"))
}

func TestEnvBoolDefault_Unset_ReturnsDefault(t *testing.T) {
	assert.True(t, envBoolDefault("TEST_ENVBOOLDEFAULT_UNSET", true))
	assert.False(t, envBoolDefault("TEST_ENVBOOLDEFAULT_UNSET", false))
}

func TestEnvBoolDefault_Set_OverridesDefault(t *testing.T) {
	t.Setenv("TEST_ENVBOOLDEFAULT", "false")
	assert.False(t, envBoolDefault("TEST_ENVBOOLDEFAULT", true))
}

func TestEnvBoolDefault_Invalid_ReturnsDefault(t *testing.T) {
	t.Setenv("TEST_ENVBOOLDEFAULT", "badvalue")
	assert.True(t, envBoolDefault("TEST_ENVBOOLDEFAULT", true))
}

func TestConfigFromEnv_AllSet(t *testing.T) {
	t.Setenv("UFM_URL", "https://ufm.lab:443")
	t.Setenv("UFM_USERNAME", "admin")
	t.Setenv("UFM_PASSWORD", "supersecret")
	t.Setenv("UFM_TLS_SKIP_VERIFY", "true")
	t.Setenv("UFM_SYNC_IB_PARTITION", "false")
	t.Setenv("UFM_JOB_POLL_INTERVAL", "5s")
	t.Setenv("UFM_JOB_TIMEOUT", "3m")

	cfg := ConfigFromEnv()

	assert.Equal(t, "https://ufm.lab:443", cfg.UFMURL)
	assert.Equal(t, "admin", cfg.UFMUsername)
	assert.Equal(t, "supersecret", cfg.UFMPassword)
	assert.True(t, cfg.TLSSkipVerify)
	assert.False(t, cfg.Features.SyncIBPartition)
	assert.Equal(t, 5*time.Second, cfg.JobPollInterval)
	assert.Equal(t, 3*time.Minute, cfg.JobTimeout)
}

func TestConfigFromEnv_InvalidDurations(t *testing.T) {
	t.Setenv("UFM_URL", "https://ufm.lab")
	t.Setenv("UFM_USERNAME", "admin")
	t.Setenv("UFM_PASSWORD", "secret")
	t.Setenv("UFM_JOB_POLL_INTERVAL", "not-a-duration")
	t.Setenv("UFM_JOB_TIMEOUT", "also-bad")

	cfg := ConfigFromEnv()

	assert.Equal(t, time.Duration(0), cfg.JobPollInterval, "invalid duration should be ignored")
	assert.Equal(t, time.Duration(0), cfg.JobTimeout, "invalid duration should be ignored")
}
