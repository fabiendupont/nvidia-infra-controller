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

	"github.com/NVIDIA/infra-controller/rest-api/provider"
)

// registerHooks registers hooks and reactions for the fault management system.
func (p *Server) registerHooks(registry *provider.Registry) {
	// After a fault is ingested, start remediation workflow
	registry.RegisterReaction(provider.Reaction{
		Feature:        "health",
		Event:          provider.EventPostHealthEventIngested,
		TargetWorkflow: "health-fault-remediation",
		SignalName:     "fault-ingested",
	})

	// After remediation completes, notify watcher workflow
	registry.RegisterReaction(provider.Reaction{
		Feature:        "health",
		Event:          provider.EventPostFaultRemediation,
		TargetWorkflow: "health-fault-watcher",
		SignalName:     "fault-remediated",
	})

	// After a fault is resolved, notify watcher workflow
	registry.RegisterReaction(provider.Reaction{
		Feature:        "health",
		Event:          provider.EventPostFaultResolved,
		TargetWorkflow: "health-fault-watcher",
		SignalName:     "fault-resolved",
	})

	// After a fault is escalated, notify watcher workflow
	registry.RegisterReaction(provider.Reaction{
		Feature:        "health",
		Event:          provider.EventPostFaultEscalated,
		TargetWorkflow: "health-fault-watcher",
		SignalName:     "fault-escalated",
	})

	// Block instance creation on machines with open critical faults
	registry.RegisterHook(provider.SyncHook{
		Feature: "compute",
		Event:   provider.EventPreCreateInstance,
		Handler: p.blockInstanceOnFaultyMachine,
	})
}
