// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package site

import tsdkWorker "go.temporal.io/sdk/worker"

// RegisterWorkflows is a no-op: all activities are registered by the existing
// workflow worker. In-tree providers share the main task queue.
func (p *SiteProvider) RegisterWorkflows(_ tsdkWorker.Worker) {}

// TaskQueue returns the empty string: in-tree providers do not manage their own
// Temporal task queue.
func (p *SiteProvider) TaskQueue() string { return "" }
