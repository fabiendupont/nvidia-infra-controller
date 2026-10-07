// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package migrations holds the ordered forward-only schema migrations for the
// health provider. All tables are created in the "health" schema.
package migrations

import (
	"context"

	"github.com/uptrace/bun"

	sdkdb "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// All returns the ordered list of migrations for the health schema.
func All() []sdkdb.Migration {
	return []sdkdb.Migration{
		{
			Name: "20261001000001_create_health_tables",
			Up:   createHealthTables,
		},
	}
}

func createHealthTables(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS fault_events (
			id                      TEXT        PRIMARY KEY,
			org_id                  TEXT        NOT NULL,
			tenant_id               TEXT,
			site_id                 TEXT        NOT NULL,
			machine_id              TEXT,
			instance_id             TEXT,
			source                  TEXT        NOT NULL,
			severity                TEXT        NOT NULL,
			component               TEXT        NOT NULL,
			classification          TEXT,
			message                 TEXT        NOT NULL,
			state                   TEXT        NOT NULL,
			detected_at             TIMESTAMPTZ NOT NULL,
			acknowledged_at         TIMESTAMPTZ,
			resolved_at             TIMESTAMPTZ,
			suppressed_until        TIMESTAMPTZ,
			remediation_workflow_id TEXT,
			remediation_attempts    INT         NOT NULL DEFAULT 0,
			escalation_level        INT         NOT NULL DEFAULT 0,
			metadata                JSONB,
			created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE INDEX IF NOT EXISTS fault_events_site_id_idx
			ON fault_events (site_id);

		CREATE INDEX IF NOT EXISTS fault_events_machine_id_idx
			ON fault_events (machine_id)
			WHERE machine_id IS NOT NULL;

		CREATE INDEX IF NOT EXISTS fault_events_state_severity_idx
			ON fault_events (state, severity);

		CREATE TABLE IF NOT EXISTS service_events (
			id                       TEXT        PRIMARY KEY,
			org_id                   TEXT        NOT NULL,
			tenant_id                TEXT        NOT NULL,
			instance_id              TEXT,
			summary                  TEXT        NOT NULL,
			impact                   TEXT        NOT NULL,
			state                    TEXT        NOT NULL,
			started_at               TIMESTAMPTZ NOT NULL,
			estimated_resolution_at  TIMESTAMPTZ,
			resolved_at              TIMESTAMPTZ,
			downtime_excluded        BOOL        NOT NULL DEFAULT false,
			created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE INDEX IF NOT EXISTS service_events_tenant_id_idx
			ON service_events (tenant_id);
	`)
	return err
}
