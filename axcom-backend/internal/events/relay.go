// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"

	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
)

// OutboxRelay polls the outbox table for unpublished events and publishes them
// to the event bus. It runs as a background goroutine.
type OutboxRelay struct {
	outbox   OutboxRepository
	bus      EventBus
	interval time.Duration
	batch    int
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewOutboxRelay creates a relay that polls at the given interval.
func NewOutboxRelay(outbox OutboxRepository, bus EventBus, interval time.Duration, batchSize int) *OutboxRelay {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &OutboxRelay{
		outbox:   outbox,
		bus:      bus,
		interval: interval,
		batch:    batchSize,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the background polling loop.
func (r *OutboxRelay) Start() {
	go r.run()
	logger.Info("Outbox relay started: interval=%s, batch=%d", r.interval, r.batch)
}

// Stop signals the relay to stop and waits for it to finish.
func (r *OutboxRelay) Stop() {
	close(r.stopCh)
	<-r.doneCh
	logger.Info("Outbox relay stopped")
}

func (r *OutboxRelay) run() {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.poll()
		case <-r.stopCh:
			return
		}
	}
}

func (r *OutboxRelay) poll() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	records, err := r.outbox.FetchUnsent(ctx, r.batch)
	if err != nil {
		logger.Error("outbox relay: fetch unsent: %v", err)
		metrics.EventsOutboxRelayErrorsTotal.Inc()
		return
	}
	if len(records) == 0 {
		return
	}

	metrics.EventsOutboxPendingBatch.Set(float64(len(records)))

	ids := make([]string, 0, len(records))
	for _, rec := range records {
		event := rec.ToEvent()
		r.bus.Publish(event)
		ids = append(ids, rec.ID)
	}

	if err := r.outbox.MarkPublished(ctx, ids); err != nil {
		logger.Error("outbox relay: mark published: %v", err)
		metrics.EventsOutboxRelayErrorsTotal.Inc()
		return
	}

	metrics.EventsOutboxRelayPublishedTotal.Add(float64(len(ids)))
	logger.Info("outbox relay: published %d event(s)", len(ids))
}
