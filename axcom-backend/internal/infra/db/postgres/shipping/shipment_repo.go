// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ecom-engine/internal/core/shipping"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type dbStatusHistoryEntry struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
}

type PostgresShipmentRepository struct {
	db db.Database
}

func NewPostgresShipmentRepository(database db.Database) *PostgresShipmentRepository {
	return &PostgresShipmentRepository{
		db: database,
	}
}

func (r *PostgresShipmentRepository) Create(ctx context.Context, s *shipping.Shipment) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.Create")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Creating shipment for ID: %s", s.ID)

	historyBytes, err := r.marshalHistory(s.StatusHistory)
	if err != nil {
		span.RecordError(err)
		return err
	}

	query := `INSERT INTO shipments (id, order_id, carrier, tracking_number, status, weight, value, shipping_cost, estimated_delivery_at, status_history, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err = r.db.ExecResult(ctx, query, s.ID, s.OrderID, s.Carrier, s.TrackingNumber, string(s.Status), s.Weight, s.Value, s.ShippingCost, s.EstimatedDeliveryAt, string(historyBytes), s.CreatedAt, s.UpdatedAt)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to create shipment: %v", err)
		return err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully created shipment for ID: %s", s.ID)
	return nil
}

func (r *PostgresShipmentRepository) GetByID(ctx context.Context, id string) (*shipping.Shipment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.GetByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding shipment by ID: %s", id)

	query := `SELECT id, order_id, carrier, tracking_number, status, weight, value, shipping_cost, estimated_delivery_at, status_history, created_at, updated_at 
              FROM shipments WHERE id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch shipment: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err = rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during shipment row iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Shipment not found for ID: %s", id)
		return nil, errors.New("shipment not found")
	}

	s, err := r.scanShipment(rows)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found shipment for ID: %s", id)
	return s, nil
}

func (r *PostgresShipmentRepository) GetByOrderID(ctx context.Context, orderID string) (*shipping.Shipment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.GetByOrderID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding shipment by Order ID: %s", orderID)

	query := `SELECT id, order_id, carrier, tracking_number, status, weight, value, shipping_cost, estimated_delivery_at, status_history, created_at, updated_at 
              FROM shipments WHERE order_id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch shipment: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err = rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during shipment row iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Shipment not found for Order ID: %s", orderID)
		return nil, errors.New("shipment not found")
	}

	s, err := r.scanShipment(rows)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found shipment for Order ID: %s", orderID)
	return s, nil
}

func (r *PostgresShipmentRepository) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*shipping.Shipment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.GetByTrackingNumber")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding shipment by Tracking Number: %s", trackingNumber)

	query := `SELECT id, order_id, carrier, tracking_number, status, weight, value, shipping_cost, estimated_delivery_at, status_history, created_at, updated_at 
              FROM shipments WHERE tracking_number = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, trackingNumber)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to fetch shipment: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err = rows.Err(); err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Error during shipment row iteration: %v", err)
			return nil, err
		}
		logger.DebugCtx(ctx, "Postgres: Shipment not found for Tracking Number: %s", trackingNumber)
		return nil, errors.New("shipment not found")
	}

	s, err := r.scanShipment(rows)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	logger.DebugCtx(ctx, "Postgres: Successfully found shipment")
	return s, nil
}

func (r *PostgresShipmentRepository) Update(ctx context.Context, s *shipping.Shipment) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.Update")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating shipment ID: %s", s.ID)

	historyBytes, err := r.marshalHistory(s.StatusHistory)
	if err != nil {
		span.RecordError(err)
		return err
	}

	query := `UPDATE shipments 
              SET order_id = $1, carrier = $2, tracking_number = $3, status = $4, weight = $5, value = $6, shipping_cost = $7, estimated_delivery_at = $8, status_history = $9, updated_at = $10 
              WHERE id = $11`
	res, err := r.db.ExecResult(ctx, query, s.OrderID, s.Carrier, s.TrackingNumber, string(s.Status), s.Weight, s.Value, s.ShippingCost, s.EstimatedDeliveryAt, string(historyBytes), s.UpdatedAt, s.ID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update shipment: %v", err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		logger.DebugCtx(ctx, "Postgres: Update failed, shipment ID not found: %s", s.ID)
		return errors.New("shipment not found")
	}

	logger.DebugCtx(ctx, "Postgres: Successfully updated shipment ID: %s", s.ID)
	return nil
}

func (r *PostgresShipmentRepository) ListAll(ctx context.Context, limit, offset int) ([]shipping.Shipment, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.ListAll")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing all shipments")

	query := `SELECT id, order_id, carrier, tracking_number, status, weight, value, shipping_cost, estimated_delivery_at, status_history, created_at, updated_at 
              FROM shipments ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to list shipments: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var shipmentsList []shipping.Shipment
	for rows.Next() {
		s, err := r.scanShipment(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		shipmentsList = append(shipmentsList, *s)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Error iterating shipments: %v", err)
		return nil, err
	}

	return shipmentsList, nil
}

func (r *PostgresShipmentRepository) Delete(ctx context.Context, id string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresShipmentRepository.Delete")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Deleting shipment ID: %s", id)

	query := "DELETE FROM shipments WHERE id = $1"
	res, err := r.db.ExecResult(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to delete shipment: %v", err)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		logger.DebugCtx(ctx, "Postgres: Delete failed, shipment ID not found: %s", id)
		return errors.New("shipment not found")
	}

	logger.DebugCtx(ctx, "Postgres: Successfully deleted shipment ID: %s", id)
	return nil
}

func (r *PostgresShipmentRepository) marshalHistory(history []shipping.StatusHistoryEntry) ([]byte, error) {
	dbHistory := make([]dbStatusHistoryEntry, len(history))
	for i, h := range history {
		dbHistory[i] = dbStatusHistoryEntry{
			Status:    string(h.Status),
			Timestamp: h.Timestamp,
			Actor:     h.Actor,
		}
	}
	return json.Marshal(dbHistory)
}

func (r *PostgresShipmentRepository) unmarshalHistory(historyStr string) ([]shipping.StatusHistoryEntry, error) {
	if historyStr == "" || historyStr == "null" {
		return []shipping.StatusHistoryEntry{}, nil
	}
	var dbHistory []dbStatusHistoryEntry
	if err := json.Unmarshal([]byte(historyStr), &dbHistory); err != nil {
		return nil, err
	}
	history := make([]shipping.StatusHistoryEntry, len(dbHistory))
	for i, h := range dbHistory {
		history[i] = shipping.StatusHistoryEntry{
			Status:    shipping.ShipmentStatus(h.Status),
			Timestamp: h.Timestamp,
			Actor:     h.Actor,
		}
	}
	return history, nil
}

func (r *PostgresShipmentRepository) scanShipment(rows db.Rows) (*shipping.Shipment, error) {
	var s shipping.Shipment
	var statusStr string
	var historyStr string
	err := rows.Scan(&s.ID, &s.OrderID, &s.Carrier, &s.TrackingNumber, &statusStr, &s.Weight, &s.Value, &s.ShippingCost, &s.EstimatedDeliveryAt, &historyStr, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.Status = shipping.ShipmentStatus(statusStr)
	history, err := r.unmarshalHistory(historyStr)
	if err != nil {
		return nil, err
	}
	s.StatusHistory = history
	return &s, nil
}
