// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shippingwiring

import (
	"context"

	"ecom-engine/internal/core/orders"
)

// orderAdapter adapts orders.Service to the shipping controller's OrderService interface.
type orderAdapter struct {
	svc orders.Service
}

func (a *orderAdapter) GetOrder(ctx context.Context, id string) (*orders.Order, error) {
	return a.svc.GetOrder(ctx, id)
}
