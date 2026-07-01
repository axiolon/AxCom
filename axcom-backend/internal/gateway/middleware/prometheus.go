// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"strconv"
	"time"

	"ecom-engine/pkg/metrics"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware records per-request Prometheus metrics:
//   - ecom_engine_http_requests_in_flight  (gauge)
//   - ecom_engine_http_requests_total      (counter, labels: method / route / status)
//   - ecom_engine_http_request_duration_seconds (histogram, labels: method / route)
//
// The route label uses c.FullPath() which returns the gin route template
// (e.g. /api/v1/products/:id) rather than the raw URL, keeping cardinality low.
// Unmatched paths (404s) are bucketed under the "unmatched" route label.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics.HTTPRequestsInFlight.Inc()
		start := time.Now()

		c.Next()

		metrics.HTTPRequestsInFlight.Dec()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		metrics.HTTPRequestsTotal.
			WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).
			Inc()

		metrics.HTTPRequestDuration.
			WithLabelValues(c.Request.Method, route).
			Observe(time.Since(start).Seconds())
	}
}
