// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles incoming HTTP requests for the core catalog feature,
// converting HTTP payloads and query parameters into domain models and
// dispatching them to the catalog QueryService and CommandService.
type Controller struct {
	querySvc   QueryService
	commandSvc CommandService
}

// NewController instantiates a new Controller with the provided catalog query and command services.
func NewController(querySvc QueryService, commandSvc CommandService) *Controller {
	return &Controller{
		querySvc:   querySvc,
		commandSvc: commandSvc,
	}
}

// ListProducts handles GET /products and GET /products/search.
func (ctrl *Controller) ListProducts(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received ListProducts request")

	var query ListProductsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid query parameters", err))
		return
	}

	products, err := ctrl.querySvc.GetProducts(c.Request.Context(), &query)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, products)
}

// GetProduct handles GET /products/:id.
func (ctrl *Controller) GetProduct(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetProduct request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	product, err := ctrl.querySvc.GetProduct(c.Request.Context(), id)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, product)
}

// CreateProduct handles POST /products.
func (ctrl *Controller) CreateProduct(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received CreateProduct request")

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid product payload", err))
		return
	}

	variants := make([]domain.Variant, len(req.Variants))
	for i, v := range req.Variants {
		variants[i] = domain.Variant{
			ID:         v.ID,
			SKU:        v.SKU,
			Name:       v.Name,
			Price:      v.Price,
			Attributes: v.Attributes,
		}
	}

	p := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Variants:    variants,
	}

	if err := ctrl.commandSvc.AddProduct(c.Request.Context(), p); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, p)
}

// UpdateProduct handles PUT /products/:id.
func (ctrl *Controller) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received UpdateProduct request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid product update payload", err))
		return
	}

	variants := make([]domain.Variant, len(req.Variants))
	for i, v := range req.Variants {
		variants[i] = domain.Variant{
			ID:         v.ID,
			SKU:        v.SKU,
			Name:       v.Name,
			Price:      v.Price,
			Attributes: v.Attributes,
		}
	}

	p := &domain.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Variants:    variants,
	}

	if err := ctrl.commandSvc.UpdateProduct(c.Request.Context(), p); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, p)
}

// DeleteProduct handles DELETE /products/:id.
func (ctrl *Controller) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received DeleteProduct request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	if err := ctrl.commandSvc.DeleteProduct(c.Request.Context(), id); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "product deleted"})
}

// ListCategories handles GET /categories.
func (ctrl *Controller) ListCategories(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received ListCategories request")

	categories, err := ctrl.querySvc.GetCategories(c.Request.Context())
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, categories)
}

// GetCategory handles GET /categories/:id.
func (ctrl *Controller) GetCategory(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetCategory request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Category ID is required", nil))
		return
	}

	category, err := ctrl.querySvc.GetCategory(c.Request.Context(), id)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, category)
}

// CreateCategory handles POST /categories.
func (ctrl *Controller) CreateCategory(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received CreateCategory request")

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid category payload", err))
		return
	}

	cat := &domain.Category{
		Name:     req.Name,
		Slug:     req.Slug,
		ParentID: req.ParentID,
	}

	if err := ctrl.commandSvc.AddCategory(c.Request.Context(), cat); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, cat)
}

// UpdateCategory handles PUT /categories/:id.
func (ctrl *Controller) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received UpdateCategory request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Category ID is required", nil))
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid category update payload", err))
		return
	}

	cat := &domain.Category{
		ID:       id,
		Name:     req.Name,
		Slug:     req.Slug,
		ParentID: req.ParentID,
	}

	if err := ctrl.commandSvc.UpdateCategory(c.Request.Context(), cat); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, cat)
}

// DeleteCategory handles DELETE /categories/:id.
func (ctrl *Controller) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received DeleteCategory request for ID: %s", id)

	if id == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Category ID is required", nil))
		return
	}

	if err := ctrl.commandSvc.DeleteCategory(c.Request.Context(), id); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "category deleted"})
}
