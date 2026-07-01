// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package response

import (
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// APIResponse defines the standard envelope structure for all JSON API responses.
type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// JSON sends a structured APIResponse JSON payload with the given HTTP status.
func JSON(w http.ResponseWriter, r *http.Request, status int, success bool, data any, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var traceID string
	if r != nil {
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}
	}

	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: success,
		Data:    data,
		Error:   errMsg,
		TraceID: traceID,
	})
}

// OK is a helper to write a successful 200 OK JSON response.
func OK(w http.ResponseWriter, r *http.Request, data any) {
	JSON(w, r, http.StatusOK, true, data, "")
}

// Error writes an RFC 7807 problem detail response with the given HTTP status.
func Error(w http.ResponseWriter, r *http.Request, status int, errMsg string) {
	writeProblem(w, r, ProblemDetail{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: errMsg,
	})
}

// WriteError inspects the error and writes the appropriate RFC 7807 problem detail response.
// If the error is an *AppError, it uses its status and type. Otherwise returns 500.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()
	if appErr, ok := err.(*apperrors.AppError); ok {
		if appErr.Err != nil {
			logger.ErrorCtx(ctx, "Request failed [%d]: %s: %v", appErr.Code, appErr.Message, appErr.Err)
		} else {
			logger.ErrorCtx(ctx, "Request failed [%d]: %s", appErr.Code, appErr.Message)
		}
		problemType := appErr.Type
		if problemType == "" {
			problemType = "about:blank"
		}
		writeProblem(w, r, ProblemDetail{
			Type:   problemType,
			Title:  http.StatusText(appErr.Code),
			Status: appErr.Code,
			Detail: appErr.Message,
		})
		return
	}

	logger.ErrorCtx(ctx, "Unhandled runtime error: %v", err)
	writeProblem(w, r, ProblemDetail{
		Type:   "about:blank",
		Title:  http.StatusText(http.StatusInternalServerError),
		Status: http.StatusInternalServerError,
		Detail: "Internal server error",
	})
}

// GinOK is a helper to write a successful JSON response using Gin.
func GinOK(c *gin.Context, data any) {
	var traceID string
	if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		TraceID: traceID,
	})
}

// GinError writes an RFC 7807 problem detail error response using Gin.
func GinError(c *gin.Context, status int, errMsg string) {
	ginWriteProblem(c, ProblemDetail{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: errMsg,
	})
}

// GinWriteError writes RFC 7807 error responses using Gin context.
func GinWriteError(c *gin.Context, err error) {
	ctx := c.Request.Context()
	if appErr, ok := err.(*apperrors.AppError); ok {
		if appErr.Err != nil {
			logger.ErrorCtx(ctx, "Request failed [%d]: %s: %v", appErr.Code, appErr.Message, appErr.Err)
		} else {
			logger.ErrorCtx(ctx, "Request failed [%d]: %s", appErr.Code, appErr.Message)
		}
		problemType := appErr.Type
		if problemType == "" {
			problemType = "about:blank"
		}
		ginWriteProblem(c, ProblemDetail{
			Type:   problemType,
			Title:  http.StatusText(appErr.Code),
			Status: appErr.Code,
			Detail: appErr.Message,
		})
		return
	}

	logger.ErrorCtx(ctx, "Unhandled runtime error: %v", err)
	ginWriteProblem(c, ProblemDetail{
		Type:   "about:blank",
		Title:  http.StatusText(http.StatusInternalServerError),
		Status: http.StatusInternalServerError,
		Detail: "Internal server error",
	})
}

// writeProblem writes an RFC 7807 problem detail response using net/http.
func writeProblem(w http.ResponseWriter, r *http.Request, p ProblemDetail) {
	if r != nil {
		p.Instance = r.URL.Path
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			p.TraceID = span.SpanContext().TraceID().String()
		}
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// ginWriteProblem writes an RFC 7807 problem detail response using Gin.
func ginWriteProblem(c *gin.Context, p ProblemDetail) {
	p.Instance = c.Request.URL.Path
	if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
		p.TraceID = span.SpanContext().TraceID().String()
	}
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(p.Status, p)
}
