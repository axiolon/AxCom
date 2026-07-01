// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError(t *testing.T) {
	t.Run("creates BadRequest error", func(t *testing.T) {
		cause := errors.New("underlying cause")
		err := NewBadRequest("bad input", cause)

		assert.Equal(t, 400, err.Code)
		assert.Equal(t, "bad input", err.Message)
		assert.Equal(t, cause, err.Unwrap())
		assert.Contains(t, err.Error(), "code=400")
		assert.Contains(t, err.Error(), "message=\"bad input\"")
		assert.Contains(t, err.Error(), "err=underlying cause")
	})

	t.Run("creates TooManyRequests error", func(t *testing.T) {
		err := NewTooManyRequests("too fast", nil)

		assert.Equal(t, 429, err.Code)
		assert.Equal(t, "too fast", err.Message)
		assert.Nil(t, err.Unwrap())
		assert.Contains(t, err.Error(), "code=429")
		assert.Contains(t, err.Error(), "message=\"too fast\"")
	})

	t.Run("errors.As matching", func(t *testing.T) {
		var appErr *AppError
		err := NewNotFound("not found", nil)

		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})
}
