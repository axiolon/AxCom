// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"
	"ecom-engine/internal/infra/db"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoAdapter struct {
	client    *mongo.Client
	txTimeout time.Duration
}

func NewMongoAdapter(client *mongo.Client, txTimeout time.Duration) *MongoAdapter {
	return &MongoAdapter{client: client, txTimeout: txTimeout}
}

func (a *MongoAdapter) Connect(_ context.Context, _ string) error {
	return nil
}

func (a *MongoAdapter) Close(_ context.Context) error {
	return nil
}

func (a *MongoAdapter) Exec(_ context.Context, _ string, _ ...any) error {
	return fmt.Errorf("Exec not supported in MongoAdapter")
}

func (a *MongoAdapter) ExecResult(_ context.Context, _ string, _ ...any) (db.Result, error) {
	return nil, fmt.Errorf("ExecResult not supported in MongoAdapter")
}

func (a *MongoAdapter) Query(_ context.Context, _ string, _ ...any) (db.Rows, error) {
	return nil, fmt.Errorf("Query not supported in MongoAdapter")
}

type mongoTxKey struct{}

func mongoSessionFromContext(ctx context.Context) *mongo.Session {
	if sess, ok := ctx.Value(mongoTxKey{}).(*mongo.Session); ok {
		return sess
	}
	return nil
}

func (a *MongoAdapter) BeginTx(_ context.Context) (db.Transaction, error) {
	session, err := a.client.StartSession()
	if err != nil {
		return nil, err
	}
	return &MongoTx{adapter: a, session: session}, nil
}

// RunInTx implements db.TransactionManager using real MongoDB sessions and transactions.
func (a *MongoAdapter) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if sess := mongoSessionFromContext(ctx); sess != nil {
		return fn(ctx)
	}

	// Apply transaction timeout if configured.
	if a.txTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.txTimeout)
		defer cancel()
	}

	session, err := a.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx context.Context) (interface{}, error) {
		txCtx := context.WithValue(sessCtx, mongoTxKey{}, session)
		return nil, fn(txCtx)
	})
	return err
}

type MongoTx struct {
	adapter *MongoAdapter
	session *mongo.Session
}

func (t *MongoTx) Connect(_ context.Context, _ string) error {
	return nil
}

func (t *MongoTx) Close(_ context.Context) error {
	return nil
}

func (t *MongoTx) Exec(_ context.Context, _ string, _ ...any) error {
	return fmt.Errorf("Exec not supported in MongoTx")
}

func (t *MongoTx) ExecResult(_ context.Context, _ string, _ ...any) (db.Result, error) {
	return nil, fmt.Errorf("ExecResult not supported in MongoTx")
}

func (t *MongoTx) Query(_ context.Context, _ string, _ ...any) (db.Rows, error) {
	return nil, fmt.Errorf("Query not supported in MongoTx")
}

func (t *MongoTx) BeginTx(_ context.Context) (db.Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (t *MongoTx) Commit(ctx context.Context) error {
	return t.session.CommitTransaction(ctx)
}

func (t *MongoTx) Rollback(ctx context.Context) error {
	return t.session.AbortTransaction(ctx)
}
