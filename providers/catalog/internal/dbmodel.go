// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// dbBlueprint is the bun model for the catalog.blueprint table.
type dbBlueprint struct {
	bun.BaseModel `bun:"table:blueprint,alias:bp"`

	ID          uuid.UUID              `bun:"id,type:uuid,pk,default:gen_random_uuid()"`
	Name        string                 `bun:"name,notnull"`
	Version     string                 `bun:"version,notnull,default:'1.0.0'"`
	Description string                 `bun:"description"`
	Parameters  map[string]interface{} `bun:"parameters,type:jsonb"`
	Resources   map[string]interface{} `bun:"resources,type:jsonb"`
	Labels      map[string]string      `bun:"labels,type:jsonb"`
	Pricing     map[string]interface{} `bun:"pricing,type:jsonb"`
	TenantID    *uuid.UUID             `bun:"tenant_id,type:uuid"`
	Visibility  string                 `bun:"visibility,notnull,default:'public'"`
	BasedOn     *string                `bun:"based_on"`
	IsActive    bool                   `bun:"is_active,notnull,default:true"`
	Created     time.Time              `bun:"created,nullzero,notnull,default:current_timestamp"`
	Updated     time.Time              `bun:"updated,nullzero,notnull,default:current_timestamp"`
	Deleted     *time.Time             `bun:"deleted,soft_delete"`
}

var _ bun.BeforeAppendModelHook = (*dbBlueprint)(nil)

func (b *dbBlueprint) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().UTC()
	switch query.(type) {
	case *bun.InsertQuery:
		b.Created = now
		b.Updated = now
	case *bun.UpdateQuery:
		b.Updated = now
	}
	return nil
}

// blueprintDAO provides data access for dbBlueprint against a provider-sdk Session.
type blueprintDAO struct {
	db *bun.DB
}

func newBlueprintDAO(s *sdk.Session) *blueprintDAO {
	return &blueprintDAO{db: s.DB}
}

func (d *blueprintDAO) create(ctx context.Context, b *dbBlueprint) (*dbBlueprint, error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	_, err := d.db.NewInsert().Model(b).Exec(ctx)
	return b, err
}

func (d *blueprintDAO) getByID(ctx context.Context, id uuid.UUID) (*dbBlueprint, error) {
	b := new(dbBlueprint)
	err := d.db.NewSelect().Model(b).Where("id = ?", id).WhereAllWithDeleted().Scan(ctx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (d *blueprintDAO) getByNameVersion(ctx context.Context, name, version string) (*dbBlueprint, error) {
	b := new(dbBlueprint)
	q := d.db.NewSelect().Model(b).Where("name = ?", name)
	if version != "" {
		q = q.Where("version = ?", version)
	} else {
		q = q.OrderExpr("created DESC").Limit(1)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

func (d *blueprintDAO) getAll(ctx context.Context, isActive *bool) ([]dbBlueprint, error) {
	var results []dbBlueprint
	q := d.db.NewSelect().Model(&results)
	if isActive != nil {
		q = q.Where("is_active = ?", *isActive)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return results, nil
}

func (d *blueprintDAO) update(ctx context.Context, b *dbBlueprint) (*dbBlueprint, error) {
	_, err := d.db.NewUpdate().Model(b).WherePK().Exec(ctx)
	return b, err
}

func (d *blueprintDAO) deleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := d.db.NewDelete().Model(&dbBlueprint{}).Where("id = ?", id).Exec(ctx)
	return err
}
