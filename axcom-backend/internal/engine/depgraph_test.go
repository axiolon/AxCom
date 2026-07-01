// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockModule struct {
	name      string
	requires  []string
	basePaths []string
}

func (m *mockModule) Name() string                            { return m.name }
func (m *mockModule) Requires() []string                      { return m.requires }
func (m *mockModule) BasePaths() []string                     { return m.basePaths }
func (m *mockModule) Init(_ *Container) error                 { return nil }
func (m *mockModule) RegisterRoutes(_, _, _ *gin.RouterGroup) {}
func (m *mockModule) Shutdown(_ context.Context) error        { return nil }

func TestValidateAndSort(t *testing.T) {
	t.Parallel()

	t.Run("happy path - independent modules", func(t *testing.T) {
		t.Parallel()
		m1 := &mockModule{name: "catalog"}
		m2 := &mockModule{name: "shipping"}

		sorted, err := validateAndSort([]Module{m1, m2})
		assert.NoError(t, err)
		assert.Len(t, sorted, 2)
		// Since there are no dependencies, they can be in any order, but order is deterministic based on Kahn's queue.
		assert.Contains(t, []string{"catalog", "shipping"}, sorted[0].Name())
		assert.Contains(t, []string{"catalog", "shipping"}, sorted[1].Name())
	})

	t.Run("happy path - sequential dependency", func(t *testing.T) {
		t.Parallel()
		// cart requires catalog
		mCatalog := &mockModule{name: "catalog"}
		mCart := &mockModule{name: "cart", requires: []string{"catalog"}}

		// We pass them out of order: cart, then catalog
		sorted, err := validateAndSort([]Module{mCart, mCatalog})
		assert.NoError(t, err)
		assert.Len(t, sorted, 2)
		// Catalog must come first
		assert.Equal(t, "catalog", sorted[0].Name())
		assert.Equal(t, "cart", sorted[1].Name())
	})

	t.Run("happy path - transitive dependency", func(t *testing.T) {
		t.Parallel()
		// orders requires cart, cart requires catalog
		mCatalog := &mockModule{name: "catalog"}
		mCart := &mockModule{name: "cart", requires: []string{"catalog"}}
		mOrders := &mockModule{name: "orders", requires: []string{"cart"}}

		sorted, err := validateAndSort([]Module{mOrders, mCatalog, mCart})
		assert.NoError(t, err)
		assert.Len(t, sorted, 3)
		assert.Equal(t, "catalog", sorted[0].Name())
		assert.Equal(t, "cart", sorted[1].Name())
		assert.Equal(t, "orders", sorted[2].Name())
	})

	t.Run("missing dependency error", func(t *testing.T) {
		t.Parallel()
		// cart requires catalog, but catalog is disabled / missing
		mCart := &mockModule{name: "cart", requires: []string{"catalog"}}

		sorted, err := validateAndSort([]Module{mCart})
		assert.Nil(t, sorted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `requires "catalog" but "catalog" is either disabled or not registered`)
	})

	t.Run("circular dependency direct cycle error", func(t *testing.T) {
		t.Parallel()
		// A requires B, B requires A
		mA := &mockModule{name: "A", requires: []string{"B"}}
		mB := &mockModule{name: "B", requires: []string{"A"}}

		sorted, err := validateAndSort([]Module{mA, mB})
		assert.Nil(t, sorted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency detected among modules")
	})

	t.Run("circular dependency multi-node cycle error", func(t *testing.T) {
		t.Parallel()
		// A requires B, B requires C, C requires A
		mA := &mockModule{name: "A", requires: []string{"B"}}
		mB := &mockModule{name: "B", requires: []string{"C"}}
		mC := &mockModule{name: "C", requires: []string{"A"}}

		sorted, err := validateAndSort([]Module{mA, mB, mC})
		assert.Nil(t, sorted)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency detected among modules")
		assert.Contains(t, err.Error(), "A")
		assert.Contains(t, err.Error(), "B")
		assert.Contains(t, err.Error(), "C")
		assert.Contains(t, err.Error(), "->")
	})
}
