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

package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bunotel"
)

// ConnectWithSchema opens a PostgreSQL connection scoped to the named schema.
// It creates the schema if it does not exist, then sets search_path so all
// subsequent queries run inside that schema without explicit qualification.
// The schema name must be a valid PostgreSQL identifier; the caller is
// responsible for validating it before passing it here.
func ConnectWithSchema(ctx context.Context, c Config, schema string) (*Session, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if schema == "" {
		return nil, fmt.Errorf("schema name is required")
	}

	// Append search_path to DSN so every connection in the pool uses the schema.
	dsn := c.BuildDSN() + "&search_path=" + schema

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool for schema %s: %w", schema, err)
	}

	db := bun.NewDB(stdlib.OpenDBFromPool(pool), pgdialect.New(), bun.WithDiscardUnknownColumns())

	if os.Getenv("TRACING_SERVICE_NAME") != "" {
		db.AddQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName(c.DBName),
			bunotel.WithFormattedQueries(true),
		))
	}

	// Create the schema if it does not already exist. This is safe to run on
	// every startup: IF NOT EXISTS makes it idempotent.
	if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %q", schema)); err != nil {
		pool.Close()
		return nil, fmt.Errorf("create schema %s: %w", schema, err)
	}

	return &Session{
		DBName:       c.DBName,
		DB:           db,
		pool:         pool,
		errorChecker: &PostgresErrorChecker{},
	}, nil
}
