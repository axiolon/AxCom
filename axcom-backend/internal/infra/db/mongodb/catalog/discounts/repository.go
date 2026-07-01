// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

import (
	"context"
	"errors"

	"ecom-engine/internal/core/catalog/domain"
	featureDiscounts "ecom-engine/internal/core/catalog/features/discounts"
	"ecom-engine/internal/infra/db/mongodb/catalog/shared"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoDiscountRepository struct {
	productsCol *mongo.Collection
}

// NewMongoRepository creates a new Mongo repository for discounts.
func NewMongoRepository(db *mongo.Database) featureDiscounts.Repository {
	return &MongoDiscountRepository{
		productsCol: db.Collection("products"),
	}
}

func (r *MongoDiscountRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	var doc shared.MongoProduct
	err := r.productsCol.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return shared.ToDomainProduct(&doc), nil
}

func (r *MongoDiscountRepository) UpdateProductDiscount(ctx context.Context, id string, discount *domain.ProductDiscount) error {
	var mongoDiscount *shared.MongoProductDiscount
	if discount != nil {
		mongoDiscount = &shared.MongoProductDiscount{
			Type:  discount.Type,
			Value: discount.Value,
		}
	}

	res, err := r.productsCol.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"discount": mongoDiscount}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}
