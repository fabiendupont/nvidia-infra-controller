// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"

	"github.com/uptrace/bun"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// catalogMigrations are the ordered forward migrations for the catalog schema.
var catalogMigrations = []sdk.Migration{
	{
		Name: "20261001000000_create_blueprint_table",
		Up: func(ctx context.Context, db *bun.DB) error {
			_, err := db.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS blueprint (
					id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
					name        TEXT        NOT NULL,
					version     TEXT        NOT NULL DEFAULT '1.0.0',
					description TEXT,
					parameters  JSONB,
					resources   JSONB,
					labels      JSONB,
					pricing     JSONB,
					tenant_id   UUID,
					visibility  TEXT        NOT NULL DEFAULT 'public',
					based_on    TEXT,
					is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
					created     TIMESTAMPTZ NOT NULL DEFAULT now(),
					updated     TIMESTAMPTZ NOT NULL DEFAULT now(),
					deleted     TIMESTAMPTZ
				)
			`)
			return err
		},
	},
}
