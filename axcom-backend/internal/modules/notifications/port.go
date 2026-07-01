// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package notifications

import "context"

type NotificationSender interface {
	Send(ctx context.Context, recipient, message string) error
	GetName() string
}
