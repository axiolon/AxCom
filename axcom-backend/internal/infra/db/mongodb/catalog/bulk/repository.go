// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"
	"errors"

	"ecom-engine/internal/core/catalog/domain"
	featureBulk "ecom-engine/internal/core/catalog/features/bulk"
	"ecom-engine/internal/infra/db/mongodb/catalog/shared"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoBulkRepository struct {
	productsCol   *mongo.Collection
	categoriesCol *mongo.Collection
}

// NewMongoRepository creates a new Mongo repository for bulk operations.
func NewMongoRepository(db *mongo.Database) featureBulk.Repository {
	return &MongoBulkRepository{
		productsCol:   db.Collection("products"),
		categoriesCol: db.Collection("categories"),
	}
}

func (r *MongoBulkRepository) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	var doc shared.MongoCategory
	err := r.categoriesCol.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, err
	}
	return &domain.Category{
		ID:       doc.ID,
		Name:     doc.Name,
		Slug:     doc.Slug,
		ParentID: doc.ParentID,
	}, nil
}

func (r *MongoBulkRepository) BulkCreate(ctx context.Context, products []*domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	var writeModels []mongo.WriteModel
	for _, p := range products {
		doc := shared.ToProductDoc(p)
		writeModels = append(writeModels, mongo.NewInsertOneModel().SetDocument(doc))
	}

	_, err := r.productsCol.BulkWrite(ctx, writeModels)
	return err
}

func (r *MongoBulkRepository) BulkUpdate(ctx context.Context, products []*domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	var writeModels []mongo.WriteModel
	for _, p := range products {
		doc := shared.ToProductDoc(p)
		writeModels = append(writeModels, mongo.NewReplaceOneModel().SetFilter(bson.M{"_id": p.ID}).SetReplacement(doc))
	}

	_, err := r.productsCol.BulkWrite(ctx, writeModels)
	return err
}

func (r *MongoBulkRepository) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	_, err := r.productsCol.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": ids}})
	return err
}
