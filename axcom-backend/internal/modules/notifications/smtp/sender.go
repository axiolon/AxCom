// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"context"
	"fmt"
)

type SMTPSender struct { //nolint:revive // Name is intentionally explicit for the public API.
	host string
	port int
}

func NewSMTPSender(host string, port int) *SMTPSender {
	return &SMTPSender{host: host, port: port}
}

func (s *SMTPSender) Send(_ context.Context, recipient, message string) error {
	fmt.Printf("SMTP: Sent email to %s via %s:%d - Content: %s\n", recipient, s.host, s.port, message)
	return nil
}

func (s *SMTPSender) GetName() string {
	return "SMTP Email Sender"
}
