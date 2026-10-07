// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package migrations holds the ordered forward-only schema migrations for the
// fulfillment provider. All tables are created in the "fulfillment" schema.
package migrations

import (
	"context"

	"github.com/uptrace/bun"

	sdkdb "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// All returns the ordered list of migrations for the fulfillment schema.
func All() []sdkdb.Migration {
	return []sdkdb.Migration{
		{
			Name: "20261001000000_create_fulfillment_tables",
			Up:   createFulfillmentTables,
		},
	}
}

func createFulfillmentTables(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			blueprint_id   UUID        NOT NULL,
			blueprint_name TEXT        NOT NULL,
			tenant_id      UUID        NOT NULL,
			parameters     JSONB,
			status         TEXT        NOT NULL,
			status_message TEXT        NOT NULL DEFAULT '',
			workflow_id    TEXT        NOT NULL DEFAULT '',
			service_id     UUID,
			created        TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated        TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS services (
			id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			order_id       UUID        NOT NULL,
			blueprint_id   UUID        NOT NULL,
			blueprint_name TEXT        NOT NULL,
			tenant_id      UUID        NOT NULL,
			name           TEXT        NOT NULL,
			status         TEXT        NOT NULL,
			resources      JSONB,
			created        TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated        TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}
