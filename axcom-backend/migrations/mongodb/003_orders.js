/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

db.createCollection("orders");
db.orders.createIndex({ customer_id: 1 });
