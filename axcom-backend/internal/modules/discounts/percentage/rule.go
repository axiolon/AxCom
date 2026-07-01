// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package percentage

import "ecom-engine/internal/modules/discounts"

type PercentageRule struct { //nolint:revive // Name is intentionally explicit for the public API.
	percentage float64 // e.g. 0.10 for 10%
}

func NewPercentageRule(percentage float64) *PercentageRule {
	return &PercentageRule{percentage: percentage}
}

func (r *PercentageRule) Apply(order discounts.OrderInfo) float64 {
	return order.TotalAmount * r.percentage
}

func (r *PercentageRule) GetName() string {
	return "Percentage Discount"
}
