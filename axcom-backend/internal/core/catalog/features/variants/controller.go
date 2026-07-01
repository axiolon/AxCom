// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import (
	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	service Service
}

func NewController(service Service) *Controller {
	return &Controller{service: service}
}

type CreateVariantRequest struct {
	SKU          string            `json:"sku" binding:"required"`
	Name         string            `json:"name" binding:"required"`
	Price        float64           `json:"price" binding:"required,min=0"`
	InitialStock *int              `json:"initial_stock,omitempty"`
	Attributes   map[string]string `json:"attributes"`
}

type UpdateVariantRequest struct {
	SKU        string            `json:"sku" binding:"required"`
	Name       string            `json:"name" binding:"required"`
	Price      float64           `json:"price" binding:"required,min=0"`
	Attributes map[string]string `json:"attributes"`
}

func (ctrl *Controller) GetVariants(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received GetVariants request for product ID: %s", productID)

	if productID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	variants, err := ctrl.service.GetVariants(c.Request.Context(), productID)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, variants)
}

func (ctrl *Controller) AddVariant(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received AddVariant request for product ID: %s", productID)

	if productID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	var req CreateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid variant payload", err))
		return
	}

	stock := 0
	if req.InitialStock != nil {
		stock = *req.InitialStock
	}

	v := &domain.Variant{
		SKU:        req.SKU,
		Name:       req.Name,
		Price:      req.Price,
		Stock:      stock,
		Attributes: req.Attributes,
	}

	if err := ctrl.service.AddVariant(c.Request.Context(), productID, v); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, v)
}

func (ctrl *Controller) UpdateVariant(c *gin.Context) {
	productID := c.Param("id")
	variantID := c.Param("variantId")
	logger.InfoCtx(c.Request.Context(), "Received UpdateVariant request for product ID: %s, variant ID: %s", productID, variantID)

	if productID == "" || variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID and Variant ID are required", nil))
		return
	}

	var req UpdateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid variant update payload", err))
		return
	}

	v := &domain.Variant{
		ID:         variantID,
		SKU:        req.SKU,
		Name:       req.Name,
		Price:      req.Price,
		Attributes: req.Attributes,
	}

	if err := ctrl.service.UpdateVariant(c.Request.Context(), productID, v); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, v)
}

func (ctrl *Controller) DeleteVariant(c *gin.Context) {
	productID := c.Param("id")
	variantID := c.Param("variantId")
	logger.InfoCtx(c.Request.Context(), "Received DeleteVariant request for product ID: %s, variant ID: %s", productID, variantID)

	if productID == "" || variantID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID and Variant ID are required", nil))
		return
	}

	if err := ctrl.service.DeleteVariant(c.Request.Context(), productID, variantID); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "variant deleted"})
}
