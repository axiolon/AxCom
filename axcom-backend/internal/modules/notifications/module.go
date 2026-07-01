// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package notifications

import (
	"context"

	"ecom-engine/internal/core/notifications"
	"ecom-engine/internal/engine"

	"github.com/gin-gonic/gin"
)

// Module wires the notifications domain. No dependencies on other modules.
type Module struct {
	svc *notifications.NotificationService
}

func New(_ engine.Config) engine.Module {
	return &Module{}
}

func (m *Module) Name() string        { return "notifications" }
func (m *Module) Requires() []string  { return nil }
func (m *Module) BasePaths() []string { return []string{"/notifications"} }

func (m *Module) Init(c *engine.Container) error {
	m.svc = notifications.NewNotificationService(c.EventBus)
	return nil
}

func (m *Module) RegisterRoutes(_, secured, _ *gin.RouterGroup) {
	notifications.RegisterRoutes(secured, notifications.NewNotificationHandler(m.svc))
}

func (m *Module) Shutdown(_ context.Context) error { return nil }
