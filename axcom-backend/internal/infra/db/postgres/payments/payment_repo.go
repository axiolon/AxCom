// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	"errors"

	"ecom-engine/internal/core/payments"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
)

// pgErrUniqueViolation is the PostgreSQL error code for unique constraint violations.
const pgErrUniqueViolation = "23505"

type PostgresPaymentRepository struct {
	db db.Database
}

func NewPostgresPaymentRepository(database db.Database) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{
		db: database,
	}
}

func (r *PostgresPaymentRepository) Create(ctx context.Context, p *payments.Payment) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.Create")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Creating payment for ID: %s", p.ID)

	query := `INSERT INTO payments (id, order_id, customer_id, amount, currency, provider, provider_intent_id, status, idempotency_key, failure_reason, created_at, updated_at, refunded_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	err := r.db.Exec(ctx, query, p.ID, p.OrderID, p.CustomerID, p.Amount, p.Currency, p.Provider, p.ProviderIntentID, string(p.Status), p.IdempotencyKey, p.FailureReason, p.CreatedAt, p.UpdatedAt, p.RefundedAt)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to create payment: %v", err)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return payments.ErrDuplicatePayment
		}
		return err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully created payment for ID: %s", p.ID)
	return nil
}

func (r *PostgresPaymentRepository) GetByID(ctx context.Context, id string) (*payments.Payment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.GetByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding payment by ID: %s", id)

	query := `SELECT id, order_id, customer_id, amount, currency, provider, provider_intent_id, status, idempotency_key, failure_reason, created_at, updated_at, refunded_at
              FROM payments WHERE id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch payment: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		logger.DebugCtx(ctx, "Postgres: Payment not found for ID: %s", id)
		return nil, payments.ErrNotFound
	}

	p, err := r.scanPayment(rows)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found payment for ID: %s", id)
	return p, nil
}

func (r *PostgresPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*payments.Payment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.GetByOrderID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding payment by Order ID: %s", orderID)

	query := `SELECT id, order_id, customer_id, amount, currency, provider, provider_intent_id, status, idempotency_key, failure_reason, created_at, updated_at, refunded_at
              FROM payments WHERE order_id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch payment: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		logger.DebugCtx(ctx, "Postgres: Payment not found for Order ID: %s", orderID)
		return nil, payments.ErrNotFound
	}

	p, err := r.scanPayment(rows)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found payment for Order ID: %s", orderID)
	return p, nil
}

func (r *PostgresPaymentRepository) GetByProviderIntentID(ctx context.Context, provider string, intentID string) (*payments.Payment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.GetByProviderIntentID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding payment by Provider: %s, Intent ID: %s", provider, intentID)

	query := `SELECT id, order_id, customer_id, amount, currency, provider, provider_intent_id, status, idempotency_key, failure_reason, created_at, updated_at, refunded_at
              FROM payments WHERE provider = $1 AND provider_intent_id = $2 LIMIT 1`
	rows, err := r.db.Query(ctx, query, provider, intentID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch payment: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		logger.DebugCtx(ctx, "Postgres: Payment not found for Provider: %s, Intent ID: %s", provider, intentID)
		return nil, payments.ErrNotFound
	}

	p, err := r.scanPayment(rows)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found payment")
	return p, nil
}

func (r *PostgresPaymentRepository) Update(ctx context.Context, p *payments.Payment) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.Update")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating payment ID: %s", p.ID)

	query := `UPDATE payments
              SET order_id = $1, customer_id = $2, amount = $3, currency = $4, provider = $5, provider_intent_id = $6, status = $7, idempotency_key = $8, failure_reason = $9, updated_at = $10, refunded_at = $11
              WHERE id = $12`
	result, err := r.db.ExecResult(ctx, query, p.OrderID, p.CustomerID, p.Amount, p.Currency, p.Provider, p.ProviderIntentID, string(p.Status), p.IdempotencyKey, p.FailureReason, p.UpdatedAt, p.RefundedAt, p.ID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update payment: %v", err)
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if n == 0 {
		logger.DebugCtx(ctx, "Postgres: Payment not found for update, ID: %s", p.ID)
		return payments.ErrNotFound
	}

	logger.DebugCtx(ctx, "Postgres: Successfully updated payment ID: %s", p.ID)
	return nil
}

func (r *PostgresPaymentRepository) ListAll(ctx context.Context, limit, offset int) ([]payments.Payment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.ListAll")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing all payments")

	query := `SELECT id, order_id, customer_id, amount, currency, provider, provider_intent_id, status, idempotency_key, failure_reason, created_at, updated_at, refunded_at
              FROM payments ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to list payments: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var paymentsList []payments.Payment
	for rows.Next() {
		p, err := r.scanPayment(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		paymentsList = append(paymentsList, *p)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return paymentsList, nil
}

func (r *PostgresPaymentRepository) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]payments.Payment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresPaymentRepository.ListByCustomerID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing payments for customer ID: %s", customerID)

	query := `SELECT id, order_id, customer_id, amount, currency, provider, provider_intent_id, status, idempotency_key, failure_reason, created_at, updated_at, refunded_at
              FROM payments WHERE customer_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, customerID, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to list customer payments: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var paymentsList []payments.Payment
	for rows.Next() {
		p, err := r.scanPayment(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		paymentsList = append(paymentsList, *p)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return paymentsList, nil
}

func (r *PostgresPaymentRepository) scanPayment(rows db.Rows) (*payments.Payment, error) {
	var p payments.Payment
	var statusStr string
	err := rows.Scan(&p.ID, &p.OrderID, &p.CustomerID, &p.Amount, &p.Currency, &p.Provider, &p.ProviderIntentID, &statusStr, &p.IdempotencyKey, &p.FailureReason, &p.CreatedAt, &p.UpdatedAt, &p.RefundedAt)
	if err != nil {
		return nil, err
	}
	p.Status = payments.PaymentStatus(statusStr)
	return &p, nil
}
