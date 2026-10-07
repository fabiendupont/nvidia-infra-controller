// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package fulfillment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// --- Bun model structs (not exported; only used for DB I/O) ---

type orderRow struct {
	bun.BaseModel `bun:"table:orders"`

	ID            uuid.UUID              `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	BlueprintID   uuid.UUID              `bun:"blueprint_id,type:uuid,notnull"`
	BlueprintName string                 `bun:"blueprint_name,notnull"`
	TenantID      uuid.UUID              `bun:"tenant_id,type:uuid,notnull"`
	Parameters    map[string]interface{} `bun:"parameters,type:jsonb"`
	Status        string                 `bun:"status,notnull"`
	StatusMessage string                 `bun:"status_message,notnull,default:''"`
	WorkflowID    string                 `bun:"workflow_id,notnull,default:''"`
	ServiceID     *uuid.UUID             `bun:"service_id,type:uuid"`
	Created       time.Time              `bun:"created,notnull,default:now()"`
	Updated       time.Time              `bun:"updated,notnull,default:now()"`
}

type serviceRow struct {
	bun.BaseModel `bun:"table:services"`

	ID            uuid.UUID         `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	OrderID       uuid.UUID         `bun:"order_id,type:uuid,notnull"`
	BlueprintID   uuid.UUID         `bun:"blueprint_id,type:uuid,notnull"`
	BlueprintName string            `bun:"blueprint_name,notnull"`
	TenantID      uuid.UUID         `bun:"tenant_id,type:uuid,notnull"`
	Name          string            `bun:"name,notnull"`
	Status        string            `bun:"status,notnull"`
	Resources     map[string]string `bun:"resources,type:jsonb"`
	Created       time.Time         `bun:"created,notnull,default:now()"`
	Updated       time.Time         `bun:"updated,notnull,default:now()"`
}

// --- OrderSQLStore ---

// OrderSQLStore is a PostgreSQL-backed order store.
type OrderSQLStore struct {
	db *bun.DB
}

// NewOrderSQLStore creates an order store backed by the given bun DB.
// db must be scoped to the fulfillment schema via ConnectWithSchema.
func NewOrderSQLStore(db *bun.DB) *OrderSQLStore {
	return &OrderSQLStore{db: db}
}

func (s *OrderSQLStore) Create(order *Order) error {
	row := &orderRow{
		BlueprintID:   order.BlueprintID,
		BlueprintName: order.BlueprintName,
		TenantID:      order.TenantID,
		Parameters:    order.Parameters,
		Status:        string(order.Status),
		StatusMessage: order.StatusMessage,
		WorkflowID:    order.WorkflowID,
		ServiceID:     order.ServiceID,
	}
	if _, err := s.db.NewInsert().Model(row).Returning("*").Exec(context.Background()); err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	order.ID = row.ID
	order.Created = row.Created
	order.Updated = row.Updated
	return nil
}

func (s *OrderSQLStore) Get(id uuid.UUID) (*Order, error) {
	row := new(orderRow)
	if err := s.db.NewSelect().Model(row).Where("id = ?", id).Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("order %s not found", id)
	}
	return rowToOrder(row), nil
}

func (s *OrderSQLStore) Update(order *Order) error {
	row := orderToRow(order)
	row.Updated = time.Now()
	if _, err := s.db.NewUpdate().Model(row).WherePK().Returning("updated").Exec(context.Background()); err != nil {
		return fmt.Errorf("update order %s: %w", order.ID, err)
	}
	order.Updated = row.Updated
	return nil
}

func (s *OrderSQLStore) Delete(id uuid.UUID) error {
	if _, err := s.db.NewDelete().Model((*orderRow)(nil)).Where("id = ?", id).Exec(context.Background()); err != nil {
		return fmt.Errorf("delete order %s: %w", id, err)
	}
	return nil
}

func (s *OrderSQLStore) List() []*Order {
	var rows []orderRow
	if err := s.db.NewSelect().Model(&rows).Scan(context.Background()); err != nil {
		return nil
	}
	result := make([]*Order, len(rows))
	for i := range rows {
		result[i] = rowToOrder(&rows[i])
	}
	return result
}

func (s *OrderSQLStore) ListByTenant(tenantID uuid.UUID) []*Order {
	var rows []orderRow
	if err := s.db.NewSelect().Model(&rows).Where("tenant_id = ?", tenantID).Scan(context.Background()); err != nil {
		return nil
	}
	result := make([]*Order, len(rows))
	for i := range rows {
		result[i] = rowToOrder(&rows[i])
	}
	return result
}

// --- ServiceSQLStore ---

// ServiceSQLStore is a PostgreSQL-backed service store.
type ServiceSQLStore struct {
	db *bun.DB
}

// NewServiceSQLStore creates a service store backed by the given bun DB.
func NewServiceSQLStore(db *bun.DB) *ServiceSQLStore {
	return &ServiceSQLStore{db: db}
}

func (s *ServiceSQLStore) Create(svc *Service) error {
	row := serviceToRow(svc)
	if _, err := s.db.NewInsert().Model(row).Returning("*").Exec(context.Background()); err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	svc.ID = row.ID
	svc.Created = row.Created
	svc.Updated = row.Updated
	return nil
}

func (s *ServiceSQLStore) Get(id uuid.UUID) (*Service, error) {
	row := new(serviceRow)
	if err := s.db.NewSelect().Model(row).Where("id = ?", id).Scan(context.Background()); err != nil {
		return nil, fmt.Errorf("service %s not found", id)
	}
	return rowToService(row), nil
}

func (s *ServiceSQLStore) Update(svc *Service) error {
	row := serviceToRow(svc)
	row.Updated = time.Now()
	if _, err := s.db.NewUpdate().Model(row).WherePK().Returning("updated").Exec(context.Background()); err != nil {
		return fmt.Errorf("update service %s: %w", svc.ID, err)
	}
	svc.Updated = row.Updated
	return nil
}

func (s *ServiceSQLStore) Delete(id uuid.UUID) error {
	if _, err := s.db.NewDelete().Model((*serviceRow)(nil)).Where("id = ?", id).Exec(context.Background()); err != nil {
		return fmt.Errorf("delete service %s: %w", id, err)
	}
	return nil
}

func (s *ServiceSQLStore) List() []*Service {
	var rows []serviceRow
	if err := s.db.NewSelect().Model(&rows).Scan(context.Background()); err != nil {
		return nil
	}
	result := make([]*Service, len(rows))
	for i := range rows {
		result[i] = rowToService(&rows[i])
	}
	return result
}

func (s *ServiceSQLStore) ListByTenant(tenantID uuid.UUID) []*Service {
	var rows []serviceRow
	if err := s.db.NewSelect().Model(&rows).Where("tenant_id = ?", tenantID).Scan(context.Background()); err != nil {
		return nil
	}
	result := make([]*Service, len(rows))
	for i := range rows {
		result[i] = rowToService(&rows[i])
	}
	return result
}

// --- Conversion helpers ---

func orderToRow(o *Order) *orderRow {
	return &orderRow{
		ID:            o.ID,
		BlueprintID:   o.BlueprintID,
		BlueprintName: o.BlueprintName,
		TenantID:      o.TenantID,
		Parameters:    o.Parameters,
		Status:        string(o.Status),
		StatusMessage: o.StatusMessage,
		WorkflowID:    o.WorkflowID,
		ServiceID:     o.ServiceID,
		Created:       o.Created,
		Updated:       o.Updated,
	}
}

func rowToOrder(row *orderRow) *Order {
	return &Order{
		ID:            row.ID,
		BlueprintID:   row.BlueprintID,
		BlueprintName: row.BlueprintName,
		TenantID:      row.TenantID,
		Parameters:    row.Parameters,
		Status:        OrderStatus(row.Status),
		StatusMessage: row.StatusMessage,
		WorkflowID:    row.WorkflowID,
		ServiceID:     row.ServiceID,
		Created:       row.Created,
		Updated:       row.Updated,
	}
}

func serviceToRow(s *Service) *serviceRow {
	return &serviceRow{
		ID:            s.ID,
		OrderID:       s.OrderID,
		BlueprintID:   s.BlueprintID,
		BlueprintName: s.BlueprintName,
		TenantID:      s.TenantID,
		Name:          s.Name,
		Status:        string(s.Status),
		Resources:     s.Resources,
		Created:       s.Created,
		Updated:       s.Updated,
	}
}

func rowToService(row *serviceRow) *Service {
	return &Service{
		ID:            row.ID,
		OrderID:       row.OrderID,
		BlueprintID:   row.BlueprintID,
		BlueprintName: row.BlueprintName,
		TenantID:      row.TenantID,
		Name:          row.Name,
		Status:        ServiceStatus(row.Status),
		Resources:     row.Resources,
		Created:       row.Created,
		Updated:       row.Updated,
	}
}
