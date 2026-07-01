// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import "github.com/gin-gonic/gin"

// RegisterRoutes registers product images API routes onto the router.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller, authMiddleware, adminOnlyMiddleware gin.HandlerFunc) {
	// Protected endpoints (image uploads, deletions and primary setting are admin/auth operations)
	rg.POST("/products/:id/images/presign", authMiddleware, adminOnlyMiddleware, ctrl.PresignUpload)
	rg.POST("/products/:id/images/register", authMiddleware, adminOnlyMiddleware, ctrl.RegisterUploadedImages)
	rg.DELETE("/products/:id/images/:imageId", authMiddleware, adminOnlyMiddleware, ctrl.Delete)
	rg.PUT("/products/:id/images/:imageId/primary", authMiddleware, adminOnlyMiddleware, ctrl.SetPrimary)
}
