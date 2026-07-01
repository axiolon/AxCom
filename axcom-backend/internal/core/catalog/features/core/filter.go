// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"strings"

	"ecom-engine/internal/core/catalog/domain"
)

// FilterProducts filters the list of products based on query criteria.
// It matches categories hierarchically, filters variants by price/attributes, and supports full-text q searching.
func FilterProducts(products []domain.Product, categories []domain.Category, query *ListProductsQuery) []domain.Product {
	if query == nil {
		return products
	}

	// 1. Filter by category (hierarchical check)
	var allowedCategories map[string]bool
	catID := query.CategoryID
	if catID == "" {
		catID = query.Category
	}
	if catID != "" {
		allowedCategories = getCategoryDescendants(catID, categories)
	}

	// 2. Resolve pricing filter limits
	minPrice := query.PriceMin
	if minPrice == nil {
		minPrice = query.MinPrice
	}
	maxPrice := query.PriceMax
	if maxPrice == nil {
		maxPrice = query.MaxPrice
	}

	// 3. Parse attributes filter
	queryAttrs := parseAttributes(query.Attributes)

	var filtered []domain.Product
	for _, p := range products {
		// Category check
		if allowedCategories != nil && !allowedCategories[p.CategoryID] {
			continue
		}

		// Text search check
		if query.Q != "" {
			qLower := strings.ToLower(query.Q)
			matchText := strings.Contains(strings.ToLower(p.Name), qLower) ||
				strings.Contains(strings.ToLower(p.Description), qLower)

			if !matchText {
				// check SKU/Name matches on variants
				for _, v := range p.Variants {
					if strings.Contains(strings.ToLower(v.SKU), qLower) ||
						strings.Contains(strings.ToLower(v.Name), qLower) {
						matchText = true
						break
					}
				}
			}

			if !matchText {
				continue // skip product as it doesn't match Q query string
			}
		}

		// Variant criteria check (Price & Attributes)
		hasMatchingVariant := false
		for _, v := range p.Variants {
			if matchVariant(v, minPrice, maxPrice, queryAttrs) {
				hasMatchingVariant = true
				break
			}
		}

		// If no variants match when filters are active, skip this product
		hasFiltersActive := minPrice != nil || maxPrice != nil || len(queryAttrs) > 0
		if hasFiltersActive && !hasMatchingVariant {
			continue
		}

		filtered = append(filtered, p)
	}

	return filtered
}

// getCategoryDescendants finds a category and all of its subcategories recursively.
func getCategoryDescendants(targetID string, allCategories []domain.Category) map[string]bool {
	descendants := map[string]bool{targetID: true}
	adj := make(map[string][]string)
	for _, cat := range allCategories {
		if cat.ParentID != nil && *cat.ParentID != "" {
			adj[*cat.ParentID] = append(adj[*cat.ParentID], cat.ID)
		}
	}

	var visit func(id string)
	visit = func(id string) {
		for _, childID := range adj[id] {
			if !descendants[childID] {
				descendants[childID] = true
				visit(childID)
			}
		}
	}
	visit(targetID)
	return descendants
}

// parseAttributes converts a string like "size:XL,color:blue" to a map.
func parseAttributes(attrStr string) map[string]string {
	result := make(map[string]string)
	if strings.TrimSpace(attrStr) == "" {
		return result
	}
	parts := strings.Split(attrStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			if k != "" && v != "" {
				result[strings.ToLower(k)] = strings.ToLower(v)
			}
		}
	}
	return result
}

// matchVariant checks if a variant matches price range and attribute filters.
func matchVariant(v domain.Variant, minPrice, maxPrice *float64, queryAttrs map[string]string) bool {
	if minPrice != nil && v.Price < *minPrice {
		return false
	}
	if maxPrice != nil && v.Price > *maxPrice {
		return false
	}

	if len(queryAttrs) == 0 {
		return true
	}
	if len(v.Attributes) == 0 {
		return false
	}

	// Check if all queryAttrs are present in variant attributes (case-insensitive keys and values)
	for qk, qv := range queryAttrs {
		matched := false
		for vk, vv := range v.Attributes {
			if strings.ToLower(vk) == qk && strings.ToLower(vv) == qv {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}
