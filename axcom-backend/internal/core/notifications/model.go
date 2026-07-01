// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package notifications

import "time"

type Notification struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Type    string    `json:"type"` // e.g. email, sms, webhook
	Message string    `json:"message"`
	SentAt  time.Time `json:"sent_at"`
	Status  string    `json:"status"` // sent, failed
}

type Event struct {
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
}
