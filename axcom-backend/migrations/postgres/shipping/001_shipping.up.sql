-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0
-- order_id is an opaque identifier — no FK to orders module.

CREATE TABLE shipments (
    id VARCHAR(255) PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    carrier VARCHAR(100) NOT NULL,
    tracking_number VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    weight DECIMAL(10, 2) NOT NULL,
    value DECIMAL(12, 2) NOT NULL,
    shipping_cost DECIMAL(12, 2) NOT NULL,
    estimated_delivery_at TIMESTAMP WITH TIME ZONE,
    status_history TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
