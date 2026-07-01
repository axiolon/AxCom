// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

type Package struct {
	Weight float64
	Value  float64
}

type ShippingProvider interface { //nolint:revive // Name is intentionally explicit for the public API.
	CalculateRate(pkg Package) (float64, error)
	GetName() string
}
