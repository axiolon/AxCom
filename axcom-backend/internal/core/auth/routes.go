// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth provides user registration, credential validation, session tokens, and password recovery.
package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all authentication HTTP endpoints onto the provided RouterGroup.
func RegisterRoutes(rg *gin.RouterGroup, ctrl *Controller) {
	rg.POST("/auth/register", ctrl.Register)
	rg.POST("/auth/login", ctrl.Login)
	rg.POST("/auth/logout", ctrl.Logout)
	rg.POST("/auth/refresh", ctrl.Refresh)
	rg.POST("/auth/password-reset", ctrl.RequestPasswordReset)
	rg.POST("/auth/password-reset/confirm", ctrl.ConfirmPasswordReset)
}
