// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

// Package e2e contains end-to-end tests that run against real MongoDB containers.
//
// Run with:
//
//	go test -tags e2e -v ./tests/e2e/... -timeout 300s
//
// Docker must be available. Each test run gets a fresh ephemeral MongoDB
// container. State is isolated via Truncate() calls at the start of each test.
package e2e

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"ecom-engine/tests/e2e/testutil"
)

// harness is the shared test harness for the entire e2e package.
// It holds one MongoDB container + HTTP server for the full test run.
// Tests isolate state by calling harness.Truncate() at the start.
var harness *testutil.Harness

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	harness, err = testutil.New(ctx, "catalog", "inventory", "cart", "orders", "shipping", "dashboard")
	if err != nil {
		log.Fatalf("e2e: failed to start harness: %v", err)
	}

	code := m.Run()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	harness.Shutdown(shutCtx)

	os.Exit(code)
}
