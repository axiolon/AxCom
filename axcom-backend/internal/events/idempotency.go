// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
)

// DedupStore tracks which event IDs have already been processed,
// enabling at-least-once consumers to skip duplicates.
type DedupStore interface {
	Exists(ctx context.Context, eventID string) (bool, error)
	Mark(ctx context.Context, eventID string) error
}

// WithIdempotency wraps an EventHandler so that duplicate events (same ID) are skipped.
func WithIdempotency(store DedupStore, handler EventHandler) EventHandler {
	return func(event Event) error {
		ctx := context.Background()

		exists, err := store.Exists(ctx, event.ID)
		if err != nil {
			logger.Error("idempotency: failed to check event %s: %v", event.ID, err)
			return err // fail -> retry later
		}
		if exists {
			logger.Info("idempotency: skipping duplicate event %s (topic %s)", event.ID, event.Topic)
			metrics.EventsDedupSkippedTotal.WithLabelValues(event.Topic).Inc()
			return nil
		}

		if err := handler(event); err != nil {
			return err
		}

		if err := store.Mark(ctx, event.ID); err != nil {
			logger.Error("idempotency: failed to mark event %s as processed: %v", event.ID, err)
		}
		return nil
	}
}
