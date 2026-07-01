// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewInternalServer returns an http.Server that serves only the Prometheus
// /metrics endpoint on the given address. This server should be bound to an
// internal-only port (e.g. :9090) that is never exposed to the public internet.
func NewInternalServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}
