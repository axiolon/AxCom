// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package core implements the core product and category catalog features,
// defining service layers, repositories, controllers, validation rules,
// and route registrations for standard product management operations.
package core

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all core catalog HTTP endpoints.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware, adminOnlyMiddleware gin.HandlerFunc) {
	// Public routes
	rg.GET("/products", ctrl.ListProducts)
	rg.GET("/products/search", ctrl.ListProducts) // support search alias
	rg.GET("/products/:id", ctrl.GetProduct)
	rg.GET("/categories", ctrl.ListCategories)
	rg.GET("/categories/:id", ctrl.GetCategory) // missing single category fetch

	// Auth-guarded admin routes
	rg.POST("/products", authMiddleware, adminOnlyMiddleware, ctrl.CreateProduct)
	rg.PUT("/products/:id", authMiddleware, adminOnlyMiddleware, ctrl.UpdateProduct)
	rg.DELETE("/products/:id", authMiddleware, adminOnlyMiddleware, ctrl.DeleteProduct)

	rg.POST("/categories", authMiddleware, adminOnlyMiddleware, ctrl.CreateCategory)
	rg.PUT("/categories/:id", authMiddleware, adminOnlyMiddleware, ctrl.UpdateCategory)
	rg.DELETE("/categories/:id", authMiddleware, adminOnlyMiddleware, ctrl.DeleteCategory)
}
