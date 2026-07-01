// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reservation

import (
	"context"
	"errors"

	"ecom-engine/internal/core/inventory/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	stockCollection       *mongo.Collection
	reservationCollection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{
		stockCollection:       db.Collection("stocks"),
		reservationCollection: db.Collection("reservations"),
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

func (r *MongoRepository) CreateReservation(ctx context.Context, res *domain.Reservation) error {
	doc := toReservationDoc(res)
	_, err := r.reservationCollection.InsertOne(ctx, doc)
	return err
}

func (r *MongoRepository) GetReservation(ctx context.Context, resID string) (*domain.Reservation, error) {
	var doc reservationDoc
	err := r.reservationCollection.FindOne(ctx, bson.M{"_id": resID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrReservationNotFound
		}
		return nil, err
	}
	return toDomainReservation(&doc), nil
}

func (r *MongoRepository) DeleteReservation(ctx context.Context, resID string) error {
	res, err := r.reservationCollection.DeleteOne(ctx, bson.M{"_id": resID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrReservationNotFound
	}
	return nil
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
