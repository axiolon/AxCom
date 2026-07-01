// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package outbox

import (
	"context"
	"errors"

	"ecom-engine/internal/events"
)

var errNotImplemented = errors.New("outbox is not yet implemented for MongoDB")

// MongoOutboxRepository is a stub — outbox pattern requires transactional writes
// which need careful design with MongoDB's multi-document transactions.
type MongoOutboxRepository struct{}

func NewMongoOutboxRepository() *MongoOutboxRepository {
	return &MongoOutboxRepository{}
}

func (r *MongoOutboxRepository) Store(_ context.Context, _ ...events.Event) error {
	return errNotImplemented
}

func (r *MongoOutboxRepository) FetchUnsent(_ context.Context, _ int) ([]events.OutboxRecord, error) {
	return nil, errNotImplemented
}

func (r *MongoOutboxRepository) MarkPublished(_ context.Context, _ []string) error {
	return errNotImplemented
}

// MongoDedupStore is a stub.
type MongoDedupStore struct{}

func NewMongoDedupStore() *MongoDedupStore {
	return &MongoDedupStore{}
}

func (s *MongoDedupStore) Exists(_ context.Context, _ string) (bool, error) {
	return false, errNotImplemented
}

func (s *MongoDedupStore) Mark(_ context.Context, _ string) error {
	return errNotImplemented
}

var _ events.OutboxRepository = (*MongoOutboxRepository)(nil)
var _ events.DedupStore = (*MongoDedupStore)(nil)
