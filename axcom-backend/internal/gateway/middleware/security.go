// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides HTTP middleware handlers for request processing, authentication, and security.
package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware attaches standard security response headers to mitigate common web vulnerabilities.
// If the provided csp parameter is empty, it attempts to load from the CSP_DIRECTIVES environment variable,
// falling back to a default strict policy ("default-src 'self'").
func SecurityHeadersMiddleware(csp string) gin.HandlerFunc {
	if csp == "" {
		csp = os.Getenv("CSP_DIRECTIVES")
		if csp == "" {
			csp = "default-src 'self'"
		}
	}
	return func(c *gin.Context) {
		// Enforce HTTPS connections for subdomains.
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent browsers from performing content-type sniffing.
		c.Header("X-Content-Type-Options", "nosniff")

		// Restrict iframe embedding to prevent clickjacking.
		c.Header("X-Frame-Options", "DENY")

		// Activate legacy browser cross-site scripting filters.
		c.Header("X-XSS-Protection", "1; mode=block")

		// Limit resource loading to the server origin or configured domains.
		c.Header("Content-Security-Policy", csp)

		// Control referrer header leaks on cross-origin requests.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features (camera, microphone, geolocation, etc.)
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		c.Next()
	}
}

// RequestSizeLimitMiddleware limits the maximum body size of incoming requests.
func RequestSizeLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
