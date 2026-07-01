-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE carts (
    customer_id VARCHAR(255) PRIMARY KEY,
    items JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
