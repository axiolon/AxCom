// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides HTTP middleware handlers for request processing, authentication, and security.
package middleware

import (
	"context"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/response"
	"ecom-engine/pkg/token"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates incoming JWT bearer tokens and injects user identity into the request context.
func AuthMiddleware(jwtManager *token.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.GinWriteError(c, apperrors.NewUnauthorized("authorization header required", nil))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.GinWriteError(c, apperrors.NewUnauthorized("invalid authorization header format", nil))
			return
		}

		claims, err := jwtManager.Validate(parts[1])
		if err != nil {
			response.GinWriteError(c, apperrors.NewUnauthorized("invalid or expired token", err))
			return
		}

		// Inject headers for compatibility with simple downstream handlers.
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Role", claims.Role)

		c.Set(string(ctxkeys.UserIDKey), claims.UserID)
		c.Set(string(ctxkeys.UserRoleKey), claims.Role)

		// Propagate context to standard library http.Request context for non-Gin packages.
		ctx := context.WithValue(c.Request.Context(), ctxkeys.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, ctxkeys.UserRoleKey, claims.Role)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
