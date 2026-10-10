// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package switchconfig

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ConfigRecord is one versioned rendered configuration snapshot.
type ConfigRecord struct {
	bun.BaseModel `bun:"table:switch_configs"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	SwitchID    string    `bun:"switch_id,notnull"`
	SiteID      string    `bun:"site_id,notnull"`
	Format      string    `bun:"format,notnull"` // "cli" | "nvue-json"
	ContentHash string    `bun:"content_hash,notnull"`
	Content     []byte    `bun:"content,notnull"` // gzip-compressed
	Author      string    `bun:"author,notnull"`
	CommitMsg   string    `bun:"commit_msg,notnull"`
	CreatedAt   time.Time `bun:"created_at,nullzero,default:now()"`
}

// ValidationResult records one cable-validation outcome per port.
type ValidationResult struct {
	bun.BaseModel `bun:"table:switch_validation_results"`

	ID           uuid.UUID `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	SiteID       string    `bun:"site_id,notnull"`
	SwitchID     string    `bun:"switch_id,notnull"`
	Port         string    `bun:"port,notnull"`
	ExpectedPeer string    `bun:"expected_peer,notnull"`
	ObservedPeer string    `bun:"observed_peer,notnull"`
	Match        bool      `bun:"match,notnull"`
	ValidatedAt  time.Time `bun:"validated_at,nullzero,default:now()"`
}

// ConfigStore persists rendered switch configs with content-hash deduplication
// and gzip compression.
type ConfigStore struct {
	db *bun.DB
}

// NewConfigStore creates a ConfigStore backed by db.
func NewConfigStore(db *bun.DB) *ConfigStore {
	return &ConfigStore{db: db}
}

// Store renders, compresses, and persists a config. Returns the existing record
// if an identical hash already exists (deduplication).
func (s *ConfigStore) Store(ctx context.Context, switchID, siteID, format, author, msg string, rendered []byte) (*ConfigRecord, error) {
	hash := contentHash(rendered)

	// Dedup check
	existing := &ConfigRecord{}
	err := s.db.NewSelect().Model(existing).
		Where("switch_id = ? AND format = ? AND content_hash = ?", switchID, format, hash).
		Limit(1).Scan(ctx)
	if err == nil {
		return existing, nil
	}

	compressed, err := gzipCompress(rendered)
	if err != nil {
		return nil, fmt.Errorf("compress config: %w", err)
	}

	rec := &ConfigRecord{
		SwitchID:    switchID,
		SiteID:      siteID,
		Format:      format,
		ContentHash: hash,
		Content:     compressed,
		Author:      author,
		CommitMsg:   msg,
	}
	if _, err := s.db.NewInsert().Model(rec).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert config record: %w", err)
	}
	return rec, nil
}

// Latest returns the most recently stored config for the given switch and format.
func (s *ConfigStore) Latest(ctx context.Context, switchID, format string) (*ConfigRecord, error) {
	rec := &ConfigRecord{}
	err := s.db.NewSelect().Model(rec).
		Where("switch_id = ? AND format = ?", switchID, format).
		OrderExpr("created_at DESC").
		Limit(1).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// Decompress returns the uncompressed content of a ConfigRecord.
func Decompress(rec *ConfigRecord) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(rec.Content))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// StoreValidationResults persists cable-validation results for a site.
func (s *ConfigStore) StoreValidationResults(ctx context.Context, results []*ValidationResult) error {
	if len(results) == 0 {
		return nil
	}
	_, err := s.db.NewInsert().Model(&results).Exec(ctx)
	return err
}

// LatestValidationResults returns the most recent validation results for a site.
func (s *ConfigStore) LatestValidationResults(ctx context.Context, siteID string) ([]*ValidationResult, error) {
	var results []*ValidationResult
	err := s.db.NewSelect().Model(&results).
		Where("site_id = ?", siteID).
		OrderExpr("validated_at DESC").
		Limit(500).Scan(ctx)
	return results, err
}

func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
