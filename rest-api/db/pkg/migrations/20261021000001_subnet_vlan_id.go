// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/NVIDIA/infra-controller/rest-api/db/pkg/db/model"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		tx, terr := db.BeginTx(ctx, &sql.TxOptions{})
		if terr != nil {
			handlePanic(terr, "failed to begin transaction")
		}

		// Add vlan_id to subnet for L2 fabric switch configuration.
		// Maps each subnet to its 802.1Q VLAN tag on the physical fabric switches,
		// enabling the switch-config provider to program VLAN/SVI/VNI entries per subnet.
		// Nullable: existing subnets have no VLAN assignment; NULL means unassigned.
		_, err := tx.NewAddColumn().Model((*model.Subnet)(nil)).IfNotExists().ColumnExpr("vlan_id INT").Exec(ctx)
		handleError(tx, err)

		terr = tx.Commit()
		if terr != nil {
			handlePanic(terr, "failed to commit transaction")
		}

		fmt.Print(" [up migration] Added 'vlan_id' column to 'subnet' table successfully.")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		tx, terr := db.BeginTx(ctx, &sql.TxOptions{})
		if terr != nil {
			handlePanic(terr, "failed to begin transaction")
		}

		_, err := tx.NewDropColumn().Model((*model.Subnet)(nil)).Column("vlan_id").Exec(ctx)
		handleError(tx, err)

		terr = tx.Commit()
		if terr != nil {
			handlePanic(terr, "failed to commit transaction")
		}

		fmt.Print(" [down migration] Removed 'vlan_id' column from 'subnet' table successfully.")
		return nil
	})
}
