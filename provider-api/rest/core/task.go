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

package core

import (
	"fmt"
	"time"
)

// Rack task status names
const (
	APIRackTaskStatusUnknown    = "Unknown"
	APIRackTaskStatusPending    = "Pending"
	APIRackTaskStatusRunning    = "Running"
	APIRackTaskStatusSucceeded  = "Succeeded"
	APIRackTaskStatusFailed     = "Failed"
	APIRackTaskStatusTerminated = "Terminated"
	APIRackTaskStatusWaiting    = "Waiting"
)

// APIRackTask is the API response model for a rack task (OpenAPI schema RackTask).
type APIRackTask struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	Message     string     `json:"message"`
	Started     *time.Time `json:"started"`
	Finished    *time.Time `json:"finished"`
	Created     time.Time  `json:"created"`
	Updated     time.Time  `json:"updated"`
}

// APIGetTaskRequest captures query parameters for getting a task by ID.
type APIGetTaskRequest struct {
	SiteID string `query:"siteId"`
}

// Validate ensures the values passed in the request are acceptable.
func (r *APIGetTaskRequest) Validate() error {
	if r.SiteID == "" {
		return fmt.Errorf("siteId query parameter is required")
	}
	return nil
}
