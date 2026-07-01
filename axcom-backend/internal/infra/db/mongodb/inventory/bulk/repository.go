// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"
	"errors"

	"ecom-engine/internal/core/inventory/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	stockCollection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		stockCollection: db.Collection("stocks"),
	}
}

func (r *MongoRepository) GetStock(ctx context.Context, variantID string, locationID string) (*domain.StockItem, error) {
	var doc stockItemDoc
	err := r.stockCollection.FindOne(ctx, bson.M{"variant_id": variantID, "location_id": locationID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomainStockItem(&doc), nil
}

func (r *MongoRepository) SaveStock(ctx context.Context, stock *domain.StockItem) error {
	doc := toStockItemDoc(stock)
	opts := options.Replace().SetUpsert(true)
	_, err := r.stockCollection.ReplaceOne(
		ctx,
		bson.M{"variant_id": stock.VariantID, "location_id": stock.LocationID},
		doc,
		opts,
	)
	return err
}

func (r *MongoRepository) AdjustQuantity(ctx context.Context, variantID, locationID string, delta int) error {
	var doc stockItemDoc
	err := r.stockCollection.FindOne(ctx, bson.M{"variant_id": variantID, "location_id": locationID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ErrNotFound
		}
		return err
	}

	newQty := doc.Quantity + delta
	if doc.AllowBackorders {
		if newQty < -doc.BackorderLimit {
			return domain.ErrInsufficientStock
		}
	} else {
		if newQty < 0 {
			return domain.ErrInsufficientStock
		}
	}

	res, err := r.stockCollection.UpdateOne(
		ctx,
		bson.M{"variant_id": variantID, "location_id": locationID, "quantity": doc.Quantity},
		bson.M{"$set": bson.M{"quantity": newQty}},
	)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return errors.New("concurrent update conflict")
	}
	return nil
}
