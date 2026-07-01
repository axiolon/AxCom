// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockShutdownModule struct {
	name       string
	onShutdown func()
}

func (m *mockShutdownModule) Name() string                            { return m.name }
func (m *mockShutdownModule) Requires() []string                      { return nil }
func (m *mockShutdownModule) BasePaths() []string                     { return nil }
func (m *mockShutdownModule) Init(_ *Container) error                 { return nil }
func (m *mockShutdownModule) RegisterRoutes(_, _, _ *gin.RouterGroup) {}
func (m *mockShutdownModule) Shutdown(_ context.Context) error {
	m.onShutdown()
	return nil
}

type mockDB struct {
	closed bool
}

func (m *mockDB) Ping(_ context.Context) error { return nil }
func (m *mockDB) Close() error {
	m.closed = true
	return nil
}

func TestEngine_Shutdown(t *testing.T) {
	t.Parallel()

	shutdownOrder := make([]string, 0)
	mCatalog := &mockShutdownModule{
		name: "catalog",
		onShutdown: func() {
			shutdownOrder = append(shutdownOrder, "catalog")
		},
	}
	mCart := &mockShutdownModule{
		name: "cart",
		onShutdown: func() {
			shutdownOrder = append(shutdownOrder, "cart")
		},
	}

	dbConn := &mockDB{}

	eng := &Engine{
		activeModules: []Module{mCatalog, mCart}, // init order: catalog first, then cart
		DBConn:        dbConn,
	}

	err := eng.Shutdown(context.Background())
	assert.NoError(t, err)

	// Shutdown must tear down modules in reverse chronological order: cart, then catalog
	assert.Len(t, shutdownOrder, 2)
	assert.Equal(t, "cart", shutdownOrder[0])
	assert.Equal(t, "catalog", shutdownOrder[1])

	// Database connection must be closed
	assert.True(t, dbConn.closed)
}

func TestNewEngine_BootstrapFailures(t *testing.T) {
	t.Parallel()

	t.Run("unsupported database type", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.DB.Type = "invalid_db"

		eng, err := NewEngine(cfg, nil, nil)
		assert.Nil(t, eng)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported database type "invalid_db"`)
	})

	t.Run("unsupported cache type", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		cfg.Cache.Type = "invalid_cache"

		eng, err := NewEngine(cfg, nil, nil)
		assert.Nil(t, eng)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `unsupported cache type "invalid_cache"`)
	})

}

type mockCircularModule struct {
	name     string
	requires []string
}

func (m mockCircularModule) Name() string                            { return m.name }
func (m mockCircularModule) Requires() []string                      { return m.requires }
func (m mockCircularModule) BasePaths() []string                     { return nil }
func (m mockCircularModule) Init(_ *Container) error                 { return nil }
func (m mockCircularModule) RegisterRoutes(_, _, _ *gin.RouterGroup) {}
func (m mockCircularModule) Shutdown(_ context.Context) error        { return nil }

func TestNewEngine_SortingFailure(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.DB.Type = "postgres"
	cfg.DB.ConnectionString = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	mA := mockCircularModule{name: "A", requires: []string{"B"}}
	mB := mockCircularModule{name: "B", requires: []string{"A"}}

	eng, err := NewEngine(cfg, []Module{mA, mB}, nil)
	assert.Nil(t, eng)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module wiring: circular dependency detected")
}
