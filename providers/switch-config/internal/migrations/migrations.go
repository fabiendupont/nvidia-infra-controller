// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"context"

	"github.com/uptrace/bun"
	sdkdb "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// All returns the ordered list of migrations for the switch_config schema.
func All() []sdkdb.Migration {
	return []sdkdb.Migration{
		{
			Name: "20261010000000_create_switch_config_tables",
			Up: func(ctx context.Context, db *bun.DB) error {
				_, err := db.ExecContext(ctx, `
					CREATE TABLE IF NOT EXISTS switch_configs (
						id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
						switch_id     TEXT        NOT NULL,
						site_id       TEXT        NOT NULL,
						format        TEXT        NOT NULL,
						content_hash  TEXT        NOT NULL,
						content       BYTEA       NOT NULL,
						author        TEXT        NOT NULL DEFAULT '',
						commit_msg    TEXT        NOT NULL DEFAULT '',
						created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
						UNIQUE (switch_id, format, content_hash)
					);
					CREATE INDEX IF NOT EXISTS switch_configs_switch_id_format_idx
						ON switch_configs (switch_id, format, created_at DESC);

					CREATE TABLE IF NOT EXISTS switch_validation_results (
						id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
						site_id       TEXT        NOT NULL,
						switch_id     TEXT        NOT NULL,
						port          TEXT        NOT NULL,
						expected_peer TEXT        NOT NULL DEFAULT '',
						observed_peer TEXT        NOT NULL DEFAULT '',
						match         BOOLEAN     NOT NULL,
						validated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
					);
					CREATE INDEX IF NOT EXISTS switch_validation_results_site_idx
						ON switch_validation_results (site_id, validated_at DESC);
				`)
				return err
			},
		},
	}
}
