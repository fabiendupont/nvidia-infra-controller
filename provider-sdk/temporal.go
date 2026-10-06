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

package sdk

import (
	"fmt"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// NewTemporalClient creates a Temporal client. When endpoint or namespace are
// empty, values are read from TEMPORAL_ENDPOINT and TEMPORAL_NAMESPACE
// environment variables.
func NewTemporalClient(endpoint, namespace string) (client.Client, error) {
	if endpoint == "" {
		endpoint = os.Getenv("TEMPORAL_ENDPOINT")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("temporal endpoint not configured")
	}

	if namespace == "" {
		namespace = os.Getenv("TEMPORAL_NAMESPACE")
	}
	if namespace == "" {
		namespace = "default"
	}

	c, err := client.Dial(client.Options{
		HostPort:  endpoint,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("dial temporal at %s: %w", endpoint, err)
	}

	return c, nil
}

// NewTemporalWorker creates a Temporal worker for the given task queue.
func NewTemporalWorker(tc client.Client, taskQueue string) worker.Worker {
	return worker.New(tc, taskQueue, worker.Options{})
}
