// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package coupon

import "ecom-engine/internal/modules/discounts"

type CouponRule struct { //nolint:revive // Name is intentionally explicit for the public API.
	code   string
	amount float64
}

func NewCouponRule(code string, amount float64) *CouponRule {
	return &CouponRule{code: code, amount: amount}
}

func (r *CouponRule) Apply(order discounts.OrderInfo) float64 {
	if order.CouponCode == r.code {
		if order.TotalAmount < r.amount {
			return order.TotalAmount
		}
		return r.amount
	}
	return 0.0
}

func (r *CouponRule) GetName() string {
	return "Fixed Coupon Discount"
}
