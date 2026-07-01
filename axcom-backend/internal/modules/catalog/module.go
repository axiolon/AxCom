// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"

	catalogBulk "ecom-engine/internal/core/catalog/features/bulk"
	catalogCore "ecom-engine/internal/core/catalog/features/core"
	catalogDiscounts "ecom-engine/internal/core/catalog/features/discounts"
	catalogImages "ecom-engine/internal/core/catalog/features/images"
	catalogReviews "ecom-engine/internal/core/catalog/features/reviews"
	catalogVariants "ecom-engine/internal/core/catalog/features/variants"
	"ecom-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

// Module wires the catalog domain. Core (products + categories) always runs when
// the module is enabled. Images, variants, discounts, bulk, and reviews are optional.
type Module struct {
	cfg     engine.CatalogModuleConfig
	authMW  gin.HandlerFunc
	adminMW gin.HandlerFunc

	coreQuery   catalogCore.QueryService
	coreCommand catalogCore.CommandService

	imagesSvc    catalogImages.Service
	variantsSvc  catalogVariants.Service
	discountsSvc catalogDiscounts.Service
	bulkSvc      catalogBulk.Service
	reviewsSvc   catalogReviews.Service
}

func New(cfg engine.Config) engine.Module {
	return &Module{cfg: cfg.Modules.Catalog}
}

func (m *Module) Name() string       { return "catalog" }
func (m *Module) Requires() []string { return nil }
func (m *Module) BasePaths() []string {
	return []string{"/products", "/categories", "/reviews", "/catalog"}
}

func (m *Module) Init(c *engine.Container) error {
	m.authMW = c.AuthMiddleware
	m.adminMW = c.AdminMiddleware

	coreRepo := c.Repos.CatalogCoreRepo()
	m.coreQuery = catalogCore.NewCatalogQueryService(coreRepo, c.CacheManager)
	m.coreCommand = catalogCore.NewCatalogCommandService(coreRepo, c.CacheManager)
	m.coreCommand.SubscribeStockEvents(c.EventBus)

	c.Provide(engine.ServiceCatalogQuery, m.coreQuery)
	c.Provide(engine.ServiceCatalogCommand, m.coreCommand)

	if m.cfg.Features.Images {
		m.imagesSvc = catalogImages.NewService(c.Repos.CatalogImagesRepo(), c.FileStorage)
		c.Provide(engine.ServiceCatalogImages, m.imagesSvc)
	}
	if m.cfg.Features.Variants {
		m.variantsSvc = catalogVariants.NewService(c.Repos.CatalogVariantsRepo())
		c.Provide(engine.ServiceCatalogVariants, m.variantsSvc)
	}
	if m.cfg.Features.Discounts {
		m.discountsSvc = catalogDiscounts.NewService(c.Repos.CatalogDiscountsRepo())
		c.Provide(engine.ServiceCatalogDiscounts, m.discountsSvc)
	}
	if m.cfg.Features.Bulk {
		m.bulkSvc = catalogBulk.NewService(c.Repos.CatalogBulkRepo())
		c.Provide(engine.ServiceCatalogBulk, m.bulkSvc)
	}
	if m.cfg.Features.Reviews {
		verifier := catalogReviews.NewCatalogProductVerifier(m.coreQuery)
		m.reviewsSvc = catalogReviews.NewReviewService(c.Repos.CatalogReviewsRepo(), verifier)
		c.Provide(engine.ServiceCatalogReviews, m.reviewsSvc)
	}

	return nil
}

func (m *Module) RegisterRoutes(public, _, _ *gin.RouterGroup) {
	catalogCore.RegisterRoutes(public, catalogCore.NewController(m.coreQuery, m.coreCommand), m.authMW, m.adminMW)

	if m.imagesSvc != nil {
		catalogImages.RegisterRoutes(public, catalogImages.NewController(m.imagesSvc), m.authMW, m.adminMW)
	}
	if m.variantsSvc != nil {
		catalogVariants.RegisterRoutes(public, catalogVariants.NewController(m.variantsSvc), m.authMW, m.adminMW)
	}
	if m.discountsSvc != nil {
		catalogDiscounts.RegisterRoutes(public, catalogDiscounts.NewController(m.discountsSvc), m.authMW, m.adminMW)
	}
	if m.bulkSvc != nil {
		catalogBulk.RegisterRoutes(public, catalogBulk.NewController(m.bulkSvc), m.authMW, m.adminMW)
	}
	if m.reviewsSvc != nil {
		catalogReviews.RegisterRoutes(public, catalogReviews.NewController(m.reviewsSvc), m.authMW, m.adminMW)
	}
}

func (m *Module) Shutdown(_ context.Context) error { return nil }
