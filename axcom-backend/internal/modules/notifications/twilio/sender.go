// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package twilio

import (
	"context"
	"fmt"
)

type TwilioSender struct { //nolint:revive // Name is intentionally explicit for the public API.
	accountSid string
}

func NewTwilioSender(accountSid string) *TwilioSender {
	return &TwilioSender{accountSid: accountSid}
}

func (s *TwilioSender) Send(_ context.Context, recipient, message string) error {
	fmt.Printf("Twilio: Sent SMS to %s - Content: %s\n", recipient, message)
	return nil
}

func (s *TwilioSender) GetName() string {
	return "Twilio SMS Sender"
}
