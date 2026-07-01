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

	"ecom-engine/pkg/logger"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoMigrator struct {
	client *mongo.Client
	db     *mongo.Database
}

func newMongoMigrator(ctx context.Context, connStr, dbName string) (*mongoMigrator, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping: %w", err)
	}
	if dbName == "" {
		dbName = "ecom_db"
	}
	return &mongoMigrator{client: client, db: client.Database(dbName)}, nil
}

func (m *mongoMigrator) close() {
	_ = m.client.Disconnect(context.Background())
}

// indexSpec mirrors the JSON structure in indexes.json.
type indexSpec struct {
	Collections []collectionSpec `json:"collections"`
}

type collectionSpec struct {
	Name      string           `json:"name"`
	Indexes   []mongoIndexSpec `json:"indexes"`
	Validator *bson.M          `json:"validator,omitempty"`
}

type mongoIndexSpec struct {
	Keys    map[string]interface{} `json:"keys"`
	Options struct {
		Unique             bool   `json:"unique"`
		Sparse             bool   `json:"sparse"`
		Name               string `json:"name"`
		ExpireAfterSeconds *int32 `json:"expireAfterSeconds,omitempty"`
	} `json:"options"`
}

// seed reads indexes.json in dir and ensures every collection/index exists.
func (m *mongoMigrator) seed(ctx context.Context, module, dir string) error {
	specPath := filepath.Clean(filepath.Join(dir, "indexes.json"))
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("abs dir: %w", err)
	}
	absSpec, err := filepath.Abs(specPath)
	if err != nil {
		return fmt.Errorf("abs spec: %w", err)
	}
	if !strings.HasPrefix(absSpec, absDir) {
		return fmt.Errorf("path traversal detected: %s", specPath)
	}
	// #nosec G304 -- specPath is checked to remain inside the migrations directory
	data, err := os.ReadFile(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("[migrate] no indexes.json for module=%s, skipping", module)
			return nil
		}
		return fmt.Errorf("read %s: %w", specPath, err)
	}

	var spec indexSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("parse %s: %w", specPath, err)
	}

	for _, colSpec := range spec.Collections {
		if err := m.ensureCollection(ctx, colSpec); err != nil {
			return fmt.Errorf("collection %q: %w", colSpec.Name, err)
		}
		logger.Info("[migrate] seeded module=%-12s collection=%s", module, colSpec.Name)
	}
	return nil
}

// ensureCollection creates the collection if absent, applies a validator if
// specified, then creates any missing indexes.
func (m *mongoMigrator) ensureCollection(ctx context.Context, spec collectionSpec) error {
	existing, err := m.db.ListCollectionNames(ctx, bson.M{"name": spec.Name})
	if err != nil {
		return err
	}

	if len(existing) == 0 {
		createOpts := options.CreateCollection()
		if spec.Validator != nil {
			createOpts.SetValidator(*spec.Validator)
		}
		if err := m.db.CreateCollection(ctx, spec.Name, createOpts); err != nil {
			return fmt.Errorf("create collection: %w", err)
		}
	}

	col := m.db.Collection(spec.Name)
	for _, idxSpec := range spec.Indexes {
		if err := m.ensureIndex(ctx, col, idxSpec); err != nil {
			return fmt.Errorf("index %q: %w", idxSpec.Options.Name, err)
		}
	}
	return nil
}

// ensureIndex creates an index if one with the same name does not already exist.
func (m *mongoMigrator) ensureIndex(ctx context.Context, col *mongo.Collection, spec mongoIndexSpec) error {
	// Check if index already exists by name.
	cursor, err := col.Indexes().List(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close(ctx) }()

	for cursor.Next(ctx) {
		var idx bson.M
		if err := cursor.Decode(&idx); err != nil {
			continue
		}
		if name, ok := idx["name"].(string); ok && name == spec.Options.Name {
			return nil // already exists
		}
	}

	// Build the index keys document.
	keysDoc := bson.D{}
	for k, v := range spec.Keys {
		switch val := v.(type) {
		case float64:
			keysDoc = append(keysDoc, bson.E{Key: k, Value: int(val)})
		default:
			keysDoc = append(keysDoc, bson.E{Key: k, Value: v})
		}
	}

	idxOpts := options.Index().SetName(spec.Options.Name)
	if spec.Options.Unique {
		idxOpts.SetUnique(true)
	}
	if spec.Options.Sparse {
		idxOpts.SetSparse(true)
	}
	if spec.Options.ExpireAfterSeconds != nil {
		idxOpts.SetExpireAfterSeconds(*spec.Options.ExpireAfterSeconds)
	}

	model := mongo.IndexModel{Keys: keysDoc, Options: idxOpts}
	if _, err := col.Indexes().CreateOne(ctx, model); err != nil {
		return err
	}
	return nil
}
