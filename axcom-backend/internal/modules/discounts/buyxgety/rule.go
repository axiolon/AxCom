// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package buyxgety

import "ecom-engine/internal/modules/discounts"

type BuyXGetYRule struct { //nolint:revive // Name is intentionally explicit for the public API.
	x         int
	y         int
	itemPrice float64
}

func NewBuyXGetYRule(x, y int, itemPrice float64) *BuyXGetYRule {
	return &BuyXGetYRule{x: x, y: y, itemPrice: itemPrice}
}

func (r *BuyXGetYRule) Apply(order discounts.OrderInfo) float64 {
	// Simple simulation: for every X items purchased, get Y items free.
	// Assume the free items are of 'itemPrice'
	if order.ItemCount >= r.x+r.y {
		sets := order.ItemCount / (r.x + r.y)
		return float64(sets*r.y) * r.itemPrice
	}
	return 0.0
}

func (r *BuyXGetYRule) GetName() string {
	return "Buy X Get Y Free Discount"
}
