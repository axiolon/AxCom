// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoRepository struct {
	stockCollection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		stockCollection: db.Collection("stocks"),
	}
}

func (r *MongoRepository) GetLowStockItems(ctx context.Context) ([]*domain.StockItem, error) {
	query := bson.M{
		"$expr": bson.M{"$lte": []interface{}{"$quantity", "$low_stock_threshold"}},
	}
	cursor, err := r.stockCollection.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []*stockItemDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	items := make([]*domain.StockItem, len(docs))
	for i, doc := range docs {
		items[i] = toDomainStockItem(doc)
	}
	return items, nil
}

func (r *MongoRepository) GetAllStockItems(ctx context.Context) ([]*domain.StockItem, error) {
	cursor, err := r.stockCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []*stockItemDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	items := make([]*domain.StockItem, len(docs))
	for i, doc := range docs {
		items[i] = toDomainStockItem(doc)
	}
	return items, nil
}
