// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import (
	"context"
	"errors"

	"ecom-engine/internal/core/catalog/domain"
	featureVariants "ecom-engine/internal/core/catalog/features/variants"
	"ecom-engine/internal/infra/db/mongodb/catalog/shared"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoVariantRepository struct {
	productsCol *mongo.Collection
}

// NewMongoRepository creates a new Mongo repository for variants.
func NewMongoRepository(db *mongo.Database) featureVariants.Repository {
	return &MongoVariantRepository{
		productsCol: db.Collection("products"),
	}
}

func (r *MongoVariantRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
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

func (r *MongoVariantRepository) UpdateProductVariants(ctx context.Context, id string, variants []domain.Variant) error {
	mongoVariants := make([]shared.MongoVariant, len(variants))
	for i, v := range variants {
		mongoVariants[i] = shared.MongoVariant{
			ID:         v.ID,
			SKU:        v.SKU,
			Name:       v.Name,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: v.Attributes,
		}
	}

	res, err := r.productsCol.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"variants": mongoVariants}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}
