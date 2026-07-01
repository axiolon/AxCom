// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"context"
	"ecom-engine/internal/core/catalog/domain"
	catalogCore "ecom-engine/internal/core/catalog/features/core"
)

type CatalogProductVerifier struct {
	coreSvc catalogCore.QueryService
}

// NewCatalogProductVerifier creates a new CatalogProductVerifier adapter.
func NewCatalogProductVerifier(coreSvc catalogCore.QueryService) ProductVerifier {
	return &CatalogProductVerifier{coreSvc: coreSvc}
}

// GetProduct retrieves a product entity using the catalog query service.
func (v *CatalogProductVerifier) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	return v.coreSvc.GetProductEntity(ctx, id)
}
