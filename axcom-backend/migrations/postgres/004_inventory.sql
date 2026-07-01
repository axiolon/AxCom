-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE stock_items (
    variant_id VARCHAR(255) NOT NULL,
    location_id VARCHAR(255) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    low_stock_threshold INT NOT NULL DEFAULT 0,
    allow_backorders BOOLEAN NOT NULL DEFAULT FALSE,
    backorder_limit INT NOT NULL DEFAULT 0,
    PRIMARY KEY (variant_id, location_id)
);

CREATE TABLE reservations (
    id VARCHAR(255) PRIMARY KEY,
    variant_id VARCHAR(255) NOT NULL,
    location_id VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE alerts (
    id VARCHAR(255) PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    variant_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE stock_history (
    id VARCHAR(255) PRIMARY KEY,
    variant_id VARCHAR(255) NOT NULL,
    location_id VARCHAR(255) NOT NULL,
    old_quantity INT NOT NULL,
    new_quantity INT NOT NULL,
    change_reason VARCHAR(255) NOT NULL,
    changed_by VARCHAR(255) NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Indexes for stock items queries
CREATE INDEX idx_stock_items_location ON stock_items(location_id);
CREATE INDEX idx_stock_items_low_stock ON stock_items(quantity) WHERE quantity <= low_stock_threshold;

-- Index for reservations lookup
CREATE INDEX idx_reservations_lookup ON reservations(variant_id, location_id);

-- Index for stock history queries
CREATE INDEX idx_stock_history_variant ON stock_history(variant_id);
