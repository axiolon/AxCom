// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/pkg/logger"
)

type DashboardAlertDispatcher struct {
	repo Repository
}

func NewDashboardAlertDispatcher(repo Repository) domain.AlertDispatcher {
	return &DashboardAlertDispatcher{
		repo: repo,
	}
}

func (d *DashboardAlertDispatcher) Dispatch(ctx context.Context, alert domain.Alert) error {
	logger.InfoCtx(ctx, "Dispatching alert: %s (Variant: %s)", alert.Message, alert.VariantID)

	if err := d.repo.SaveAlert(ctx, &alert); err != nil {
		logger.ErrorCtx(ctx, "Failed to save alert to dashboard: %v", err)
		return err
	}

	return nil
}
