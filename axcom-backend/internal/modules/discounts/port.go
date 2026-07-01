// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

type OrderInfo struct {
	TotalAmount float64
	ItemCount   int
	CouponCode  string
}

type DiscountRule interface {
	Apply(order OrderInfo) float64
	GetName() string
}
