// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// --- Store interfaces ---
// Both in-memory and SQL stores implement these interfaces.

// FaultStoreI is the contract for fault event storage.
type FaultStoreI interface {
	Create(event *FaultEvent) error
	GetByID(id string) (*FaultEvent, error)
	GetAll(filter FaultEventFilter) []*FaultEvent
	Update(event *FaultEvent) (*FaultEvent, error)
	Delete(id string) error
	GetSummary() *FaultSummary
	ResolveMachineContext(machineID string) (*MachineContext, error)
	ListOpenCriticalByMachine(ctx context.Context, machineID string) ([]*FaultEvent, error)
	GetResolvedOlderThan(cutoff time.Time) []*FaultEvent
}

// ServiceEventStoreI is the contract for service event storage.
type ServiceEventStoreI interface {
	Create(event *ServiceEvent) (*ServiceEvent, error)
	GetByID(id string) (*ServiceEvent, error)
	GetByTenantID(tenantID string) []*ServiceEvent
	Update(event *ServiceEvent) (*ServiceEvent, error)
}

// FaultServiceEventStoreI is the contract for the fault↔service join table.
type FaultServiceEventStoreI interface {
	Link(faultID, serviceID string)
	GetServiceEventsByFaultID(faultID string) []string
	GetFaultEventsByServiceID(serviceID string) []string
}

// In-memory stores implement all three interfaces — verify at compile time.
var _ FaultStoreI = (*FaultEventStore)(nil)
var _ ServiceEventStoreI = (*ServiceEventStore)(nil)
var _ FaultServiceEventStoreI = (*FaultServiceEventStore)(nil)

// --- Bun model structs ---

type faultEventRow struct {
	bun.BaseModel `bun:"table:fault_events"`

	ID                    string                 `bun:"id,pk"`
	OrgID                 string                 `bun:"org_id,notnull"`
	TenantID              *string                `bun:"tenant_id"`
	SiteID                string                 `bun:"site_id,notnull"`
	MachineID             *string                `bun:"machine_id"`
	InstanceID            *string                `bun:"instance_id"`
	Source                string                 `bun:"source,notnull"`
	Severity              string                 `bun:"severity,notnull"`
	Component             string                 `bun:"component,notnull"`
	Classification        *string                `bun:"classification"`
	Message               string                 `bun:"message,notnull"`
	State                 string                 `bun:"state,notnull"`
	DetectedAt            time.Time              `bun:"detected_at,notnull"`
	AcknowledgedAt        *time.Time             `bun:"acknowledged_at"`
	ResolvedAt            *time.Time             `bun:"resolved_at"`
	SuppressedUntil       *time.Time             `bun:"suppressed_until"`
	RemediationWorkflowID *string                `bun:"remediation_workflow_id"`
	RemediationAttempts   int                    `bun:"remediation_attempts,notnull,default:0"`
	EscalationLevel       int                    `bun:"escalation_level,notnull,default:0"`
	Metadata              map[string]interface{} `bun:"metadata,type:jsonb"`
	CreatedAt             time.Time              `bun:"created_at,notnull,default:now()"`
	UpdatedAt             time.Time              `bun:"updated_at,notnull,default:now()"`
}

type serviceEventRow struct {
	bun.BaseModel `bun:"table:service_events"`

	ID                    string     `bun:"id,pk"`
	OrgID                 string     `bun:"org_id,notnull"`
	TenantID              string     `bun:"tenant_id,notnull"`
	InstanceID            *string    `bun:"instance_id"`
	Summary               string     `bun:"summary,notnull"`
	Impact                string     `bun:"impact,notnull"`
	State                 string     `bun:"state,notnull"`
	StartedAt             time.Time  `bun:"started_at,notnull"`
	EstimatedResolutionAt *time.Time `bun:"estimated_resolution_at"`
	ResolvedAt            *time.Time `bun:"resolved_at"`
	DowntimeExcluded      bool       `bun:"downtime_excluded,notnull,default:false"`
	CreatedAt             time.Time  `bun:"created_at,notnull,default:now()"`
	UpdatedAt             time.Time  `bun:"updated_at,notnull,default:now()"`
}

// --- FaultEventSQLStore ---

// FaultEventSQLStore is a PostgreSQL-backed fault event store.
type FaultEventSQLStore struct {
	db *bun.DB
}

// NewFaultEventSQLStore creates a fault event store backed by the given bun DB.
func NewFaultEventSQLStore(db *bun.DB) *FaultEventSQLStore {
	return &FaultEventSQLStore{db: db}
}

func (s *FaultEventSQLStore) Create(event *FaultEvent) error {
	row := faultEventToRow(event)
	if _, err := s.db.NewInsert().Model(row).Returning("*").Exec(context.Background()); err != nil {
		return fmt.Errorf("create fault event: %w", err)
	}
	event.CreatedAt = row.CreatedAt
	event.UpdatedAt = row.UpdatedAt
	return nil
}

func (s *FaultEventSQLStore) GetByID(id string) (*FaultEvent, error) {
	row := new(faultEventRow)
	if err := s.db.NewSelect().Model(row).Where("id = ?", id).Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("fault event %s not found", id)
	}
	return rowToFaultEvent(row), nil
}

func (s *FaultEventSQLStore) GetAll(filter FaultEventFilter) []*FaultEvent {
	q := s.db.NewSelect().Model((*faultEventRow)(nil))
	if filter.SiteID != nil {
		q = q.Where("site_id = ?", *filter.SiteID)
	}
	if filter.MachineID != nil {
		q = q.Where("machine_id = ?", *filter.MachineID)
	}
	if filter.Source != nil {
		q = q.Where("source = ?", *filter.Source)
	}
	if len(filter.Severity) > 0 {
		q = q.Where("severity IN (?)", bun.In(filter.Severity))
	}
	if len(filter.Component) > 0 {
		q = q.Where("component IN (?)", bun.In(filter.Component))
	}
	if len(filter.State) > 0 {
		q = q.Where("state IN (?)", bun.In(filter.State))
	}
	if filter.DetectedAfter != nil {
		q = q.Where("detected_at >= ?", *filter.DetectedAfter)
	}
	if filter.DetectedBefore != nil {
		q = q.Where("detected_at <= ?", *filter.DetectedBefore)
	}

	var rows []faultEventRow
	if err := q.Scan(context.Background(), &rows); err != nil {
		return nil
	}
	result := make([]*FaultEvent, len(rows))
	for i := range rows {
		result[i] = rowToFaultEvent(&rows[i])
	}
	return result
}

func (s *FaultEventSQLStore) Update(event *FaultEvent) (*FaultEvent, error) {
	row := faultEventToRow(event)
	row.UpdatedAt = time.Now()
	if _, err := s.db.NewUpdate().Model(row).WherePK().Returning("*").Exec(context.Background()); err != nil {
		return nil, fmt.Errorf("update fault event %s: %w", event.ID, err)
	}
	return rowToFaultEvent(row), nil
}

func (s *FaultEventSQLStore) Delete(id string) error {
	if _, err := s.db.NewDelete().Model((*faultEventRow)(nil)).Where("id = ?", id).Exec(context.Background()); err != nil {
		return fmt.Errorf("delete fault event %s: %w", id, err)
	}
	return nil
}

func (s *FaultEventSQLStore) GetSummary() *FaultSummary {
	events := s.GetAll(FaultEventFilter{})
	summary := &FaultSummary{
		BySeverity:  make(map[string]int),
		ByComponent: make(map[string]int),
		ByState:     make(map[string]int),
	}
	siteMap := make(map[string]*SiteFaultSummary)
	for _, e := range events {
		summary.BySeverity[e.Severity]++
		summary.ByComponent[e.Component]++
		summary.ByState[e.State]++
		site, ok := siteMap[e.SiteID]
		if !ok {
			site = &SiteFaultSummary{SiteID: e.SiteID}
			siteMap[e.SiteID] = site
		}
		if e.State == FaultStateOpen || e.State == FaultStateAcknowledged || e.State == FaultStateRemediating {
			site.Open++
		}
		if e.Severity == SeverityCritical {
			site.Critical++
		}
	}
	for _, s := range siteMap {
		summary.BySite = append(summary.BySite, *s)
	}
	return summary
}

// ResolveMachineContext is not backed by SQL (requires cross-domain lookup).
// Returns nil context; handlers fall back to OrgID/SiteID from the request.
func (s *FaultEventSQLStore) ResolveMachineContext(_ string) (*MachineContext, error) {
	return nil, nil
}

func (s *FaultEventSQLStore) ListOpenCriticalByMachine(ctx context.Context, machineID string) ([]*FaultEvent, error) {
	var rows []faultEventRow
	err := s.db.NewSelect().
		Model(&rows).
		Where("machine_id = ?", machineID).
		Where("severity = ?", SeverityCritical).
		Where("state IN (?)", bun.In([]string{FaultStateOpen, FaultStateAcknowledged, FaultStateRemediating})).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*FaultEvent, len(rows))
	for i := range rows {
		result[i] = rowToFaultEvent(&rows[i])
	}
	return result, nil
}

func (s *FaultEventSQLStore) GetResolvedOlderThan(cutoff time.Time) []*FaultEvent {
	var rows []faultEventRow
	if err := s.db.NewSelect().
		Model(&rows).
		Where("state = ?", FaultStateResolved).
		Where("resolved_at < ?", cutoff).
		Scan(context.Background()); err != nil {
		return nil
	}
	result := make([]*FaultEvent, len(rows))
	for i := range rows {
		result[i] = rowToFaultEvent(&rows[i])
	}
	return result
}

// --- ServiceEventSQLStore ---

// ServiceEventSQLStore is a PostgreSQL-backed service event store.
type ServiceEventSQLStore struct {
	db *bun.DB
}

// NewServiceEventSQLStore creates a service event store backed by the given bun DB.
func NewServiceEventSQLStore(db *bun.DB) *ServiceEventSQLStore {
	return &ServiceEventSQLStore{db: db}
}

func (s *ServiceEventSQLStore) Create(event *ServiceEvent) (*ServiceEvent, error) {
	row := serviceEventToRow(event)
	if _, err := s.db.NewInsert().Model(row).Returning("*").Exec(context.Background()); err != nil {
		return nil, fmt.Errorf("create service event: %w", err)
	}
	return rowToServiceEvent(row), nil
}

func (s *ServiceEventSQLStore) GetByID(id string) (*ServiceEvent, error) {
	row := new(serviceEventRow)
	if err := s.db.NewSelect().Model(row).Where("id = ?", id).Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("service event %s not found", id)
	}
	return rowToServiceEvent(row), nil
}

func (s *ServiceEventSQLStore) GetByTenantID(tenantID string) []*ServiceEvent {
	var rows []serviceEventRow
	if err := s.db.NewSelect().Model(&rows).Where("tenant_id = ?", tenantID).Scan(context.Background()); err != nil {
		return nil
	}
	result := make([]*ServiceEvent, len(rows))
	for i := range rows {
		result[i] = rowToServiceEvent(&rows[i])
	}
	return result
}

func (s *ServiceEventSQLStore) Update(event *ServiceEvent) (*ServiceEvent, error) {
	row := serviceEventToRow(event)
	row.UpdatedAt = time.Now()
	if _, err := s.db.NewUpdate().Model(row).WherePK().Returning("*").Exec(context.Background()); err != nil {
		return nil, fmt.Errorf("update service event %s: %w", event.ID, err)
	}
	return rowToServiceEvent(row), nil
}

// --- Conversion helpers ---

func faultEventToRow(e *FaultEvent) *faultEventRow {
	return &faultEventRow{
		ID:                    e.ID,
		OrgID:                 e.OrgID,
		TenantID:              e.TenantID,
		SiteID:                e.SiteID,
		MachineID:             e.MachineID,
		InstanceID:            e.InstanceID,
		Source:                e.Source,
		Severity:              e.Severity,
		Component:             e.Component,
		Classification:        e.Classification,
		Message:               e.Message,
		State:                 e.State,
		DetectedAt:            e.DetectedAt,
		AcknowledgedAt:        e.AcknowledgedAt,
		ResolvedAt:            e.ResolvedAt,
		SuppressedUntil:       e.SuppressedUntil,
		RemediationWorkflowID: e.RemediationWorkflowID,
		RemediationAttempts:   e.RemediationAttempts,
		EscalationLevel:       e.EscalationLevel,
		Metadata:              e.Metadata,
		CreatedAt:             e.CreatedAt,
		UpdatedAt:             e.UpdatedAt,
	}
}

func rowToFaultEvent(row *faultEventRow) *FaultEvent {
	return &FaultEvent{
		ID:                    row.ID,
		OrgID:                 row.OrgID,
		TenantID:              row.TenantID,
		SiteID:                row.SiteID,
		MachineID:             row.MachineID,
		InstanceID:            row.InstanceID,
		Source:                row.Source,
		Severity:              row.Severity,
		Component:             row.Component,
		Classification:        row.Classification,
		Message:               row.Message,
		State:                 row.State,
		DetectedAt:            row.DetectedAt,
		AcknowledgedAt:        row.AcknowledgedAt,
		ResolvedAt:            row.ResolvedAt,
		SuppressedUntil:       row.SuppressedUntil,
		RemediationWorkflowID: row.RemediationWorkflowID,
		RemediationAttempts:   row.RemediationAttempts,
		EscalationLevel:       row.EscalationLevel,
		Metadata:              row.Metadata,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func serviceEventToRow(e *ServiceEvent) *serviceEventRow {
	return &serviceEventRow{
		ID:                    e.ID,
		OrgID:                 e.OrgID,
		TenantID:              e.TenantID,
		InstanceID:            e.InstanceID,
		Summary:               e.Summary,
		Impact:                e.Impact,
		State:                 e.State,
		StartedAt:             e.StartedAt,
		EstimatedResolutionAt: e.EstimatedResolutionAt,
		ResolvedAt:            e.ResolvedAt,
		DowntimeExcluded:      e.DowntimeExcluded,
		CreatedAt:             e.CreatedAt,
		UpdatedAt:             e.UpdatedAt,
	}
}

func rowToServiceEvent(row *serviceEventRow) *ServiceEvent {
	return &ServiceEvent{
		ID:                    row.ID,
		OrgID:                 row.OrgID,
		TenantID:              row.TenantID,
		InstanceID:            row.InstanceID,
		Summary:               row.Summary,
		Impact:                row.Impact,
		State:                 row.State,
		StartedAt:             row.StartedAt,
		EstimatedResolutionAt: row.EstimatedResolutionAt,
		ResolvedAt:            row.ResolvedAt,
		DowntimeExcluded:      row.DowntimeExcluded,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}
