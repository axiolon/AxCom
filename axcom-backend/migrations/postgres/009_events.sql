-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0

-- Outbox table: events are written here transactionally alongside business data.
-- The relay polls for unpublished rows and publishes them to the event bus.
CREATE TABLE IF NOT EXISTS outbox (
    id              VARCHAR(255) PRIMARY KEY,
    topic           VARCHAR(100) NOT NULL,
    source          VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    trace_id        VARCHAR(64),
    correlation_id  VARCHAR(64),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at    TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;

-- Processed events table: consumer-side deduplication for at-least-once delivery.
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     VARCHAR(255) PRIMARY KEY,
    topic        VARCHAR(100) NOT NULL DEFAULT '',
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_processed_events_cleanup ON processed_events(processed_at);
