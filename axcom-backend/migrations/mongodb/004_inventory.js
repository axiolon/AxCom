/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

db.createCollection("stock_items");
db.stock_items.createIndex({ variant_id: 1 }, { unique: true });

db.createCollection("reservations");
db.reservations.createIndex({ expires_at: 1 }, { expireAfterSeconds: 0 });
