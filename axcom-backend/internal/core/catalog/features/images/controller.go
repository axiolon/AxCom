// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Controller handles HTTP endpoints for product images.
type Controller struct {
	service Service
}

// NewController creates a new Controller.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// PresignUpload handles POST /products/:id/images/presign requests.
func (ctrl *Controller) PresignUpload(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received PresignUpload request for product ID: %s", productID)

	if productID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	var req PresignImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request body", err))
		return
	}

	uploads, err := ctrl.service.PresignUploadURLs(c.Request.Context(), productID, req.Files)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, PresignImagesResponse{Uploads: uploads})
}

// RegisterUploadedImages handles POST /products/:id/images/register requests.
func (ctrl *Controller) RegisterUploadedImages(c *gin.Context) {
	productID := c.Param("id")
	logger.InfoCtx(c.Request.Context(), "Received RegisterUploadedImages request for product ID: %s", productID)

	if productID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID is required", nil))
		return
	}

	var req RegisterImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(c, apperrors.NewBadRequest("invalid request body", err))
		return
	}

	images, err := ctrl.service.RegisterUploadedImages(c.Request.Context(), productID, req.Images)
	if err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, images)
}

// Delete handles DELETE /products/:id/images/:imageId requests.
func (ctrl *Controller) Delete(c *gin.Context) {
	productID := c.Param("id")
	imageID := c.Param("imageId")
	logger.InfoCtx(c.Request.Context(), "Received Delete image request for product ID: %s, image ID: %s", productID, imageID)

	if productID == "" || imageID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID and Image ID are required", nil))
		return
	}

	if err := ctrl.service.DeleteImage(c.Request.Context(), productID, imageID); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "image deleted"})
}

// SetPrimary handles PUT /products/:id/images/:imageId/primary requests.
func (ctrl *Controller) SetPrimary(c *gin.Context) {
	productID := c.Param("id")
	imageID := c.Param("imageId")
	logger.InfoCtx(c.Request.Context(), "Received SetPrimary image request for product ID: %s, image ID: %s", productID, imageID)

	if productID == "" || imageID == "" {
		response.GinWriteError(c, apperrors.NewBadRequest("Product ID and Image ID are required", nil))
		return
	}

	if err := ctrl.service.SetPrimaryImage(c.Request.Context(), productID, imageID); err != nil {
		response.GinWriteError(c, err)
		return
	}

	response.GinOK(c, gin.H{"message": "primary image updated"})
}
