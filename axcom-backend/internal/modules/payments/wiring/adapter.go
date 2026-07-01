// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package paymentswiring

import (
	"context"

	"ecom-engine/internal/core/orders"
)

// orderAdapter adapts orders.Service to the payments.OrderFetcher interface.
type orderAdapter struct {
	svc orders.Service
}

func (a *orderAdapter) GetOrderAmountAndStatus(ctx context.Context, orderID string) (float64, string, error) {
	order, err := a.svc.GetOrder(ctx, orderID)
	if err != nil {
		return 0, "", err
	}
	return order.Total, string(order.Status), nil
}
