// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoConnection wraps the v2 mongo client.
type MongoConnection struct {
	Client *mongo.Client
}

func (c *MongoConnection) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx, nil)
}

func (c *MongoConnection) Close() error {
	return c.Client.Disconnect(ctxNoCancel())
}

// Helper to get non-cancelled context for cleanup.
func ctxNoCancel() context.Context {
	return context.Background()
}
