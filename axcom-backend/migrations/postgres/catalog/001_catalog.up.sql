-- Copyright 2026 Axiolon Labs
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE categories (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    parent_id VARCHAR(255) REFERENCES categories(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category_id VARCHAR(255) REFERENCES categories(id) ON DELETE SET NULL,
    version INT NOT NULL DEFAULT 1,
    discount_type VARCHAR(50),
    discount_value DECIMAL(12, 2),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE variants (
    id VARCHAR(255) PRIMARY KEY,
    product_id VARCHAR(255) REFERENCES products(id) ON DELETE CASCADE,
    sku VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(12, 2) NOT NULL,
    stock INT NOT NULL DEFAULT 0,
    attributes JSONB
);

CREATE TABLE product_images (
    id VARCHAR(255) PRIMARY KEY,
    product_id VARCHAR(255) REFERENCES products(id) ON DELETE CASCADE,
    url VARCHAR(255) NOT NULL,
    key VARCHAR(255),
    is_primary BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_variants_product_id ON variants(product_id);
CREATE INDEX idx_product_images_product_id ON product_images(product_id);