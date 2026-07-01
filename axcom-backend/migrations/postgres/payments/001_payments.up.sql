-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0
-- order_id is an opaque identifier — no FK to orders module.

CREATE TABLE payments (
    id VARCHAR(255) PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    customer_id VARCHAR(255) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_intent_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    failure_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    refunded_at TIMESTAMP WITH TIME ZONE
);

-- One payment record per order.
CREATE UNIQUE INDEX uq_payments_order_id ON payments (order_id);
-- Provider intent ID must be unique per provider (prevents duplicate webhook processing).
CREATE UNIQUE INDEX uq_payments_provider_intent ON payments (provider, provider_intent_id);
-- Customer payment history queries.
CREATE INDEX idx_payments_customer_id_created_at ON payments (customer_id, created_at DESC);
-- Global listing by created_at.
CREATE INDEX idx_payments_created_at ON payments (created_at DESC);
