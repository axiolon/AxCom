// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"database/sql"
	"encoding/json"
	"time"

	"ecom-engine/internal/core/catalog/domain"
)

type dbCategory struct {
	ID        string         `db:"id"`
	Name      string         `db:"name"`
	Slug      string         `db:"slug"`
	ParentID  sql.NullString `db:"parent_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func toDBCategory(c *domain.Category) *dbCategory {
	if c == nil {
		return nil
	}
	var parentID sql.NullString
	if c.ParentID != nil {
		parentID = sql.NullString{String: *c.ParentID, Valid: true}
	}
	return &dbCategory{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		ParentID:  parentID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toDomainCategory(dbC *dbCategory) *domain.Category {
	if dbC == nil {
		return nil
	}
	var parentID *string
	if dbC.ParentID.Valid {
		parentStr := dbC.ParentID.String
		parentID = &parentStr
	}
	return &domain.Category{
		ID:        dbC.ID,
		Name:      dbC.Name,
		Slug:      dbC.Slug,
		ParentID:  parentID,
		CreatedAt: dbC.CreatedAt,
		UpdatedAt: dbC.UpdatedAt,
	}
}

type dbProduct struct {
	ID            string          `db:"id"`
	Name          string          `db:"name"`
	Description   sql.NullString  `db:"description"`
	CategoryID    string          `db:"category_id"`
	Version       int             `db:"version"`
	DiscountType  sql.NullString  `db:"discount_type"`
	DiscountValue sql.NullFloat64 `db:"discount_value"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}

type dbVariant struct {
	ID         string  `db:"id"`
	ProductID  string  `db:"product_id"`
	SKU        string  `db:"sku"`
	Name       string  `db:"name"`
	Price      float64 `db:"price"`
	Stock      int     `db:"stock"`
	Attributes []byte  `db:"attributes"`
}

type dbProductImage struct {
	ID        string         `db:"id"`
	ProductID string         `db:"product_id"`
	URL       string         `db:"url"`
	Key       sql.NullString `db:"key"`
	IsPrimary bool           `db:"is_primary"`
}

func toDBProduct(p *domain.Product) *dbProduct {
	if p == nil {
		return nil
	}
	var desc sql.NullString
	if p.Description != "" {
		desc = sql.NullString{String: p.Description, Valid: true}
	}
	var discType sql.NullString
	var discValue sql.NullFloat64
	if p.Discount != nil {
		discType = sql.NullString{String: p.Discount.Type, Valid: true}
		discValue = sql.NullFloat64{Float64: p.Discount.Value, Valid: true}
	}
	return &dbProduct{
		ID:            p.ID,
		Name:          p.Name,
		Description:   desc,
		CategoryID:    p.CategoryID,
		Version:       p.Version,
		DiscountType:  discType,
		DiscountValue: discValue,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func toDomainProduct(dbP *dbProduct, dbVars []dbVariant, dbImgs []dbProductImage) (*domain.Product, error) {
	if dbP == nil {
		return nil, nil
	}

	variants := make([]domain.Variant, len(dbVars))
	for i, v := range dbVars {
		var attrs map[string]string
		if len(v.Attributes) > 0 {
			if err := json.Unmarshal(v.Attributes, &attrs); err != nil {
				return nil, err
			}
		}
		variants[i] = domain.Variant{
			ID:         v.ID,
			SKU:        v.SKU,
			Name:       v.Name,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: attrs,
		}
	}

	images := make([]domain.ProductImage, len(dbImgs))
	for i, img := range dbImgs {
		var keyStr string
		if img.Key.Valid {
			keyStr = img.Key.String
		}
		images[i] = domain.ProductImage{
			ID:        img.ID,
			URL:       img.URL,
			Key:       keyStr,
			IsPrimary: img.IsPrimary,
		}
	}

	var discount *domain.ProductDiscount
	if dbP.DiscountType.Valid {
		discount = &domain.ProductDiscount{
			Type:  dbP.DiscountType.String,
			Value: dbP.DiscountValue.Float64,
		}
	}

	var description string
	if dbP.Description.Valid {
		description = dbP.Description.String
	}

	return &domain.Product{
		ID:          dbP.ID,
		Name:        dbP.Name,
		Description: description,
		CategoryID:  dbP.CategoryID,
		Variants:    variants,
		Images:      images,
		Discount:    discount,
		Version:     dbP.Version,
		CreatedAt:   dbP.CreatedAt,
		UpdatedAt:   dbP.UpdatedAt,
	}, nil
}
