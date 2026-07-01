/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

db.createCollection("categories");
db.categories.createIndex({ slug: 1 }, { unique: true });

db.createCollection("products");
db.products.createIndex({ category_id: 1 });
