// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package notifications

import (
	"ecom-engine/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service *NotificationService
}

func NewNotificationHandler(service *NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// RegisterRoutes registers notifications routes on the router group.
func RegisterRoutes(rg *gin.RouterGroup, h *NotificationHandler) {
	rg.POST("/notifications/send", h.SendNotification)
}

func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, http.StatusBadRequest, err.Error())
		return
	}

	n, err := h.service.Send(c.Request.Context(), req.UserID, req.Type, req.Message)
	if err != nil {
		response.GinError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.GinOK(c, n)
}
