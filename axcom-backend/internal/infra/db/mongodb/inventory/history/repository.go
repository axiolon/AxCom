// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	historyCollection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		historyCollection: db.Collection("stock_history"),
	}
}

func (r *MongoRepository) CreateHistory(ctx context.Context, h *domain.StockHistory) error {
	doc := toHistoryDoc(h)
	_, err := r.historyCollection.InsertOne(ctx, doc)
	return err
}

func (r *MongoRepository) GetHistory(ctx context.Context, variantID string, limit, offset int) ([]*domain.StockHistory, error) {
	opts := options.Find().SetSort(bson.M{"changed_at": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	cursor, err := r.historyCollection.Find(ctx, bson.M{"variant_id": variantID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []*stockHistoryDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	res := make([]*domain.StockHistory, len(docs))
	for i, doc := range docs {
		res[i] = toDomainHistory(doc)
	}
	return res, nil
}
