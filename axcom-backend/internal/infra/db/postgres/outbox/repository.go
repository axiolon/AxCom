// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ecom-engine/internal/events"
	postgres "ecom-engine/internal/infra/db/postgres"
)

// PostgresOutboxRepository implements events.OutboxRepository using the shared
// PostgresAdapter. All writes automatically participate in an enclosing RunInTx
// transaction via the adapter's context-based tx propagation.
type PostgresOutboxRepository struct {
	db *postgres.PostgresAdapter
}

func NewPostgresOutboxRepository(db *postgres.PostgresAdapter) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{db: db}
}

func (r *PostgresOutboxRepository) Store(ctx context.Context, evts ...events.Event) error {
	for _, evt := range evts {
		payload, err := json.Marshal(evt.Payload)
		if err != nil {
			return fmt.Errorf("outbox: marshal payload for event %s: %w", evt.ID, err)
		}

		err = r.db.Exec(ctx,
			`INSERT INTO outbox (id, topic, source, payload, version, trace_id, correlation_id, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			evt.ID, evt.Topic, evt.Source, payload, evt.Version,
			evt.TraceID, evt.CorrelationID, evt.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("outbox: store event %s: %w", evt.ID, err)
		}
	}
	return nil
}

func (r *PostgresOutboxRepository) FetchUnsent(ctx context.Context, batchSize int) ([]events.OutboxRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, topic, source, payload, version, trace_id, correlation_id, created_at
		 FROM outbox
		 WHERE published_at IS NULL
		 ORDER BY created_at
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		batchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("outbox: fetch unsent: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []events.OutboxRecord
	for rows.Next() {
		var rec events.OutboxRecord
		var traceID, corrID *string
		if err := rows.Scan(&rec.ID, &rec.Topic, &rec.Source, &rec.Payload,
			&rec.Version, &traceID, &corrID, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("outbox: scan row: %w", err)
		}
		if traceID != nil {
			rec.TraceID = *traceID
		}
		if corrID != nil {
			rec.CorrelationID = *corrID
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outbox: iterate rows: %w", err)
	}
	return records, nil
}

func (r *PostgresOutboxRepository) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	// Build $1, $2, ... placeholders.
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = time.Now()
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}
	query := fmt.Sprintf(
		`UPDATE outbox SET published_at = $1 WHERE id IN (%s)`,
		strings.Join(placeholders, ", "),
	)
	if err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("outbox: mark published: %w", err)
	}
	return nil
}

// PostgresDedupStore implements events.DedupStore using the processed_events table.
type PostgresDedupStore struct {
	db *postgres.PostgresAdapter
}

func NewPostgresDedupStore(db *postgres.PostgresAdapter) *PostgresDedupStore {
	return &PostgresDedupStore{db: db}
}

func (s *PostgresDedupStore) Exists(ctx context.Context, eventID string) (bool, error) {
	rows, err := s.db.Query(ctx, `SELECT 1 FROM processed_events WHERE event_id = $1`, eventID)
	if err != nil {
		return false, fmt.Errorf("dedup: exists check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Next(), nil
}

func (s *PostgresDedupStore) Mark(ctx context.Context, eventID string) error {
	err := s.db.Exec(ctx,
		`INSERT INTO processed_events (event_id, topic, processed_at)
		 VALUES ($1, '', $2)
		 ON CONFLICT (event_id) DO NOTHING`,
		eventID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("dedup: mark event %s: %w", eventID, err)
	}
	return nil
}

var _ events.OutboxRepository = (*PostgresOutboxRepository)(nil)
var _ events.DedupStore = (*PostgresDedupStore)(nil)
