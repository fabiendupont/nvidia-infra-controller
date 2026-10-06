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

package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	providerv1 "github.com/NVIDIA/infra-controller/provider-api/provider/v1"
)

func TestServerGetInfo(t *testing.T) {
	s := NewServer()

	info, err := s.GetInfo(context.Background(), &providerv1.GetInfoRequest{})
	require.NoError(t, err)
	assert.Equal(t, "nico-health", info.Name)
	assert.Equal(t, "2.0.0", info.Version)
	assert.Equal(t, []string{"health", "fault-management"}, info.Features)
	assert.Equal(t, []string{"nico-compute"}, info.Dependencies)
}

func TestServerInit(t *testing.T) {
	s := NewServer()

	resp, err := s.Init(context.Background(), &providerv1.InitRequest{})
	require.NoError(t, err)
	assert.True(t, resp.Ready)

	// Verify stores were created.
	assert.NotNil(t, s.faultStore)
	assert.NotNil(t, s.serviceEventStore)
	assert.NotNil(t, s.faultServiceEventStore)
	assert.NotNil(t, s.classificationStore)

	// Verify handlers were created.
	assert.NotNil(t, s.faultHandler)
	assert.NotNil(t, s.serviceEventHandler)

	// Verify metrics were created.
	assert.NotNil(t, s.metrics)
	assert.NotNil(t, s.metricsRegistry)
}

func TestServerShutdown(t *testing.T) {
	s := NewServer()

	_, err := s.Shutdown(context.Background(), &providerv1.ShutdownRequest{})
	require.NoError(t, err)
}

func TestServerHealthCheck(t *testing.T) {
	s := NewServer()

	resp, err := s.HealthCheck(context.Background(), &providerv1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, providerv1.HealthCheckResponse_SERVING, resp.Status)
}

func TestServerBlockInstanceOnFaultyMachine_NoFaults(t *testing.T) {
	s := NewServer()
	_, err := s.Init(context.Background(), &providerv1.InitRequest{})
	require.NoError(t, err)

	payload := map[string]interface{}{
		"machine_id": "machine-1",
	}
	err = s.blockInstanceOnFaultyMachine(context.Background(), payload)
	require.NoError(t, err)
}

func TestServerBlockInstanceOnFaultyMachine_WithOpenCritical(t *testing.T) {
	s := NewServer()
	_, err := s.Init(context.Background(), &providerv1.InitRequest{})
	require.NoError(t, err)

	machineID := "machine-1"
	require.NoError(t, s.faultStore.Create(&FaultEvent{
		Source: "dcgm", Severity: SeverityCritical, Component: ComponentGPU,
		Message: "GPU fault", SiteID: "site-1", State: FaultStateOpen,
		MachineID: &machineID,
	}))

	payload := map[string]interface{}{
		"machine_id": machineID,
	}
	err = s.blockInstanceOnFaultyMachine(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 open critical fault")
	assert.Contains(t, err.Error(), "blocked")
}
