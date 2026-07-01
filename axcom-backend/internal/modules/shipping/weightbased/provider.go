// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package weightbased

import "ecom-engine/internal/modules/shipping"

type WeightBasedProvider struct { //nolint:revive // Name is intentionally explicit for the public API.
	ratePerKg float64
	baseRate  float64
}

func NewWeightBasedProvider(baseRate, ratePerKg float64) *WeightBasedProvider {
	return &WeightBasedProvider{
		baseRate:  baseRate,
		ratePerKg: ratePerKg,
	}
}

func (p *WeightBasedProvider) CalculateRate(pkg shipping.Package) (float64, error) {
	return p.baseRate + (pkg.Weight * p.ratePerKg), nil
}

func (p *WeightBasedProvider) GetName() string {
	return "Weight-Based Shipping"
}
