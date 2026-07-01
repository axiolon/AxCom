// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"context"
	"errors"

	"ecom-engine/internal/core/cart"
	"ecom-engine/pkg/logger"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel"
)

type MongoCartRepository struct {
	collection *mongo.Collection
}

func NewMongoCartRepository(db *mongo.Database) *MongoCartRepository {
	return &MongoCartRepository{
		collection: db.Collection("carts"),
	}
}

func (r *MongoCartRepository) GetByCustomerID(ctx context.Context, customerID string) (*cart.Cart, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoCartRepository.GetByCustomerID")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Fetching cart for customer ID: %s", customerID)

	var doc cartDoc
	err := r.collection.FindOne(ctx, bson.M{"_id": customerID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		logger.DebugCtx(ctx, "MongoDB: Cart not found for customer ID: %s", customerID)
		return nil, cart.ErrCartNotFound
	}
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to fetch cart: %v", err)
		return nil, err
	}

	return toDomainCart(&doc), nil
}

func (r *MongoCartRepository) Save(ctx context.Context, c *cart.Cart) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoCartRepository.Save")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Saving cart for customer ID: %s", c.CustomerID)

	doc := toCartDoc(c)
	opts := options.Replace().SetUpsert(true)
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": c.CustomerID}, doc, opts)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to save cart: %v", err)
		return err
	}

	logger.DebugCtx(ctx, "MongoDB: Successfully saved cart for customer ID: %s", c.CustomerID)
	return nil
}

func (r *MongoCartRepository) Delete(ctx context.Context, customerID string) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoCartRepository.Delete")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Deleting cart for customer ID: %s", customerID)

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": customerID})
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to delete cart: %v", err)
		return err
	}

	logger.DebugCtx(ctx, "MongoDB: Successfully deleted cart for customer ID: %s", customerID)
	return nil
}
