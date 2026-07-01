// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"net/http"

	"ecom-engine/pkg/response"

	"github.com/gin-gonic/gin"
)

// Handler exposes the dashboard service over HTTP.
type Handler struct {
	svc Service
}

// NewHandler constructs a Handler with the given DashboardService.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// GetStats handles GET /admin/dashboard.
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.svc.GetStats(c.Request.Context())
	if err != nil {
		response.GinError(c, http.StatusInternalServerError, "failed to fetch dashboard stats")
		return
	}
	response.GinOK(c, stats)
}
