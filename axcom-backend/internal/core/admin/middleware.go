// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"ecom-engine/pkg/ctxkeys"
	"ecom-engine/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminOnlyMiddleware() gin.HandlerFunc { //nolint:revive // Name is intentionally explicit for the public API.
	return func(c *gin.Context) {
		role := c.GetString(string(ctxkeys.UserRoleKey))
		if role != "admin" {
			response.GinError(c, http.StatusForbidden, "forbidden: admin role required")
			return
		}
		c.Next()
	}
}
