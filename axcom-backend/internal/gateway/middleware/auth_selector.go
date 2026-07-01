// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"ecom-engine/internal/core/auth"
	"ecom-engine/pkg/token"

	"github.com/gin-gonic/gin"
)

// NewAuthMiddleware returns the appropriate Gin middleware based on the configured auth mode.
// mode should be "oidc" or "local".
func NewAuthMiddleware(mode string, jwtManager *token.JWTManager, oidcValidator *token.OIDCValidator, authService auth.Service) gin.HandlerFunc {
	if mode == "oidc" && oidcValidator != nil {
		return OIDCAuthMiddleware(oidcValidator, authService)
	}
	return AuthMiddleware(jwtManager)
}
