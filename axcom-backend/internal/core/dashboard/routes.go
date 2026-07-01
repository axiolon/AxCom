// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dashboard

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the dashboard endpoint on the admin router group.
// The admin group already enforces auth + admin-role middleware.
func RegisterRoutes(adminGroup *gin.RouterGroup, h *Handler) {
	adminGroup.GET("/admin/dashboard", h.GetStats)
}
