// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"
)

// OutboxRecord represents a persisted event waiting to be published by the relay.
type OutboxRecord struct {
	ID            string
	Topic         string
	Source        string
	Payload       []byte // JSON-encoded event payload
	Version       int
	TraceID       string
	CorrelationID string
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

// ToEvent reconstructs an Event from the persisted outbox record.
func (r OutboxRecord) ToEvent() Event {
	return Event{
		ID:            r.ID,
		Topic:         r.Topic,
		Source:        r.Source,
		Version:       r.Version,
		Timestamp:     r.CreatedAt,
		Payload:       r.Payload, // stays as []byte; consumer unmarshals
		TraceID:       r.TraceID,
		CorrelationID: r.CorrelationID,
	}
}

// OutboxRepository persists events transactionally alongside business data.
// Implementations must participate in the enclosing database transaction
// (e.g. by checking for a tx in context).
type OutboxRepository interface {
	// Store writes one or more events to the outbox table within the current transaction.
	Store(ctx context.Context, events ...Event) error

	// FetchUnsent returns up to batchSize records that have not yet been published.
	// Implementations should use SELECT ... FOR UPDATE SKIP LOCKED or equivalent
	// to allow concurrent relays without double-processing.
	FetchUnsent(ctx context.Context, batchSize int) ([]OutboxRecord, error)

	// MarkPublished sets published_at for the given record IDs.
	MarkPublished(ctx context.Context, ids []string) error
}
