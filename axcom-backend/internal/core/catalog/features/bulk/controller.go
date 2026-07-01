// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

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

type BulkVariantDTO struct { //nolint:revive // Name is intentionally explicit for the public API.
	ID         string            `json:"id"`
	SKU        string            `json:"sku" binding:"required"`
	Name       string            `json:"name" binding:"required"`
	Price      float64           `json:"price" binding:"required,min=0"`
	Attributes map[string]string `json:"attributes"`
}

type BulkCreateProductRequest struct { //nolint:revive // Name is intentionally explicit for the public API.
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	CategoryID  string           `json:"category_id" binding:"required"`
	Variants    []BulkVariantDTO `json:"variants" binding:"required,dive"`
}

type BulkUpdateProductRequest struct { //nolint:revive // Name is intentionally explicit for the public API.
	ID          string           `json:"id" binding:"required"`
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	CategoryID  string           `json:"category_id" binding:"required"`
	Variants    []BulkVariantDTO `json:"variants" binding:"required,dive"`
}

type BulkDeleteRequest struct { //nolint:revive // Name is intentionally explicit for the public API.
	IDs []string `json:"ids" binding:"required,min=1"`
}

func (ctrl *Controller) BulkCreate(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received BulkCreate request")

	var req []BulkCreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid bulk create payload", err))
		return
	}

	products := make([]*domain.Product, len(req))
	for i, reqItem := range req {
		variants := make([]domain.Variant, len(reqItem.Variants))
		for j, v := range reqItem.Variants {
			variants[j] = domain.Variant{
				ID:         v.ID,
				SKU:        v.SKU,
				Name:       v.Name,
				Price:      v.Price,
				Attributes: v.Attributes,
			}
		}

		products[i] = &domain.Product{
			Name:        reqItem.Name,
			Description: reqItem.Description,
			CategoryID:  reqItem.CategoryID,
			Variants:    variants,
		}
	}

	if err := ctrl.service.BulkCreate(c.Request.Context(), products); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, products)
}

func (ctrl *Controller) BulkUpdate(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received BulkUpdate request")

	var req []BulkUpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid bulk update payload", err))
		return
	}

	products := make([]*domain.Product, len(req))
	for i, reqItem := range req {
		variants := make([]domain.Variant, len(reqItem.Variants))
		for j, v := range reqItem.Variants {
			variants[j] = domain.Variant{
				ID:         v.ID,
				SKU:        v.SKU,
				Name:       v.Name,
				Price:      v.Price,
				Attributes: v.Attributes,
			}
		}

		products[i] = &domain.Product{
			ID:          reqItem.ID,
			Name:        reqItem.Name,
			Description: reqItem.Description,
			CategoryID:  reqItem.CategoryID,
			Variants:    variants,
		}
	}

	if err := ctrl.service.BulkUpdate(c.Request.Context(), products); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, products)
}

func (ctrl *Controller) BulkDelete(c *gin.Context) {
	logger.InfoCtx(c.Request.Context(), "Received BulkDelete request")

	var req BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("Invalid bulk delete payload", err))
		return
	}

	if err := ctrl.service.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "products bulk deleted"})
}
