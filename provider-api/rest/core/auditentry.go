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
	"net/url"
	"time"
)

// APIAuditEntry is a data structure to capture audit log information
type APIAuditEntry struct {
	ID            string                 `json:"id"`
	Endpoint      string                 `json:"endpoint"`
	QueryParams   url.Values             `json:"queryParams"`
	Method        string                 `json:"method"`
	Body          map[string]interface{} `json:"body"`
	StatusCode    int                    `json:"statusCode"`
	StatusMessage string                 `json:"statusMessage"`
	ClientIP      string                 `json:"clientIP"`
	UserID        *string                `json:"userID"`
	User          *APIUser               `json:"user"`
	OrgName       string                 `json:"orgName"`
	ExtraData     map[string]interface{} `json:"extraData"`
	Timestamp     time.Time              `json:"timestamp"`
	DurationMs    int64                  `json:"durationMs"`
	APIVersion    string                 `json:"apiVersion"`
}
