// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package provisioner

import "context"

// HookFirer is the interface that workflow activities use to fire lifecycle
// hooks. It is implemented by provider.HookRunner in the API binary.
// Defined here so site-workflow activities can reference it without importing
// the full rest-api/provider package (which carries API-layer dependencies
// not available in the site-agent binary).
type HookFirer interface {
	FireSync(ctx context.Context, feature, event string, payload interface{}) error
	FireAsync(ctx context.Context, feature, event string, payload interface{})
}
