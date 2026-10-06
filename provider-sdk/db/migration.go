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

// Package db provides PostgreSQL connection and migration helpers for NICo
// providers. Each provider uses a dedicated PostgreSQL schema so that
// migrations, models, and tracking tables are isolated from NICo core and
// other providers.
package db

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// Migration represents a single forward-only schema migration.
type Migration struct {
	// Name is a unique identifier, typically a timestamp prefix:
	// "20261001000000_create_blueprint_table".
	Name string

	// Up applies the migration. It receives the bun.DB already scoped to the
	// provider schema via search_path.
	Up func(ctx context.Context, db *bun.DB) error
}

// Migrator runs ordered migrations for a provider schema.
type Migrator struct {
	db         *bun.DB
	schema     string
	migrations []Migration
}

// NewMigrator creates a Migrator scoped to the named schema. db must be
// obtained via ConnectWithSchema so search_path is already set.
func NewMigrator(db *bun.DB, schema string, migrations []Migration) *Migrator {
	return &Migrator{db: db, schema: schema, migrations: migrations}
}

// Run applies all migrations that have not yet been recorded in the
// schema-local tracking table. Migrations are applied in the order they are
// registered; the table is created on first run.
func (m *Migrator) Run(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return err
	}
	applied, err := m.appliedNames(ctx)
	if err != nil {
		return err
	}
	for _, mig := range m.migrations {
		if applied[mig.Name] {
			continue
		}
		if err := mig.Up(ctx, m.db); err != nil {
			return fmt.Errorf("migration %s: %w", mig.Name, err)
		}
		if _, err := m.db.ExecContext(ctx,
			"INSERT INTO provider_migrations (name) VALUES (?)", mig.Name,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", mig.Name, err)
		}
	}
	return nil
}

func (m *Migrator) ensureTable(ctx context.Context) error {
	// Table is created inside the schema because search_path is already set.
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS provider_migrations (
			name       TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

func (m *Migrator) appliedNames(ctx context.Context) (map[string]bool, error) {
	var names []string
	rows, err := m.db.QueryContext(ctx, "SELECT name FROM provider_migrations")
	if err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(names))
	for _, n := range names {
		result[n] = true
	}
	return result, nil
}
