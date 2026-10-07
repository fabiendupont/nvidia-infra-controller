// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package showback

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// dbUsageRecord is the bun model for the showback.usage_record table.
type dbUsageRecord struct {
	bun.BaseModel `bun:"table:usage_record,alias:ur"`

	ID         uuid.UUID  `bun:"id,type:uuid,pk,default:gen_random_uuid()"`
	TenantID   uuid.UUID  `bun:"tenant_id,type:uuid,notnull"`
	ServiceID  uuid.UUID  `bun:"service_id,type:uuid"`
	ResourceID uuid.UUID  `bun:"resource_id,type:uuid,notnull"`
	MetricName string     `bun:"metric_name,notnull"`
	Value      float64    `bun:"value,notnull,default:0"`
	StartTime  time.Time  `bun:"start_time,notnull,default:current_timestamp"`
	EndTime    *time.Time `bun:"end_time"`
	Created    time.Time  `bun:"created,nullzero,notnull,default:current_timestamp"`
	Updated    time.Time  `bun:"updated,nullzero,notnull,default:current_timestamp"`
}

var _ bun.BeforeAppendModelHook = (*dbUsageRecord)(nil)

func (u *dbUsageRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().UTC()
	switch query.(type) {
	case *bun.InsertQuery:
		u.Created = now
		u.Updated = now
	case *bun.UpdateQuery:
		u.Updated = now
	}
	return nil
}

// usageRecordDAO provides data access for usage records.
type usageRecordDAO struct {
	db *bun.DB
}

func newUsageRecordDAO(s *sdk.Session) *usageRecordDAO {
	return &usageRecordDAO{db: s.DB}
}

func (d *usageRecordDAO) create(ctx context.Context, r *dbUsageRecord) (*dbUsageRecord, error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	_, err := d.db.NewInsert().Model(r).Exec(ctx)
	return r, err
}

func (d *usageRecordDAO) getByResourceID(ctx context.Context, resourceID uuid.UUID) (*dbUsageRecord, error) {
	r := new(dbUsageRecord)
	err := d.db.NewSelect().Model(r).
		Where("resource_id = ? AND end_time IS NULL", resourceID).
		OrderExpr("start_time DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *usageRecordDAO) update(ctx context.Context, r *dbUsageRecord) (*dbUsageRecord, error) {
	_, err := d.db.NewUpdate().Model(r).WherePK().Exec(ctx)
	return r, err
}

func (d *usageRecordDAO) getAllByTenant(ctx context.Context, tenantID uuid.UUID) ([]dbUsageRecord, error) {
	var records []dbUsageRecord
	err := d.db.NewSelect().Model(&records).Where("tenant_id = ?", tenantID).Scan(ctx)
	return records, err
}

func (d *usageRecordDAO) getAllByService(ctx context.Context, serviceID uuid.UUID) ([]dbUsageRecord, error) {
	var records []dbUsageRecord
	err := d.db.NewSelect().Model(&records).Where("service_id = ?", serviceID).Scan(ctx)
	return records, err
}
