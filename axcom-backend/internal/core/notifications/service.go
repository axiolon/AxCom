// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package notifications

import (
	"context"
	"ecom-engine/internal/events"
	"fmt"
	"time"
)

type NotificationService struct {
	eventBus events.EventBus
}

func NewNotificationService(eventBus events.EventBus) *NotificationService {
	s := &NotificationService{eventBus: eventBus}
	// Subscribe to order events
	eventBus.Subscribe(events.OrderCreatedTopic, s.handleOrderCreated)
	eventBus.Subscribe(events.OrderPaidTopic, s.handleOrderPaid)
	return s
}

func (s *NotificationService) Send(_ context.Context, userID, nType, msg string) (*Notification, error) {
	n := &Notification{
		ID:      "ntf_" + time.Now().Format("20060102150405"),
		UserID:  userID,
		Type:    nType,
		Message: msg,
		SentAt:  time.Now(),
		Status:  "sent",
	}
	fmt.Printf("[Notification] Sending %s to user %s: %s\n", nType, userID, msg)
	return n, nil
}

func (s *NotificationService) handleOrderCreated(ev events.Event) error {
	fmt.Printf("[Notification] Handling order created event: %v\n", ev.Payload)
	return nil
}

func (s *NotificationService) handleOrderPaid(ev events.Event) error {
	fmt.Printf("[Notification] Handling order paid event: %v\n", ev.Payload)
	return nil
}
