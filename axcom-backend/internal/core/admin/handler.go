// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

import "github.com/gin-gonic/gin"

// AdminHandler holds admin meta-routes that are always registered,
// independent of the dashboard module.
type AdminHandler struct{} //nolint:revive // Name is intentionally explicit for the public API.

// NewAdminHandler constructs an AdminHandler.
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// RegisterRoutes registers always-on admin meta-routes.
// Dashboard stats are served by the dashboard module (internal/modules/dashboard).
func RegisterRoutes(rg *gin.RouterGroup, h *AdminHandler) {
	// Reserved for future admin meta-endpoints (e.g. config reload, module status).
	_ = h
	_ = rg
}
