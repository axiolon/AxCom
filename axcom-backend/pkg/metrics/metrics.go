// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package metrics registers all Prometheus metrics for ecom-engine and provides
// helpers for wiring them into the HTTP router and infrastructure adapters.
package metrics

import (
	infradb "ecom-engine/internal/infra/db"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const ns = "ecom_engine"

// ---------------------------------------------------------------------------
// HTTP metrics
// ---------------------------------------------------------------------------

// HTTPRequestsTotal counts every completed HTTP request, labelled by HTTP
// method, route template (e.g. /api/v1/products/:id), and status code.
var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "http_requests_total",
		Help:      "Total HTTP requests completed, partitioned by method, route, and status code.",
	},
	[]string{"method", "route", "status"},
)

// HTTPRequestDuration tracks request latency as a histogram, labelled by
// method and route template. Buckets cover sub-millisecond to 10-second spans.
var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds, partitioned by method and route.",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	},
	[]string{"method", "route"},
)

// HTTPRequestsInFlight is the current number of HTTP requests being handled.
var HTTPRequestsInFlight = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "http_requests_in_flight",
		Help:      "Current number of HTTP requests actively being processed.",
	},
)

// ---------------------------------------------------------------------------
// DB pool collector
// ---------------------------------------------------------------------------

// DBPoolCollector implements prometheus.Collector and exposes connection pool
// metrics sourced from any infradb.PoolStatsProvider (currently PostgreSQL only;
// MongoDB's driver does not expose pgx-style pool statistics).
type DBPoolCollector struct {
	provider infradb.PoolStatsProvider

	maxConns          *prometheus.Desc
	totalConns        *prometheus.Desc
	acquiredConns     *prometheus.Desc
	idleConns         *prometheus.Desc
	acquireCount      *prometheus.Desc
	emptyAcquireCount *prometheus.Desc
	acquireDuration   *prometheus.Desc
}

// NewDBPoolCollector creates a collector wrapping the given PoolStatsProvider.
func NewDBPoolCollector(p infradb.PoolStatsProvider) *DBPoolCollector {
	fqn := func(sub, name string) string {
		return prometheus.BuildFQName(ns, sub, name)
	}
	return &DBPoolCollector{
		provider: p,
		maxConns: prometheus.NewDesc(
			fqn("db_pool", "max_conns"),
			"Maximum number of connections allowed in the pool.", nil, nil,
		),
		totalConns: prometheus.NewDesc(
			fqn("db_pool", "total_conns"),
			"Current total connections in the pool (acquired + idle).", nil, nil,
		),
		acquiredConns: prometheus.NewDesc(
			fqn("db_pool", "acquired_conns"),
			"Connections currently checked out and in use by the application.", nil, nil,
		),
		idleConns: prometheus.NewDesc(
			fqn("db_pool", "idle_conns"),
			"Connections sitting idle and available for acquisition.", nil, nil,
		),
		acquireCount: prometheus.NewDesc(
			fqn("db_pool", "acquire_count_total"),
			"Cumulative number of successful connection acquisitions.", nil, nil,
		),
		emptyAcquireCount: prometheus.NewDesc(
			fqn("db_pool", "empty_acquire_count_total"),
			"Cumulative acquisitions that had to wait because the pool was empty.", nil, nil,
		),
		acquireDuration: prometheus.NewDesc(
			fqn("db_pool", "acquire_duration_seconds_total"),
			"Cumulative time spent waiting to acquire connections, in seconds.", nil, nil,
		),
	}
}

// Describe sends each metric descriptor to the Prometheus registry.
func (c *DBPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxConns
	ch <- c.totalConns
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.acquireCount
	ch <- c.emptyAcquireCount
	ch <- c.acquireDuration
}

// Collect reads the live pool stats and emits current metric values.
func (c *DBPoolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.provider.PoolStats()
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(s.MaxConns))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(s.TotalConns))
	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(s.AcquiredConns))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(s.IdleConns))
	ch <- prometheus.MustNewConstMetric(c.acquireCount, prometheus.CounterValue, float64(s.AcquireCount))
	ch <- prometheus.MustNewConstMetric(c.emptyAcquireCount, prometheus.CounterValue, float64(s.EmptyAcquireCount))
	ch <- prometheus.MustNewConstMetric(c.acquireDuration, prometheus.CounterValue, s.AcquireDuration.Seconds())
}
