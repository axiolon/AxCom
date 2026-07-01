// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package freeabove

import "ecom-engine/internal/modules/shipping"

type FreeAboveProvider struct { //nolint:revive // Name is intentionally explicit for the public API.
	threshold float64
	baseRate  float64
}

func NewFreeAboveProvider(threshold, baseRate float64) *FreeAboveProvider {
	return &FreeAboveProvider{
		threshold: threshold,
		baseRate:  baseRate,
	}
}

func (p *FreeAboveProvider) CalculateRate(pkg shipping.Package) (float64, error) {
	if pkg.Value >= p.threshold {
		return 0.0, nil
	}
	return p.baseRate, nil
}

func (p *FreeAboveProvider) GetName() string {
	return "Free Above Threshold Shipping"
}
