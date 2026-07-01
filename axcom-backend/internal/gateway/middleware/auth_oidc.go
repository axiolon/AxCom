// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"ecom-engine/internal/core/auth"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/response"
	"ecom-engine/pkg/token"
	"strings"

	"github.com/gin-gonic/gin"
)

// OIDCAuthMiddleware validates an external OIDC Bearer token via JWKS,
// then syncs the user identity with the local database.
func OIDCAuthMiddleware(validator *token.OIDCValidator, authService auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.GinWriteError(c, apperrors.NewUnauthorized("authorization header required", nil))
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.GinWriteError(c, apperrors.NewUnauthorized("invalid authorization header format", nil))
			c.Abort()
			return
		}

		claims, err := validator.Validate(parts[1])
		if err != nil {
			response.GinWriteError(c, apperrors.NewUnauthorized("invalid or expired external token", err))
			c.Abort()
			return
		}

		// Choose the first role if multiple are present; default to 'customer'
		role := "customer"
		if len(claims.Roles) > 0 {
			role = claims.Roles[0]
		}

		user, err := authService.SyncOIDCUser(c.Request.Context(), claims.Subject, claims.Email, claims.Name, role)
		if err != nil {
			response.GinWriteError(c, apperrors.NewInternal("failed to sync authenticated user", err))
			c.Abort()
			return
		}

		// Inject headers for compatibility with simple downstream handlers.
		c.Request.Header.Set("X-User-ID", user.ID)
		c.Request.Header.Set("X-User-Role", user.Role)

		c.Set(string(ctxkeys.UserIDKey), user.ID)
		c.Set(string(ctxkeys.UserRoleKey), user.Role)

		// Propagate context to standard library http.Request context for non-Gin packages.
		ctx := context.WithValue(c.Request.Context(), ctxkeys.UserIDKey, user.ID)
		ctx = context.WithValue(ctx, ctxkeys.UserRoleKey, user.Role)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
