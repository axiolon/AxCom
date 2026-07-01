// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides HTTP middleware handlers for request processing, authentication, and security.
package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware configures cross-origin resource sharing (CORS) headers and handles HTTP OPTIONS preflight requests.
func CORSMiddleware() gin.HandlerFunc {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := make(map[string]struct{})
	if allowedOriginsStr != "" {
		for _, origin := range strings.Split(allowedOriginsStr, ",") {
			allowedOrigins[strings.TrimSpace(origin)] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			allowed := false
			if len(allowedOrigins) == 0 {
				c.Header("Access-Control-Allow-Origin", "*")
				allowed = true
			} else {
				if _, ok := allowedOrigins[origin]; ok { // we are using map instead of slice for O(1) look up and readability
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin") // for caching instructions for proxies etc
					allowed = true
				}
			}

			if allowed {
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role")
			}
		}
		// Options check if its present and save it for 24h in browser
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Max-Age", "86400") // Cache preflight for 24h
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
