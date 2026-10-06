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

package compute

import "time"

// APIDeprecation captures API representation of a deprecation message
type APIDeprecation struct {
	// Attribute denotes the attribute that is deprecated (optional)
	Attribute *string `json:"attribute,omitempty"`
	// QueryParam denotes the query param that is deprecated (optional)
	QueryParam *string `json:"queryparam,omitempty"`
	// Endpoint denotes the endpoint that is deprecated (optional)
	Endpoint *string `json:"endpoint,omitempty"`
	// ReplacedBy denotes the field that replaces the deprecated field (optional)
	ReplacedBy *string `json:"replacedby"`
	// TakeActionBy indicates the ISO datetime string for when the deprecation takes effect
	TakeActionBy time.Time `json:"effective"`
	// Notice describes the deprecated field
	Notice string `json:"notice"`
}
