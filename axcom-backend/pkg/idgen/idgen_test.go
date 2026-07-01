// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package idgen

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	t.Run("generates with prefix and correct UUIDv7 format", func(t *testing.T) {
		prefix := "usr_"
		id, err := Generate(prefix)
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(id, prefix))

		// Stripped ID must be a valid UUID
		stripped := id[len(prefix):]
		parsed, err := uuid.Parse(stripped)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Version(7), parsed.Version())
	})

	t.Run("uniqueness of consecutive generations", func(t *testing.T) {
		id1, err := Generate("test_")
		assert.NoError(t, err)
		id2, err := Generate("test_")
		assert.NoError(t, err)
		assert.NotEqual(t, id1, id2)
	})

	t.Run("MustGenerate generates without panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			id := MustGenerate("evt_")
			assert.True(t, strings.HasPrefix(id, "evt_"))
			stripped := id[4:]
			parsed, err := uuid.Parse(stripped)
			assert.NoError(t, err)
			assert.Equal(t, uuid.Version(7), parsed.Version())
		})
	})
}

func TestUUIDConverters(t *testing.T) {
	t.Run("ToUUID and FromUUID roundtrip", func(t *testing.T) {
		prefix := "usr_"
		id, err := Generate(prefix)
		assert.NoError(t, err)

		rawUUID, err := ToUUID(id)
		assert.NoError(t, err)
		assert.Equal(t, uuid.Version(7), rawUUID.Version())

		reconstructed := FromUUID(prefix, rawUUID)
		assert.Equal(t, id, reconstructed)
	})

	t.Run("ToUUID with direct UUID string", func(t *testing.T) {
		uid := uuid.New()
		parsed, err := ToUUID(uid.String())
		assert.NoError(t, err)
		assert.Equal(t, uid, parsed)
	})

	t.Run("ToUUID invalid strings", func(t *testing.T) {
		_, err := ToUUID("invalid-uuid")
		assert.Error(t, err)

		_, err = ToUUID("usr_invalid-uuid")
		assert.Error(t, err)
	})

	t.Run("MustToUUID panic and success", func(t *testing.T) {
		assert.Panics(t, func() {
			MustToUUID("usr_invalid-uuid")
		})

		assert.NotPanics(t, func() {
			prefix := "usr_"
			id, err := Generate(prefix)
			assert.NoError(t, err)
			rawUUID := MustToUUID(id)
			assert.Equal(t, uuid.Version(7), rawUUID.Version())
		})
	})
}
