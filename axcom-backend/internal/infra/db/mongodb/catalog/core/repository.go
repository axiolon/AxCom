// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"
	"regexp"

	"ecom-engine/internal/core/catalog/domain"
	featureCore "ecom-engine/internal/core/catalog/features/core"
	"ecom-engine/internal/infra/db/mongodb/catalog/shared"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoCatalogRepository struct {
	productsCol   *mongo.Collection
	categoriesCol *mongo.Collection
}

// NewMongoCatalogRepository creates a new Mongo repository for core catalog.
func NewMongoCatalogRepository(db *mongo.Database) featureCore.Repository {
	return &MongoCatalogRepository{
		productsCol:   db.Collection("products"),
		categoriesCol: db.Collection("categories"),
	}
}

// CreateProduct persists a new product record.
func (r *MongoCatalogRepository) CreateProduct(ctx context.Context, p *domain.Product) error {
	doc := shared.ToProductDoc(p)
	_, err := r.productsCol.InsertOne(ctx, doc)
	return err
}

// GetProductByID retrieves a product by its unique identifier.
func (r *MongoCatalogRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	var doc shared.MongoProduct
	err := r.productsCol.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, featureCore.ErrProductNotFound
		}
		return nil, err
	}
	return shared.ToDomainProduct(&doc), nil
}

// GetProductByVariantID retrieves a product containing the given variant ID.
func (r *MongoCatalogRepository) GetProductByVariantID(ctx context.Context, variantID string) (*domain.Product, error) {
	var doc shared.MongoProduct
	err := r.productsCol.FindOne(ctx, bson.M{"variants.id": variantID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, featureCore.ErrProductNotFound
		}
		return nil, err
	}
	return shared.ToDomainProduct(&doc), nil
}

// UpdateVariantStock atomically updates the stock level of a single variant.
func (r *MongoCatalogRepository) UpdateVariantStock(ctx context.Context, variantID string, stock int) error {
	res, err := r.productsCol.UpdateOne(ctx,
		bson.M{"variants.id": variantID},
		bson.M{"$set": bson.M{"variants.$.stock": stock}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return featureCore.ErrProductNotFound
	}
	return nil
}

// ListProducts retrieves all products matching the supplied filter.
func (r *MongoCatalogRepository) ListProducts(ctx context.Context, filter *featureCore.ProductFilter) ([]domain.Product, error) {
	query := bson.M{}

	if filter != nil {
		if len(filter.CategoryIDs) > 0 {
			query["category_id"] = bson.M{"$in": filter.CategoryIDs}
		}

		priceFilter := bson.M{}
		if filter.MinPrice != nil {
			priceFilter["$gte"] = *filter.MinPrice
		}
		if filter.MaxPrice != nil {
			priceFilter["$lte"] = *filter.MaxPrice
		}
		if len(priceFilter) > 0 {
			query["variants.price"] = priceFilter
		}

		if filter.InStock != nil {
			if *filter.InStock {
				query["variants.stock"] = bson.M{"$gt": 0}
			} else {
				query["variants"] = bson.M{"$not": bson.M{"$elemMatch": bson.M{"stock": bson.M{"$gt": 0}}}}
			}
		}

		if len(filter.Attributes) > 0 {
			for k, v := range filter.Attributes {
				query["variants.attributes."+k] = bson.M{"$regex": "^" + regexp.QuoteMeta(v) + "$", "$options": "i"}
			}
		}

		if filter.Q != "" {
			qRegex := bson.M{"$regex": regexp.QuoteMeta(filter.Q), "$options": "i"}
			query["$or"] = []bson.M{
				{"name": qRegex},
				{"description": qRegex},
				{"variants.sku": qRegex},
				{"variants.name": qRegex},
			}
		}
	}

	opts := options.Find()
	if filter != nil {
		limit := filter.Limit
		if limit <= 0 || limit > 100 {
			limit = 100
		}
		opts.SetLimit(limit)

		if filter.Offset > 0 {
			opts.SetSkip(filter.Offset)
		}
	} else {
		opts.SetLimit(100)
	}

	cursor, err := r.productsCol.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []shared.MongoProduct
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	products := make([]domain.Product, len(docs))
	for i, doc := range docs {
		products[i] = *shared.ToDomainProduct(&doc)
	}
	return products, nil
}

// UpdateProduct updates an existing product with optimistic locking.
func (r *MongoCatalogRepository) UpdateProduct(ctx context.Context, p *domain.Product) error {
	originalVersion := p.Version
	p.Version = originalVersion + 1
	doc := shared.ToProductDoc(p)

	res, err := r.productsCol.ReplaceOne(ctx, bson.M{"_id": p.ID, "version": originalVersion}, doc)
	if err != nil {
		p.Version = originalVersion // rollback
		return err
	}
	if res.MatchedCount == 0 {
		p.Version = originalVersion // rollback
		// Distinguish between not found and conflict
		var temp shared.MongoProduct
		findErr := r.productsCol.FindOne(ctx, bson.M{"_id": p.ID}).Decode(&temp)
		if findErr != nil {
			if errors.Is(findErr, mongo.ErrNoDocuments) {
				return featureCore.ErrProductNotFound
			}
			return findErr
		}
		return featureCore.ErrVersionConflict
	}
	return nil
}

// DeleteProduct removes a product by its unique identifier.
func (r *MongoCatalogRepository) DeleteProduct(ctx context.Context, id string) error {
	res, err := r.productsCol.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return featureCore.ErrProductNotFound
	}
	return nil
}

// CreateCategory persists a new category record.
func (r *MongoCatalogRepository) CreateCategory(ctx context.Context, c *domain.Category) error {
	doc := shared.ToCategoryDoc(c)
	_, err := r.categoriesCol.InsertOne(ctx, doc)
	return err
}

// GetCategoryByID retrieves a category by its unique identifier.
func (r *MongoCatalogRepository) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	var doc shared.MongoCategory
	err := r.categoriesCol.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, featureCore.ErrCategoryNotFound
		}
		return nil, err
	}
	return shared.ToDomainCategory(&doc), nil
}

// ListCategories retrieves all categories.
func (r *MongoCatalogRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	cursor, err := r.categoriesCol.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []shared.MongoCategory
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	categories := make([]domain.Category, len(docs))
	for i, doc := range docs {
		categories[i] = *shared.ToDomainCategory(&doc)
	}
	return categories, nil
}

// UpdateCategory updates an existing category.
func (r *MongoCatalogRepository) UpdateCategory(ctx context.Context, c *domain.Category) error {
	doc := shared.ToCategoryDoc(c)
	res, err := r.categoriesCol.ReplaceOne(ctx, bson.M{"_id": c.ID}, doc)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return featureCore.ErrCategoryNotFound
	}
	return nil
}

// DeleteCategory removes a category by its unique identifier.
func (r *MongoCatalogRepository) DeleteCategory(ctx context.Context, id string) error {
	res, err := r.categoriesCol.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return featureCore.ErrCategoryNotFound
	}
	return nil
}
