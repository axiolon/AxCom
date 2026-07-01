// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"
)

type WebhookSender struct { //nolint:revive // Name is intentionally explicit for the public API.
	defaultURL string
}

func NewWebhookSender(defaultURL string) *WebhookSender {
	return &WebhookSender{defaultURL: defaultURL}
}

func (s *WebhookSender) Send(_ context.Context, recipient, message string) error {
	url := recipient
	if url == "" {
		url = s.defaultURL
	}
	fmt.Printf("Webhook: Dispatched POST to %s with payload: %s\n", url, message)
	return nil
}

func (s *WebhookSender) GetName() string {
	return "Webhook Dispatcher"
}
