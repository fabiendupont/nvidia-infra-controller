// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"encoding/json"
	"fmt"

	"context"
	"github.com/google/uuid"

	sdk "github.com/NVIDIA/infra-controller/provider-sdk/db"
)

// BlueprintSQLStore is a PostgreSQL-backed blueprint store.
type BlueprintSQLStore struct {
	dao *blueprintDAO
}

// NewBlueprintSQLStore creates a new SQL-backed blueprint store using a
// provider-sdk Session already scoped to the catalog schema.
func NewBlueprintSQLStore(s *sdk.Session) *BlueprintSQLStore {
	return &BlueprintSQLStore{dao: newBlueprintDAO(s)}
}

func (s *BlueprintSQLStore) Create(b *Blueprint) error {
	dbm, err := blueprintToDBModel(b)
	if err != nil {
		return err
	}
	created, err := s.dao.create(context.Background(), dbm)
	if err != nil {
		return err
	}
	b.ID = created.ID.String()
	b.Created = created.Created
	b.Updated = created.Updated
	b.IsActive = created.IsActive
	return nil
}

func (s *BlueprintSQLStore) GetByID(id string) (*Blueprint, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("blueprint %s not found", id)
	}
	dbm, err := s.dao.getByID(context.Background(), uid)
	if err != nil {
		return nil, fmt.Errorf("blueprint %s not found", id)
	}
	return dbModelToBlueprint(dbm)
}

func (s *BlueprintSQLStore) GetByNameVersion(name, version string) (*Blueprint, error) {
	dbm, err := s.dao.getByNameVersion(context.Background(), name, version)
	if err != nil {
		if version != "" {
			return nil, fmt.Errorf("blueprint %s@%s not found", name, version)
		}
		return nil, fmt.Errorf("blueprint %s not found", name)
	}
	return dbModelToBlueprint(dbm)
}

func (s *BlueprintSQLStore) GetAll() []*Blueprint {
	isActive := true
	dbms, err := s.dao.getAll(context.Background(), &isActive)
	if err != nil {
		return nil
	}
	var result []*Blueprint
	for i := range dbms {
		bp, err := dbModelToBlueprint(&dbms[i])
		if err != nil {
			continue
		}
		result = append(result, bp)
	}
	return result
}

func (s *BlueprintSQLStore) Update(b *Blueprint) error {
	dbm, err := blueprintToDBModel(b)
	if err != nil {
		return err
	}
	updated, err := s.dao.update(context.Background(), dbm)
	if err != nil {
		return err
	}
	b.Updated = updated.Updated
	return nil
}

func (s *BlueprintSQLStore) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("blueprint %s not found", id)
	}
	return s.dao.deleteByID(context.Background(), uid)
}

// blueprintToDBModel converts a provider Blueprint to a db model.
func blueprintToDBModel(b *Blueprint) (*dbBlueprint, error) {
	uid := uuid.Nil
	if b.ID != "" {
		var err error
		uid, err = uuid.Parse(b.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid blueprint ID: %s", b.ID)
		}
	}

	params, err := toJSONMap(b.Parameters)
	if err != nil {
		return nil, fmt.Errorf("serialize parameters: %w", err)
	}
	resources, err := toJSONMap(b.Resources)
	if err != nil {
		return nil, fmt.Errorf("serialize resources: %w", err)
	}
	pricing, err := toJSONMap(b.Pricing)
	if err != nil {
		return nil, fmt.Errorf("serialize pricing: %w", err)
	}

	visibility := b.Visibility
	if visibility == "" {
		visibility = VisibilityPublic
	}

	var basedOn *string
	if b.BasedOn != "" {
		basedOn = &b.BasedOn
	}

	var tenantID *uuid.UUID
	if b.TenantID != nil {
		tenantID = b.TenantID
	}

	return &dbBlueprint{
		ID:          uid,
		Name:        b.Name,
		Version:     b.Version,
		Description: b.Description,
		Parameters:  params,
		Resources:   resources,
		Labels:      b.Labels,
		Pricing:     pricing,
		TenantID:    tenantID,
		Visibility:  visibility,
		BasedOn:     basedOn,
		IsActive:    b.IsActive,
		Created:     b.Created,
		Updated:     b.Updated,
	}, nil
}

// dbModelToBlueprint converts a db model to a provider Blueprint.
func dbModelToBlueprint(m *dbBlueprint) (*Blueprint, error) {
	params, err := fromJSONMap[BlueprintParameter](m.Parameters)
	if err != nil {
		return nil, fmt.Errorf("deserialize parameters: %w", err)
	}
	resources, err := fromJSONMap[BlueprintResource](m.Resources)
	if err != nil {
		return nil, fmt.Errorf("deserialize resources: %w", err)
	}

	var pricing *PricingSpec
	if m.Pricing != nil {
		p, err := fromJSONValue[PricingSpec](m.Pricing)
		if err != nil {
			return nil, fmt.Errorf("deserialize pricing: %w", err)
		}
		pricing = p
	}

	var basedOn string
	if m.BasedOn != nil {
		basedOn = *m.BasedOn
	}

	return &Blueprint{
		ID:          m.ID.String(),
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Parameters:  params,
		Resources:   resources,
		Labels:      m.Labels,
		Pricing:     pricing,
		TenantID:    m.TenantID,
		Visibility:  m.Visibility,
		BasedOn:     basedOn,
		IsActive:    m.IsActive,
		Created:     m.Created,
		Updated:     m.Updated,
	}, nil
}

func fromJSONValue[T any](m map[string]interface{}) (*T, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func toJSONMap(v interface{}) (map[string]interface{}, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func fromJSONMap[T any](m map[string]interface{}) (map[string]T, error) {
	if m == nil {
		return nil, nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var result map[string]T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
