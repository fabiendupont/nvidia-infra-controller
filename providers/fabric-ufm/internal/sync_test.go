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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fabiendupont/nvidia-ufm-api/pkg/ufmclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestUFMServer creates an httptest.Server that mocks the UFM REST API.
// The handler map routes method+path patterns to handler functions.
type ufmRoute struct {
	method  string
	prefix  string
	handler http.HandlerFunc
}

func newTestUFMServer(routes []ufmRoute) *httptest.Server {
	mux := http.NewServeMux()

	// Build a dispatch handler that matches method+path.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, route := range routes {
			if r.Method == route.method && strings.HasPrefix(r.URL.Path, route.prefix) {
				route.handler(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})

	return httptest.NewServer(mux)
}

// newTestProvider creates a UFMFabricProvider connected to the given test
// server URL. SyncIBPartition is enabled by default.
func newTestProvider(serverURL string) *UFMFabricProvider {
	cfg := ProviderConfig{
		UFMURL:      serverURL,
		UFMUsername:  "admin",
		UFMPassword:  "secret",
		Features:    FeatureConfig{SyncIBPartition: true},
		JobPollInterval: 10 * time.Millisecond,
	}
	p := New(cfg)
	p.client = ufmclient.New(serverURL, "admin", "secret")
	return p
}

// --- SyncIBPartitionToFabric ---

func TestSyncIBPartitionToFabric_Success(t *testing.T) {
	var createCalled, addGUIDsCalled bool
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/add", handler: func(w http.ResponseWriter, r *http.Request) {
			createCalled = true
			w.WriteHeader(http.StatusOK)
		}},
		{method: "POST", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			addGUIDsCalled = true
			w.WriteHeader(http.StatusOK)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.SyncIBPartitionToFabric(context.Background(), "part-1", "test-partition", "0x1234", "tenant-1", []string{"guid1", "guid2"})
	require.NoError(t, err)
	assert.True(t, createCalled, "CreateEmptyPKey should have been called")
	assert.True(t, addGUIDsCalled, "AddGUIDsToPKey should have been called")
}

func TestSyncIBPartitionToFabric_NoGUIDs(t *testing.T) {
	var createCalled bool
	addGUIDsCalled := false
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/add", handler: func(w http.ResponseWriter, r *http.Request) {
			createCalled = true
			w.WriteHeader(http.StatusOK)
		}},
		{method: "POST", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			addGUIDsCalled = true
			w.WriteHeader(http.StatusOK)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.SyncIBPartitionToFabric(context.Background(), "part-1", "test-partition", "0x1234", "tenant-1", nil)
	require.NoError(t, err)
	assert.True(t, createCalled, "CreateEmptyPKey should have been called")
	assert.False(t, addGUIDsCalled, "AddGUIDsToPKey should not have been called with no GUIDs")
}

func TestSyncIBPartitionToFabric_PartitionAlreadyExists(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/add", handler: func(w http.ResponseWriter, r *http.Request) {
			// UFM returns 400 with "already exist" when PKEY exists.
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "partition already exist"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	// No GUIDs so we don't reach step 2.
	err := p.SyncIBPartitionToFabric(context.Background(), "part-1", "test-partition", "0x1234", "tenant-1", nil)
	require.NoError(t, err, "should not error when partition already exists")
}

func TestSyncIBPartitionToFabric_CreateAPIError(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/add", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "internal error"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.SyncIBPartitionToFabric(context.Background(), "part-1", "test-partition", "0x1234", "tenant-1", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating PKEY")
}

func TestSyncIBPartitionToFabric_AddGUIDsAPIError(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/add", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}},
		{method: "POST", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "internal error"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.SyncIBPartitionToFabric(context.Background(), "part-1", "test-partition", "0x1234", "tenant-1", []string{"guid1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adding")
	assert.Contains(t, err.Error(), "GUIDs")
}

// --- RemoveIBPartitionFromFabric ---

func TestRemoveIBPartitionFromFabric_Success(t *testing.T) {
	var deleteCalled bool
	srv := newTestUFMServer([]ufmRoute{
		{method: "DELETE", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.RemoveIBPartitionFromFabric(context.Background(), "part-1", "0x1234")
	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestRemoveIBPartitionFromFabric_NotFound(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "DELETE", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": "not found"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.RemoveIBPartitionFromFabric(context.Background(), "part-1", "0x1234")
	require.NoError(t, err, "404 should be treated as success")
}

func TestRemoveIBPartitionFromFabric_APIError(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "DELETE", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "server error"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.RemoveIBPartitionFromFabric(context.Background(), "part-1", "0x1234")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deleting PKEY")
}

// --- AddGUIDsToPartition ---

func TestAddGUIDsToPartition_Success(t *testing.T) {
	var reqBody ufmclient.PKeyAddGUIDsRequest
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&reqBody)
			w.WriteHeader(http.StatusOK)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.AddGUIDsToPartition(context.Background(), "0x1234", []string{"guid1", "guid2"}, "limited")
	require.NoError(t, err)
	assert.Equal(t, "limited", reqBody.Membership)
	assert.Equal(t, "0x1234", reqBody.PKey)
	assert.Equal(t, []string{"guid1", "guid2"}, reqBody.GUIDs)
}

func TestAddGUIDsToPartition_DefaultMembership(t *testing.T) {
	var reqBody ufmclient.PKeyAddGUIDsRequest
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&reqBody)
			w.WriteHeader(http.StatusOK)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.AddGUIDsToPartition(context.Background(), "0x1234", []string{"guid1"}, "")
	require.NoError(t, err)
	assert.Equal(t, "full", reqBody.Membership, "empty membership should default to full")
}

func TestAddGUIDsToPartition_APIError(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "fail"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.AddGUIDsToPartition(context.Background(), "0x1234", []string{"guid1"}, "full")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adding GUIDs")
}

// --- RemoveGUIDsFromPartition ---

func TestRemoveGUIDsFromPartition_Success(t *testing.T) {
	var deleteCalled bool
	srv := newTestUFMServer([]ufmRoute{
		{method: "DELETE", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			deleteCalled = true
			assert.Contains(t, r.URL.Path, "guid1,guid2")
			w.WriteHeader(http.StatusNoContent)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.RemoveGUIDsFromPartition(context.Background(), "0x1234", []string{"guid1", "guid2"})
	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestRemoveGUIDsFromPartition_APIError(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "DELETE", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "fail"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.RemoveGUIDsFromPartition(context.Background(), "0x1234", []string{"guid1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "removing GUIDs")
}

// --- AddHostsToPartition ---

func TestAddHostsToPartition_Success(t *testing.T) {
	var mu sync.Mutex
	jobPolled := 0

	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/hosts", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.Job{ID: "job-123", Status: "running"})
		}},
		{method: "GET", prefix: "/ufmRest/jobs/", handler: func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			jobPolled++
			count := jobPolled
			mu.Unlock()

			status := "running"
			if count >= 2 {
				status = "completed"
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.Job{ID: "job-123", Status: status})
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.AddHostsToPartition(context.Background(), "0x1234", []string{"host1", "host2"})
	require.NoError(t, err)
}

func TestAddHostsToPartition_APIError(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/hosts", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "fail"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.AddHostsToPartition(context.Background(), "0x1234", []string{"host1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adding hosts")
}

func TestAddHostsToPartition_JobFailed(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/hosts", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.Job{ID: "job-456", Status: "running"})
		}},
		{method: "GET", prefix: "/ufmRest/jobs/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.Job{ID: "job-456", Status: "failed"})
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	err := p.AddHostsToPartition(context.Background(), "0x1234", []string{"host1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waiting for host-add job")
}

func TestAddHostsToPartition_DefaultPollInterval(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "POST", prefix: "/ufmRest/resources/pkeys/hosts", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.Job{ID: "job-789", Status: "running"})
		}},
		{method: "GET", prefix: "/ufmRest/jobs/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.Job{ID: "job-789", Status: "completed"})
		}},
	})
	defer srv.Close()

	// Set JobPollInterval to 0 to test the default fallback (2s).
	// Use a short context timeout so we don't actually wait 2s.
	p := newTestProvider(srv.URL)
	p.config.JobPollInterval = 0

	err := p.AddHostsToPartition(context.Background(), "0x1234", []string{"host1"})
	require.NoError(t, err)
}

// --- validateIBPartitionConfig ---

func TestValidateIBPartitionConfig_PKEYNotFound(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "GET", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": "not found"}`)
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	payload := map[string]interface{}{"pkey": "0x1234"}
	err := p.validateIBPartitionConfig(context.Background(), payload)
	require.NoError(t, err, "PKEY not found should pass validation")
}

func TestValidateIBPartitionConfig_PKEYExists(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "GET", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.PKey{Partition: "existing-partition"})
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	payload := map[string]interface{}{"pkey": "0x1234"}
	err := p.validateIBPartitionConfig(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UFM validation failed")
	assert.Contains(t, err.Error(), "existing-partition")
}

func TestValidateIBPartitionConfig_PKEYExistsEmptyPartition(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "GET", prefix: "/ufmRest/resources/pkeys/", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.PKey{Partition: ""})
		}},
	})
	defer srv.Close()

	p := newTestProvider(srv.URL)
	payload := map[string]interface{}{"pkey": "0x1234"}
	err := p.validateIBPartitionConfig(context.Background(), payload)
	require.NoError(t, err, "empty partition name should pass validation")
}

func TestValidateIBPartitionConfig_MissingPKey(t *testing.T) {
	p := newTestProvider("http://unused")
	payload := map[string]interface{}{"name": "test"}
	err := p.validateIBPartitionConfig(context.Background(), payload)
	require.NoError(t, err, "missing pkey should skip validation")
}

func TestValidateIBPartitionConfig_NilPayload(t *testing.T) {
	p := newTestProvider("http://unused")
	err := p.validateIBPartitionConfig(context.Background(), nil)
	require.NoError(t, err, "nil payload should skip validation")
}

// --- Init with mock UFM server ---

func TestInit_Success(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "GET", prefix: "/ufmRest/app/ufm_version", handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ufmclient.UFMVersion{UFMReleaseVersion: "6.15.0"})
		}},
	})
	defer srv.Close()

	p := New(ProviderConfig{
		UFMURL:      srv.URL,
		UFMUsername:  "admin",
		UFMPassword:  "secret",
		Features:    FeatureConfig{SyncIBPartition: true},
	})

	ctx := provider.ProviderContext{
		Registry: provider.NewRegistry(),
	}
	err := p.Init(ctx)
	require.NoError(t, err)
}

func TestInit_ConnectivityCheckFails(t *testing.T) {
	srv := newTestUFMServer([]ufmRoute{
		{method: "GET", prefix: "/ufmRest/app/ufm_version", handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error": "unauthorized"}`)
		}},
	})
	defer srv.Close()

	p := New(ProviderConfig{
		UFMURL:      srv.URL,
		UFMUsername:  "admin",
		UFMPassword:  "wrong",
		Features:    FeatureConfig{SyncIBPartition: false},
	})

	ctx := provider.ProviderContext{
		Registry: provider.NewRegistry(),
	}
	err := p.Init(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "UFM connectivity check failed")
}
