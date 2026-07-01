// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package flatrate

import "ecom-engine/internal/modules/shipping"

type FlatRateProvider struct { //nolint:revive // Name is intentionally explicit for the public API.
	rate float64
}

func NewFlatRateProvider(rate float64) *FlatRateProvider {
	return &FlatRateProvider{rate: rate}
}

func (p *FlatRateProvider) CalculateRate(_ shipping.Package) (float64, error) {
	return p.rate, nil
}

func (p *FlatRateProvider) GetName() string {
	return "Flat Rate Shipping"
}
