// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package admin

// TransitionRequest is the JSON contract for POST /admin/orders/:id/transition
type TransitionRequest struct {
	Action string `json:"action" binding:"required"`
}
