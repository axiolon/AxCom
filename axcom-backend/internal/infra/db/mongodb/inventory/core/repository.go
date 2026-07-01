// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"

	"ecom-engine/internal/core/inventory/domain"
	featcore "ecom-engine/internal/core/inventory/features/core"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	stockCollection *mongo.Collection
	alertCollection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		stockCollection: db.Collection("stocks"),
		alertCollection: db.Collection("alerts"),
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

func (r *MongoRepository) DeleteStock(ctx context.Context, variantID string, locationID string) error {
	res, err := r.stockCollection.DeleteOne(ctx, bson.M{"variant_id": variantID, "location_id": locationID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MongoRepository) ListStock(ctx context.Context, filter featcore.ListStockFilter) ([]*domain.StockItem, error) {
	query := bson.M{}
	if filter.VariantID != "" {
		query["variant_id"] = filter.VariantID
	}
	if filter.LocationID != "" {
		query["location_id"] = filter.LocationID
	}
	if filter.Status == "LOW_STOCK" {
		query["$expr"] = bson.M{"$lte": []interface{}{"$quantity", "$low_stock_threshold"}}
	}

	opts := options.Find()
	if filter.Limit > 0 {
		opts.SetLimit(filter.Limit)
	}
	if filter.Offset > 0 {
		opts.SetSkip(filter.Offset)
	}

	cursor, err := r.stockCollection.Find(ctx, query, opts)
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

func (r *MongoRepository) SaveAlert(ctx context.Context, alert *domain.Alert) error {
	doc := toAlertDoc(alert)
	opts := options.Replace().SetUpsert(true)
	_, err := r.alertCollection.ReplaceOne(ctx, bson.M{"_id": alert.ID}, doc, opts)
	return err
}

func (r *MongoRepository) ListAlerts(ctx context.Context, limit, offset int) ([]*domain.Alert, error) {
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}
	cursor, err := r.alertCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []*alertDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	alerts := make([]*domain.Alert, len(docs))
	for i, doc := range docs {
		alerts[i] = toDomainAlert(doc)
	}
	return alerts, nil
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
