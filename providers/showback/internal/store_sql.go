// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package showback

import (
	"context"
	"time"

	"github.com/google/uuid"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// UsageSQLStore is a PostgreSQL-backed usage store using the showback schema.
type UsageSQLStore struct {
	dao *usageRecordDAO
}

// NewUsageSQLStore creates a new SQL-backed usage store using a provider-sdk
// Session already scoped to the showback schema.
func NewUsageSQLStore(s *sdk.Session) *UsageSQLStore {
	return &UsageSQLStore{dao: newUsageRecordDAO(s)}
}

func (s *UsageSQLStore) StartMetering(tenantID, resourceID uuid.UUID, metricName string) {
	record := &dbUsageRecord{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ResourceID: resourceID,
		MetricName: metricName,
		StartTime:  time.Now().UTC(),
	}
	s.dao.create(context.Background(), record) //nolint:errcheck // fire-and-forget
}

func (s *UsageSQLStore) StopMetering(resourceID uuid.UUID) error {
	ctx := context.Background()
	record, err := s.dao.getByResourceID(ctx, resourceID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	record.EndTime = &now
	record.Value = now.Sub(record.StartTime).Hours()
	_, err = s.dao.update(ctx, record)
	return err
}

func (s *UsageSQLStore) GetUsageByTenant(tenantID uuid.UUID) UsageSummary {
	records, err := s.dao.getAllByTenant(context.Background(), tenantID)
	if err != nil {
		return UsageSummary{TenantID: tenantID, Period: "current-month", Metrics: map[string]float64{}}
	}
	metrics := make(map[string]float64)
	for _, r := range records {
		val := r.Value
		if r.EndTime == nil {
			val = time.Since(r.StartTime).Hours()
		}
		metrics[r.MetricName] += val
	}
	return UsageSummary{TenantID: tenantID, Period: "current-month", Metrics: metrics}
}

func (s *UsageSQLStore) GetUsageByService(serviceID uuid.UUID) UsageSummary {
	records, err := s.dao.getAllByService(context.Background(), serviceID)
	if err != nil {
		return UsageSummary{Period: "current-month", Metrics: map[string]float64{}}
	}
	metrics := make(map[string]float64)
	var tenantID uuid.UUID
	for _, r := range records {
		tenantID = r.TenantID
		val := r.Value
		if r.EndTime == nil {
			val = time.Since(r.StartTime).Hours()
		}
		metrics[r.MetricName] += val
	}
	return UsageSummary{TenantID: tenantID, Period: "current-month", Metrics: metrics}
}
