// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// expectedSchema maps each module key to the tables and required columns
// that must be present for a healthy Postgres schema.
var expectedSchema = map[string]map[string][]string{
	"core": {
		"users": {"id", "email", "password", "role", "created_at", "updated_at", "failed_login_attempts", "locked_until"},
	},
	"catalog": {
		"categories":     {"id", "name", "slug", "parent_id", "created_at", "updated_at"},
		"products":       {"id", "name", "description", "category_id", "version", "discount_type", "discount_value", "created_at", "updated_at"},
		"variants":       {"id", "product_id", "sku", "name", "price", "stock", "attributes"},
		"product_images": {"id", "product_id", "url", "key", "is_primary"},
		"reviews":        {"id", "product_id", "user_id", "rating", "comment", "reply_user_id", "reply_comment", "reply_created_at", "created_at"},
	},
	"orders": {
		"orders":      {"id", "customer_id", "customer_name", "customer_email", "customer_contact_number", "total", "status", "created_at"},
		"order_items": {"id", "order_id", "variant_id", "quantity", "price"},
	},
	"inventory": {
		"stock_items":   {"variant_id", "location_id", "quantity", "low_stock_threshold", "allow_backorders", "backorder_limit"},
		"reservations":  {"id", "variant_id", "location_id", "quantity", "expires_at"},
		"alerts":        {"id", "type", "message", "variant_id", "created_at", "is_read"},
		"stock_history": {"id", "variant_id", "location_id", "old_quantity", "new_quantity", "change_reason", "changed_by", "changed_at"},
	},
	"payments": {
		"payments": {"id", "order_id", "customer_id", "amount", "currency", "provider", "provider_intent_id", "status", "idempotency_key", "failure_reason", "created_at", "updated_at", "refunded_at"},
	},
	"shipping": {
		"shipments": {"id", "order_id", "carrier", "tracking_number", "status", "weight", "value", "shipping_cost", "estimated_delivery_at", "status_history", "created_at", "updated_at"},
	},
	"cart": {
		"carts": {"customer_id", "items", "created_at", "updated_at"},
	},
	"events": {
		"outbox":           {"id", "topic", "source", "payload", "version", "trace_id", "correlation_id", "created_at", "published_at"},
		"processed_events": {"event_id", "topic", "processed_at"},
	},
}

// verifyPostgres checks tables and columns for all enabled modules.
func verifyPostgres(ctx context.Context, pg *postgresMigrator, plan []ModuleEntry) error {
	fmt.Printf("\nDatabase Integrity Report (postgres)\n%s\n", strings.Repeat("=", 45))

	totalOK, totalFail, totalSkip := 0, 0, 0

	for _, entry := range plan {
		tables, ok := expectedSchema[entry.Key]
		if !ok {
			continue
		}
		if !entry.Enabled {
			for tbl := range tables {
				fmt.Printf("  [SKIP] %-25s module disabled\n", tbl)
				totalSkip++
			}
			continue
		}

		fmt.Printf("\n%s:\n", strings.ToUpper(entry.Key))
		for tbl, cols := range tables {
			missing, err := missingColumns(ctx, pg, tbl, cols)
			if err != nil {
				fmt.Printf("  [ERR]  %-25s error: %v\n", tbl, err)
				totalFail++
				continue
			}
			if len(missing) == 0 {
				fmt.Printf("  [OK]   %-25s %d/%d columns\n", tbl, len(cols), len(cols))
				totalOK++
			} else {
				fmt.Printf("  [FAIL] %-25s missing columns: %s\n", tbl, strings.Join(missing, ", "))
				totalFail++
			}
		}
	}

	fmt.Printf("\nResult: %d OK, %d failed, %d skipped\n\n", totalOK, totalFail, totalSkip)

	if totalFail > 0 {
		return fmt.Errorf("%d table(s) failed integrity check — run 'go run ./cmd/migrate up'", totalFail)
	}
	return nil
}

// missingColumns returns the column names from want that are absent in the table.
func missingColumns(ctx context.Context, pg *postgresMigrator, table string, want []string) ([]string, error) {
	rows, err := pg.db.QueryContext(ctx,
		`SELECT column_name FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = $1`, table,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	present := make(map[string]bool)
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		present[col] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Table doesn't exist at all.
	if len(present) == 0 {
		return []string{"TABLE MISSING"}, nil
	}

	var missing []string
	for _, col := range want {
		if !present[col] {
			missing = append(missing, col)
		}
	}
	return missing, nil
}

// verifyMongo checks that expected collections and indexes exist for enabled modules.
func verifyMongo(ctx context.Context, mg *mongoMigrator, plan []ModuleEntry) error {
	fmt.Printf("\nDatabase Integrity Report (mongodb)\n%s\n", strings.Repeat("=", 45))

	totalOK, totalFail, totalSkip := 0, 0, 0

	for _, entry := range plan {
		if !entry.Enabled {
			fmt.Printf("  [SKIP] module=%-12s (disabled)\n", entry.Key)
			totalSkip++
			continue
		}

		// Load the indexes.json to know what collections are expected.
		baseDir := filepath.Clean(filepath.Join("migrations", "mongodb"))
		specPath := filepath.Clean(filepath.Join(baseDir, entry.Key, "indexes.json"))
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			fmt.Printf("  [ERR]  module=%-12s cannot get abs base: %v\n", entry.Key, err)
			totalFail++
			continue
		}
		absSpec, err := filepath.Abs(specPath)
		if err != nil {
			fmt.Printf("  [ERR]  module=%-12s cannot get abs spec: %v\n", entry.Key, err)
			totalFail++
			continue
		}
		if !strings.HasPrefix(absSpec, absBase) {
			fmt.Printf("  [ERR]  module=%-12s path traversal detected: %s\n", entry.Key, specPath)
			totalFail++
			continue
		}
		// #nosec G304 -- specPath is checked to remain inside the migrations/mongodb directory
		data, err := os.ReadFile(specPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Printf("  [ERR]  module=%-12s cannot read spec: %v\n", entry.Key, err)
			totalFail++
			continue
		}

		var spec indexSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			fmt.Printf("  [ERR]  module=%-12s cannot parse spec: %v\n", entry.Key, err)
			totalFail++
			continue
		}

		fmt.Printf("\n%s:\n", strings.ToUpper(entry.Key))
		for _, colSpec := range spec.Collections {
			existing, err := mg.db.ListCollectionNames(ctx, bson.M{"name": colSpec.Name})
			if err != nil || len(existing) == 0 {
				fmt.Printf("  [FAIL] %-25s COLLECTION MISSING\n", colSpec.Name)
				totalFail++
				continue
			}

			// Check each expected index.
			col := mg.db.Collection(colSpec.Name)
			cursor, err := col.Indexes().List(ctx)
			if err != nil {
				fmt.Printf("  [ERR]  %-25s cannot list indexes: %v\n", colSpec.Name, err)
				totalFail++
				continue
			}

			presentIdx := make(map[string]bool)
			for cursor.Next(ctx) {
				var idx bson.M
				if cursor.Decode(&idx) == nil {
					if name, ok := idx["name"].(string); ok {
						presentIdx[name] = true
					}
				}
			}
			_ = cursor.Close(ctx)

			missingIdx := []string{}
			for _, idxSpec := range colSpec.Indexes {
				if !presentIdx[idxSpec.Options.Name] {
					missingIdx = append(missingIdx, idxSpec.Options.Name)
				}
			}

			if len(missingIdx) == 0 {
				fmt.Printf("  [OK]   %-25s %d index(es)\n", colSpec.Name, len(colSpec.Indexes))
				totalOK++
			} else {
				fmt.Printf("  [WARN] %-25s missing indexes: %s\n", colSpec.Name, strings.Join(missingIdx, ", "))
				totalFail++
			}
		}
	}

	fmt.Printf("\nResult: %d OK, %d failed, %d skipped\n\n", totalOK, totalFail, totalSkip)

	if totalFail > 0 {
		return fmt.Errorf("%d collection(s) failed integrity check — run 'go run ./cmd/migrate seed'", totalFail)
	}
	return nil
}
