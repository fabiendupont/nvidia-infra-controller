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

// APISSHKeyAssociation is the data structure to capture API representation of an sshkey association
type APISSHKeyAssociation struct {
	// ID is the unique UUID v4 identifier for the security policy
	ID string `json:"id"`
	// SSHKeyID is the ID of the associated SSHKey
	SSHKeyID string `json:"sshKeyId"`
	// SSHKeyGroupID is the ID of the SSHKeyGroup
	SSHKeyGroupID string `json:"entityId"`
	// Created indicates the ISO datetime string for when the site was created
	Created time.Time `json:"created"`
	// Updated indicates the ISO datetime string for when the site was last updated
	Updated time.Time `json:"updated"`
}
