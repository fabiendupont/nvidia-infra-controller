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

import "time"

// APIInfrastructureProvider is the data structure to capture API representation of an Infrastructure Provider
type APIInfrastructureProvider struct {
	// ID is the unique UUID v4 identifier for the Infrastructure Provider
	ID string `json:"id"`
	// Org contains the name of the org this Infrastructure Provider belongs to
	Org string `json:"org"`
	// OrgDisplayName contains the display name of the org this Infrastructure Provider belongs to
	OrgDisplayName *string `json:"orgDisplayName"`
	// CreatedAt indicates the ISO datetime string for when the entity was created
	Created time.Time `json:"created"`
	// UpdatedAt indicates the ISO datetime string for when the entity was last updated
	Updated time.Time `json:"updated"`
}
