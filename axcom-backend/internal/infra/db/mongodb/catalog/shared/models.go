// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"time"

	"ecom-engine/internal/core/catalog/domain"
)

type MongoProductImage struct {
	ID        string `bson:"id"`
	URL       string `bson:"url"`
	Key       string `bson:"key"`
	IsPrimary bool   `bson:"is_primary"`
}

type MongoVariant struct {
	ID         string            `bson:"id"`
	SKU        string            `bson:"sku"`
	Name       string            `bson:"name"`
	Price      float64           `bson:"price"`
	Stock      int               `bson:"stock"`
	Attributes map[string]string `bson:"attributes"`
}

type MongoProductDiscount struct {
	Type  string  `bson:"type"`
	Value float64 `bson:"value"`
}

type MongoProduct struct {
	ID          string                `bson:"_id"`
	Name        string                `bson:"name"`
	Description string                `bson:"description"`
	CategoryID  string                `bson:"category_id"`
	Variants    []MongoVariant        `bson:"variants"`
	Images      []MongoProductImage   `bson:"images,omitempty"`
	Discount    *MongoProductDiscount `bson:"discount,omitempty"`
	Version     int                   `bson:"version"`
	CreatedAt   time.Time             `bson:"created_at"`
	UpdatedAt   time.Time             `bson:"updated_at"`
}

type MongoCategory struct {
	ID        string    `bson:"_id"`
	Name      string    `bson:"name"`
	Slug      string    `bson:"slug"`
	ParentID  *string   `bson:"parent_id,omitempty"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func ToProductDoc(p *domain.Product) *MongoProduct {
	if p == nil {
		return nil
	}

	variants := make([]MongoVariant, len(p.Variants))
	for i, v := range p.Variants {
		variants[i] = MongoVariant{
			ID:         v.ID,
			SKU:        v.SKU,
			Name:       v.Name,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: v.Attributes,
		}
	}

	images := make([]MongoProductImage, len(p.Images))
	for i, img := range p.Images {
		images[i] = MongoProductImage{
			ID:        img.ID,
			URL:       img.URL,
			Key:       img.Key,
			IsPrimary: img.IsPrimary,
		}
	}

	var discount *MongoProductDiscount
	if p.Discount != nil {
		discount = &MongoProductDiscount{
			Type:  p.Discount.Type,
			Value: p.Discount.Value,
		}
	}

	return &MongoProduct{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CategoryID:  p.CategoryID,
		Variants:    variants,
		Images:      images,
		Discount:    discount,
		Version:     p.Version,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func ToDomainProduct(doc *MongoProduct) *domain.Product {
	if doc == nil {
		return nil
	}

	variants := make([]domain.Variant, len(doc.Variants))
	for i, v := range doc.Variants {
		variants[i] = domain.Variant{
			ID:         v.ID,
			SKU:        v.SKU,
			Name:       v.Name,
			Price:      v.Price,
			Stock:      v.Stock,
			Attributes: v.Attributes,
		}
	}

	images := make([]domain.ProductImage, len(doc.Images))
	for i, img := range doc.Images {
		images[i] = domain.ProductImage{
			ID:        img.ID,
			URL:       img.URL,
			Key:       img.Key,
			IsPrimary: img.IsPrimary,
		}
	}

	var discount *domain.ProductDiscount
	if doc.Discount != nil {
		discount = &domain.ProductDiscount{
			Type:  doc.Discount.Type,
			Value: doc.Discount.Value,
		}
	}

	return &domain.Product{
		ID:          doc.ID,
		Name:        doc.Name,
		Description: doc.Description,
		CategoryID:  doc.CategoryID,
		Variants:    variants,
		Images:      images,
		Discount:    discount,
		Version:     doc.Version,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

func ToCategoryDoc(c *domain.Category) *MongoCategory {
	if c == nil {
		return nil
	}
	return &MongoCategory{
		ID:        c.ID,
		Name:      c.Name,
		Slug:      c.Slug,
		ParentID:  c.ParentID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func ToDomainCategory(doc *MongoCategory) *domain.Category {
	if doc == nil {
		return nil
	}
	return &domain.Category{
		ID:        doc.ID,
		Name:      doc.Name,
		Slug:      doc.Slug,
		ParentID:  doc.ParentID,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}
