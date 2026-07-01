// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ecom-engine/pkg/logger"

	_ "github.com/lib/pq" // postgres driver registration
)

type postgresMigrator struct {
	db *sql.DB
}

func newPostgresMigrator(ctx context.Context, connStr string) (*postgresMigrator, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &postgresMigrator{db: db}, nil
}

func (p *postgresMigrator) close() { _ = p.db.Close() }

// ensureTrackingTable creates the schema_migrations table if it does not exist.
func (p *postgresMigrator) ensureTrackingTable(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			module      TEXT        NOT NULL,
			version     INT         NOT NULL,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (module, version)
		)
	`)
	return err
}

// migrateUp scans dir for *.up.sql files and applies any whose version is not
// yet recorded in schema_migrations for the given module.
func (p *postgresMigrator) migrateUp(ctx context.Context, module, dir string) error {
	files, err := upFiles(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("[migrate] no directory for module=%s, skipping", module)
			return nil
		}
		return err
	}

	applied, err := p.appliedVersions(ctx, module)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("abs dir: %w", err)
	}

	pending := 0
	for _, f := range files {
		ver := versionFromFilename(f)
		if applied[ver] {
			continue
		}
		pending++
		cleanF := filepath.Clean(f)
		absF, err := filepath.Abs(cleanF)
		if err != nil {
			return fmt.Errorf("abs file: %w", err)
		}
		if !strings.HasPrefix(absF, absDir) {
			return fmt.Errorf("path traversal detected: %s", f)
		}
		// #nosec G304 -- cleanF is verified to be within migration dir prefix
		sql, err := os.ReadFile(cleanF)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}

		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec %s: %w", filepath.Base(f), err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (module, version) VALUES ($1, $2)`,
			module, ver,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit: %w", err)
		}
		logger.Info("[migrate] applied module=%-12s version=%d  file=%s", module, ver, filepath.Base(f))
	}

	if pending == 0 {
		logger.Info("[migrate] ok     module=%-12s (already up to date)", module)
	}
	return nil
}

// migrateDown rolls back the highest applied version for the given module.
func (p *postgresMigrator) migrateDown(ctx context.Context, module, dir string) error {
	applied, err := p.appliedVersions(ctx, module)
	if err != nil {
		return err
	}
	if len(applied) == 0 {
		return fmt.Errorf("module %q has no applied migrations", module)
	}

	latest := 0
	for ver := range applied {
		if ver > latest {
			latest = ver
		}
	}

	pattern := filepath.Join(dir, fmt.Sprintf("%03d_*.down.sql", latest))
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("no down migration found for module=%s version=%d (pattern: %s)", module, latest, pattern)
	}

	cleanF := filepath.Clean(matches[0])
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("abs dir: %w", err)
	}
	absF, err := filepath.Abs(cleanF)
	if err != nil {
		return fmt.Errorf("abs file: %w", err)
	}
	if !strings.HasPrefix(absF, absDir) {
		return fmt.Errorf("path traversal detected: %s", cleanF)
	}

	// #nosec G304 -- cleanF is verified to be within migration dir prefix
	sql, err := os.ReadFile(cleanF)
	if err != nil {
		return err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec down migration: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM schema_migrations WHERE module = $1 AND version = $2`,
		module, latest,
	); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	logger.Info("[migrate] rolled back module=%-12s version=%d", module, latest)
	return nil
}

// status prints the current applied version vs available files for each module.
func (p *postgresMigrator) status(ctx context.Context, plan []ModuleEntry, migrationsRoot string) error {
	fmt.Printf("\n%-15s  %-8s  %-8s  %s\n", "MODULE", "APPLIED", "LATEST", "STATUS")
	fmt.Println(strings.Repeat("-", 55))

	for _, entry := range plan {
		dir := filepath.Join(migrationsRoot, entry.Key)
		files, _ := upFiles(dir)
		latest := 0
		if len(files) > 0 {
			latest = versionFromFilename(files[len(files)-1])
		}

		applied, _ := p.appliedVersions(ctx, entry.Key)
		currentVer := 0
		for ver := range applied {
			if ver > currentVer {
				currentVer = ver
			}
		}

		status := "ok"
		if !entry.Enabled {
			status = "disabled"
		} else if currentVer < latest {
			status = fmt.Sprintf("PENDING (%d unapplied)", latest-currentVer)
		}

		fmt.Printf("%-15s  %-8d  %-8d  %s\n", entry.Key, currentVer, latest, status)
	}
	fmt.Println()
	return nil
}

// quickCheck verifies schema_migrations exists and core has at least version 1.
func (p *postgresMigrator) quickCheck(ctx context.Context) error {
	var count int
	err := p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'schema_migrations'`,
	).Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("schema_migrations table not found — run 'go run ./cmd/migrate up'")
	}

	err = p.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE module = 'core' AND version >= 1`,
	).Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("core schema not applied — run 'go run ./cmd/migrate up'")
	}
	return nil
}

// appliedVersions returns the set of already-applied versions for a module.
func (p *postgresMigrator) appliedVersions(ctx context.Context, module string) (map[int]bool, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT version FROM schema_migrations WHERE module = $1`, module,
	)
	if err != nil {
		// Table may not exist yet before ensureTrackingTable runs.
		return map[int]bool{}, nil
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		result[v] = true
	}
	return result, rows.Err()
}

// upFiles returns sorted *.up.sql files in dir.
func upFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// versionFromFilename parses the leading numeric prefix from a filename like
// "001_users.up.sql" → 1.
func versionFromFilename(path string) int {
	base := filepath.Base(path)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimLeft(parts[0], "0"))
	return n
}
