// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"encoding/json"
	"fmt"

	"ecom-engine/internal/core/catalog/domain"
	featureCore "ecom-engine/internal/core/catalog/features/core"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresCatalogRepository struct {
	db db.Database
}

func NewPostgresCatalogRepository(database db.Database) featureCore.Repository {
	return &PostgresCatalogRepository{
		db: database,
	}
}

func (r *PostgresCatalogRepository) CreateProduct(ctx context.Context, p *domain.Product) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.CreateProduct")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Creating product ID: %s", p.ID)

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	dbP := toDBProduct(p)
	query := `INSERT INTO products (id, name, description, category_id, version, discount_type, discount_value, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	err = tx.Exec(ctx, query, dbP.ID, dbP.Name, dbP.Description, dbP.CategoryID, dbP.Version, dbP.DiscountType, dbP.DiscountValue, dbP.CreatedAt, dbP.UpdatedAt)
	if err != nil {
		span.RecordError(err)
		return err
	}

	for _, v := range p.Variants {
		attrsBytes, _ := json.Marshal(v.Attributes)
		queryVar := `INSERT INTO variants (id, product_id, sku, name, price, stock, attributes) 
                     VALUES ($1, $2, $3, $4, $5, $6, $7)`
		err = tx.Exec(ctx, queryVar, v.ID, p.ID, v.SKU, v.Name, v.Price, v.Stock, string(attrsBytes))
		if err != nil {
			span.RecordError(err)
			return err
		}
	}

	for _, img := range p.Images {
		queryImg := `INSERT INTO product_images (id, product_id, url, key, is_primary) 
                     VALUES ($1, $2, $3, $4, $5)`
		err = tx.Exec(ctx, queryImg, img.ID, p.ID, img.URL, img.Key, img.IsPrimary)
		if err != nil {
			span.RecordError(err)
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresCatalogRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.GetProductByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding product by ID: %s", id)

	query := `SELECT id, name, description, category_id, version, discount_type, discount_value, created_at, updated_at 
              FROM products WHERE id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, featureCore.ErrProductNotFound
	}

	var dbP dbProduct
	if err = rows.Scan(&dbP.ID, &dbP.Name, &dbP.Description, &dbP.CategoryID, &dbP.Version, &dbP.DiscountType, &dbP.DiscountValue, &dbP.CreatedAt, &dbP.UpdatedAt); err != nil {
		span.RecordError(err)
		return nil, err
	}

	dbVars, err := r.fetchVariants(ctx, id)
	if err != nil {
		return nil, err
	}

	dbImgs, err := r.fetchImages(ctx, id)
	if err != nil {
		return nil, err
	}

	return toDomainProduct(&dbP, dbVars, dbImgs)
}

func (r *PostgresCatalogRepository) GetProductByVariantID(ctx context.Context, variantID string) (*domain.Product, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.GetProductByVariantID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding product by Variant ID: %s", variantID)

	query := `SELECT p.id, p.name, p.description, p.category_id, p.version, p.discount_type, p.discount_value, p.created_at, p.updated_at 
              FROM products p 
              JOIN variants v ON p.id = v.product_id 
              WHERE v.id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, variantID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, featureCore.ErrProductNotFound
	}

	var dbP dbProduct
	if err = rows.Scan(&dbP.ID, &dbP.Name, &dbP.Description, &dbP.CategoryID, &dbP.Version, &dbP.DiscountType, &dbP.DiscountValue, &dbP.CreatedAt, &dbP.UpdatedAt); err != nil {
		span.RecordError(err)
		return nil, err
	}

	dbVars, err := r.fetchVariants(ctx, dbP.ID)
	if err != nil {
		return nil, err
	}

	dbImgs, err := r.fetchImages(ctx, dbP.ID)
	if err != nil {
		return nil, err
	}

	return toDomainProduct(&dbP, dbVars, dbImgs)
}

func (r *PostgresCatalogRepository) UpdateVariantStock(ctx context.Context, variantID string, stock int) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.UpdateVariantStock")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating variant stock level: %s to %d", variantID, stock)

	query := "UPDATE variants SET stock = $1 WHERE id = $2"
	err := r.db.Exec(ctx, query, stock, variantID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (r *PostgresCatalogRepository) ListProducts(ctx context.Context, filter *featureCore.ProductFilter) ([]domain.Product, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.ListProducts")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing products with filters: %v", filter)

	query := `SELECT DISTINCT p.id, p.name, p.description, p.category_id, p.version, p.discount_type, p.discount_value, p.created_at, p.updated_at 
              FROM products p 
              LEFT JOIN variants v ON p.id = v.product_id`
	var conditions []string
	var args []interface{}
	argCount := 1

	if filter != nil {
		if len(filter.CategoryIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("p.category_id = ANY($%d)", argCount))
			args = append(args, filter.CategoryIDs)
			argCount++
		}
		if filter.MinPrice != nil {
			conditions = append(conditions, fmt.Sprintf("v.price >= $%d", argCount))
			args = append(args, *filter.MinPrice)
			argCount++
		}
		if filter.MaxPrice != nil {
			conditions = append(conditions, fmt.Sprintf("v.price <= $%d", argCount))
			args = append(args, *filter.MaxPrice)
			argCount++
		}
		if filter.InStock != nil {
			if *filter.InStock {
				conditions = append(conditions, "v.stock > 0")
			} else {
				conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM variants v2 WHERE v2.product_id = p.id AND v2.stock > 0)")
			}
		}
		if filter.Q != "" {
			qPattern := "%" + filter.Q + "%"
			conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR p.description ILIKE $%d OR v.sku ILIKE $%d OR v.name ILIKE $%d)", argCount, argCount, argCount, argCount))
			args = append(args, qPattern)
			argCount++
		}
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	limit := int64(100)
	offset := int64(0)
	if filter != nil {
		if filter.Limit > 0 && filter.Limit <= 100 {
			limit = filter.Limit
		}
		if filter.Offset > 0 {
			offset = filter.Offset
		}
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var dbProducts []dbProduct
	var productIDs []string
	for rows.Next() {
		var dbP dbProduct
		if err = rows.Scan(&dbP.ID, &dbP.Name, &dbP.Description, &dbP.CategoryID, &dbP.Version, &dbP.DiscountType, &dbP.DiscountValue, &dbP.CreatedAt, &dbP.UpdatedAt); err != nil {
			span.RecordError(err)
			return nil, err
		}
		dbProducts = append(dbProducts, dbP)
		productIDs = append(productIDs, dbP.ID)
	}
	if err = rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	if len(dbProducts) == 0 {
		return []domain.Product{}, nil
	}

	// Fetch all variants for fetched products to prevent N+1 queries
	var dbVars []dbVariant
	var varArgs []interface{}
	varArgs = append(varArgs, productIDs)
	varQuery := "SELECT id, product_id, sku, name, price, stock, attributes FROM variants WHERE product_id = ANY($1)"
	varRows, err := r.db.Query(ctx, varQuery, varArgs...)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = varRows.Close() }()

	for varRows.Next() {
		var v dbVariant
		if err = varRows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Name, &v.Price, &v.Stock, &v.Attributes); err != nil {
			span.RecordError(err)
			return nil, err
		}
		dbVars = append(dbVars, v)
	}
	if err = varRows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Fetch all images for fetched products to prevent N+1 queries
	var dbImgs []dbProductImage
	var imgArgs []interface{}
	imgArgs = append(imgArgs, productIDs)
	imgQuery := "SELECT id, product_id, url, key, is_primary FROM product_images WHERE product_id = ANY($1)"
	imgRows, err := r.db.Query(ctx, imgQuery, imgArgs...)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = imgRows.Close() }()

	for imgRows.Next() {
		var img dbProductImage
		if err := imgRows.Scan(&img.ID, &img.ProductID, &img.URL, &img.Key, &img.IsPrimary); err != nil {
			span.RecordError(err)
			return nil, err
		}
		dbImgs = append(dbImgs, img)
	}
	if err := imgRows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Group variants and images by product ID
	variantsMap := make(map[string][]dbVariant)
	for _, v := range dbVars {
		variantsMap[v.ProductID] = append(variantsMap[v.ProductID], v)
	}

	imagesMap := make(map[string][]dbProductImage)
	for _, img := range dbImgs {
		imagesMap[img.ProductID] = append(imagesMap[img.ProductID], img)
	}

	products := make([]domain.Product, len(dbProducts))
	for i, dbP := range dbProducts {
		domainP, err := toDomainProduct(&dbP, variantsMap[dbP.ID], imagesMap[dbP.ID])
		if err != nil {
			return nil, err
		}
		products[i] = *domainP
	}

	return products, nil
}

func (r *PostgresCatalogRepository) UpdateProduct(ctx context.Context, p *domain.Product) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.UpdateProduct")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating product ID: %s", p.ID)

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		span.RecordError(err)
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	originalVersion := p.Version
	p.Version = originalVersion + 1
	dbP := toDBProduct(p)

	// Optimistic locking
	query := `UPDATE products 
              SET name = $1, description = $2, category_id = $3, version = $4, discount_type = $5, discount_value = $6, updated_at = $7 
              WHERE id = $8 AND version = $9`
	res, err := tx.ExecResult(ctx, query, dbP.Name, dbP.Description, dbP.CategoryID, dbP.Version, dbP.DiscountType, dbP.DiscountValue, dbP.UpdatedAt, dbP.ID, originalVersion)
	if err != nil {
		p.Version = originalVersion // rollback
		span.RecordError(err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		p.Version = originalVersion // rollback
		span.RecordError(err)
		return err
	}

	if rowsAffected == 0 {
		p.Version = originalVersion // rollback
		// Distinguish between not found and conflict
		checkQuery := "SELECT 1 FROM products WHERE id = $1"
		checkRows, checkErr := tx.Query(ctx, checkQuery, p.ID)
		if checkErr != nil {
			span.RecordError(checkErr)
			return checkErr
		}
		defer func() { _ = checkRows.Close() }()
		if !checkRows.Next() {
			return featureCore.ErrProductNotFound
		}
		return featureCore.ErrVersionConflict
	}

	// Update variants (delete and re-insert)
	delVarsQuery := "DELETE FROM variants WHERE product_id = $1"
	err = tx.Exec(ctx, delVarsQuery, p.ID)
	if err != nil {
		p.Version = originalVersion
		span.RecordError(err)
		return err
	}

	for _, v := range p.Variants {
		attrsBytes, _ := json.Marshal(v.Attributes)
		queryVar := `INSERT INTO variants (id, product_id, sku, name, price, stock, attributes) 
                     VALUES ($1, $2, $3, $4, $5, $6, $7)`
		err = tx.Exec(ctx, queryVar, v.ID, p.ID, v.SKU, v.Name, v.Price, v.Stock, string(attrsBytes))
		if err != nil {
			p.Version = originalVersion
			span.RecordError(err)
			return err
		}
	}

	// Update images (delete and re-insert)
	delImgsQuery := "DELETE FROM product_images WHERE product_id = $1"
	err = tx.Exec(ctx, delImgsQuery, p.ID)
	if err != nil {
		p.Version = originalVersion
		span.RecordError(err)
		return err
	}

	for _, img := range p.Images {
		queryImg := `INSERT INTO product_images (id, product_id, url, key, is_primary) 
                     VALUES ($1, $2, $3, $4, $5)`
		err = tx.Exec(ctx, queryImg, img.ID, p.ID, img.URL, img.Key, img.IsPrimary)
		if err != nil {
			p.Version = originalVersion
			span.RecordError(err)
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresCatalogRepository) DeleteProduct(ctx context.Context, id string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.DeleteProduct")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Deleting product ID: %s", id)

	query := "DELETE FROM products WHERE id = $1"
	res, err := r.db.ExecResult(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		return featureCore.ErrProductNotFound
	}
	return nil
}

func (r *PostgresCatalogRepository) CreateCategory(ctx context.Context, c *domain.Category) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.CreateCategory")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Creating category ID: %s", c.ID)

	dbC := toDBCategory(c)
	query := "INSERT INTO categories (id, name, slug, parent_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
	err := r.db.Exec(ctx, query, dbC.ID, dbC.Name, dbC.Slug, dbC.ParentID, dbC.CreatedAt, dbC.UpdatedAt)
	if err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (r *PostgresCatalogRepository) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.GetCategoryByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding category by ID: %s", id)

	query := "SELECT id, name, slug, parent_id, created_at, updated_at FROM categories WHERE id = $1 LIMIT 1"
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, featureCore.ErrCategoryNotFound
	}

	var dbC dbCategory
	if err := rows.Scan(&dbC.ID, &dbC.Name, &dbC.Slug, &dbC.ParentID, &dbC.CreatedAt, &dbC.UpdatedAt); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return toDomainCategory(&dbC), nil
}

func (r *PostgresCatalogRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.ListCategories")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Querying all categories")

	query := "SELECT id, name, slug, parent_id, created_at, updated_at FROM categories"
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var categories []domain.Category
	for rows.Next() {
		var dbC dbCategory
		if err := rows.Scan(&dbC.ID, &dbC.Name, &dbC.Slug, &dbC.ParentID, &dbC.CreatedAt, &dbC.UpdatedAt); err != nil {
			span.RecordError(err)
			return nil, err
		}
		categories = append(categories, *toDomainCategory(&dbC))
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}
	return categories, nil
}

func (r *PostgresCatalogRepository) UpdateCategory(ctx context.Context, c *domain.Category) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.UpdateCategory")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating category ID: %s", c.ID)

	dbC := toDBCategory(c)
	query := "UPDATE categories SET name = $1, slug = $2, parent_id = $3, updated_at = $4 WHERE id = $5"
	res, err := r.db.ExecResult(ctx, query, dbC.Name, dbC.Slug, dbC.ParentID, dbC.UpdatedAt, dbC.ID)
	if err != nil {
		span.RecordError(err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		return featureCore.ErrCategoryNotFound
	}
	return nil
}

func (r *PostgresCatalogRepository) DeleteCategory(ctx context.Context, id string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresCatalogRepository.DeleteCategory")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Deleting category ID: %s", id)

	query := "DELETE FROM categories WHERE id = $1"
	res, err := r.db.ExecResult(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		return featureCore.ErrCategoryNotFound
	}
	return nil
}

func (r *PostgresCatalogRepository) fetchVariants(ctx context.Context, productID string) ([]dbVariant, error) {
	query := "SELECT id, product_id, sku, name, price, stock, attributes FROM variants WHERE product_id = $1"
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var dbVars []dbVariant
	for rows.Next() {
		var v dbVariant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Name, &v.Price, &v.Stock, &v.Attributes); err != nil {
			return nil, err
		}
		dbVars = append(dbVars, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dbVars, nil
}

func (r *PostgresCatalogRepository) fetchImages(ctx context.Context, productID string) ([]dbProductImage, error) {
	query := "SELECT id, product_id, url, key, is_primary FROM product_images WHERE product_id = $1"
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var dbImgs []dbProductImage
	for rows.Next() {
		var img dbProductImage
		if err := rows.Scan(&img.ID, &img.ProductID, &img.URL, &img.Key, &img.IsPrimary); err != nil {
			return nil, err
		}
		dbImgs = append(dbImgs, img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dbImgs, nil
}
