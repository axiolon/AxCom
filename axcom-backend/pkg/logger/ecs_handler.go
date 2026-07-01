// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ECSHandler is a slog.Handler that outputs Elastic Common Schema (ECS) 8.11
// formatted JSON, suitable for direct ingestion by Filebeat/Elasticsearch.
type ECSHandler struct {
	w     io.Writer
	level slog.Leveler
	attrs []slog.Attr
	group string
	mu    sync.Mutex

	serviceName string
}

// ECSHandlerOptions configures the ECS handler.
type ECSHandlerOptions struct {
	Level       slog.Leveler
	ServiceName string
}

// NewECSHandler creates a new ECS-compatible slog.Handler.
func NewECSHandler(w io.Writer, opts *ECSHandlerOptions) *ECSHandler {
	h := &ECSHandler{w: w}
	if opts != nil {
		h.level = opts.Level
		h.serviceName = opts.ServiceName
	}
	if h.level == nil {
		h.level = slog.LevelInfo
	}
	if h.serviceName == "" {
		h.serviceName = "ecom-engine"
	}
	return h
}

func (h *ECSHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *ECSHandler) Handle(_ context.Context, r slog.Record) error {
	m := make(map[string]any, 8)

	m["@timestamp"] = r.Time.UTC().Format(time.RFC3339Nano)
	m["log.level"] = strings.ToLower(r.Level.String())
	m["message"] = r.Message
	m["service.name"] = h.serviceName
	m["ecs.version"] = "8.11"

	// Merge pre-added attrs.
	for _, a := range h.attrs {
		ecsKey(m, a)
	}

	// Merge per-record attrs.
	r.Attrs(func(a slog.Attr) bool {
		ecsKey(m, a)
		return true
	})

	buf, err := json.Marshal(m)
	if err != nil {
		return err
	}
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err = h.w.Write(buf)
	return err
}

func (h *ECSHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ECSHandler{
		w:           h.w,
		level:       h.level,
		attrs:       append(cloneAttrs(h.attrs), attrs...),
		group:       h.group,
		serviceName: h.serviceName,
	}
}

func (h *ECSHandler) WithGroup(name string) slog.Handler {
	return &ECSHandler{
		w:           h.w,
		level:       h.level,
		attrs:       cloneAttrs(h.attrs),
		group:       name,
		serviceName: h.serviceName,
	}
}

// ecsKey maps slog attribute keys to ECS field names.
func ecsKey(m map[string]any, a slog.Attr) {
	switch a.Key {
	case "trace_id":
		m["trace.id"] = a.Value.String()
	case "span_id":
		m["span.id"] = a.Value.String()
	default:
		m[a.Key] = a.Value.Any()
	}
}

func cloneAttrs(src []slog.Attr) []slog.Attr {
	if len(src) == 0 {
		return nil
	}
	dst := make([]slog.Attr, len(src))
	copy(dst, src)
	return dst
}
