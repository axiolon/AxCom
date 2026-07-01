/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

db.createCollection("users");
db.users.createIndex({ email: 1 }, { unique: true });
