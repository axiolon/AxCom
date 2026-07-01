// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

// CartItemResponse represents an item with enriched catalog details.
type CartItemResponse struct {
	VariantID       string            `json:"variant_id"`
	Quantity        int               `json:"quantity"`
	Name            string            `json:"name"`
	SKU             string            `json:"sku"`
	Price           float64           `json:"price"`
	DiscountedPrice float64           `json:"discounted_price"`
	ImageURL        string            `json:"image_url"`
	Stock           int               `json:"stock"`
	Attributes      map[string]string `json:"attributes"`
}

// CartResponse represents the customer's cart with dynamically resolved product details.
type CartResponse struct {
	CustomerID           string             `json:"customer_id"`
	Items                []CartItemResponse `json:"items"`
	TotalPrice           float64            `json:"total_price"`
	TotalDiscountedPrice float64            `json:"total_discounted_price"`
	UnavailableItems     []string           `json:"unavailable_items"`
}
