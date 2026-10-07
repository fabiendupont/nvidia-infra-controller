// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package showback

import (
	"context"

	"github.com/uptrace/bun"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// showbackMigrations are the ordered forward migrations for the showback schema.
var showbackMigrations = []sdk.Migration{
	{
		Name: "20261001000000_create_usage_record_table",
		Up: func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS usage_record (
					id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
					tenant_id   UUID        NOT NULL,
					service_id  UUID,
					resource_id UUID        NOT NULL,
					metric_name TEXT        NOT NULL,
					value       FLOAT8      NOT NULL DEFAULT 0,
					start_time  TIMESTAMPTZ NOT NULL DEFAULT now(),
					end_time    TIMESTAMPTZ,
					created     TIMESTAMPTZ NOT NULL DEFAULT now(),
					updated     TIMESTAMPTZ NOT NULL DEFAULT now()
				)
			`)
			return err
		},
	},
}
