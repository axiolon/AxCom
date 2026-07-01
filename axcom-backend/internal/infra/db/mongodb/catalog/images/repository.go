// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	"context"
	"errors"

	"ecom-engine/internal/core/catalog/domain"
	featureImages "ecom-engine/internal/core/catalog/features/images"
	"ecom-engine/internal/infra/db/mongodb/catalog/shared"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoImageRepository struct {
	productsCol *mongo.Collection
}

// NewMongoRepository creates a new Mongo repository for images.
func NewMongoRepository(db *mongo.Database) featureImages.Repository {
	return &MongoImageRepository{
		productsCol: db.Collection("products"),
	}
}

func (r *MongoImageRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
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

func (r *MongoImageRepository) UpdateProductImages(ctx context.Context, id string, images []domain.ProductImage) error {
	mongoImages := make([]shared.MongoProductImage, len(images))
	for i, img := range images {
		mongoImages[i] = shared.MongoProductImage{
			ID:        img.ID,
			URL:       img.URL,
			Key:       img.Key,
			IsPrimary: img.IsPrimary,
		}
	}

	res, err := r.productsCol.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"images": mongoImages}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}
