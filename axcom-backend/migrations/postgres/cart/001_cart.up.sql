-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0
-- Items stored as JSONB. customer_id is an opaque identifier — no FK to users.

CREATE TABLE carts (
    customer_id VARCHAR(255) PRIMARY KEY,
    items JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
