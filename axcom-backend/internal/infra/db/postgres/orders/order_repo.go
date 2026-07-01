// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package orders

import (
	"context"
	"errors"
	"time"

	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresOrderRepository struct {
	db db.Database
}

func NewPostgresOrderRepository(database db.Database) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		db: database,
	}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, o *orders.Order) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.Create")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Inserting order for ID: %s", o.ID)

	queryOrder := `INSERT INTO orders (id, customer_id, customer_name, customer_email, customer_contact_number, total, status, created_at) 
                   VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	err := r.db.Exec(ctx, queryOrder, o.ID, o.CustomerID, o.CustomerSnapshot.Name, o.CustomerSnapshot.Email, o.CustomerSnapshot.ContactNumber, o.Total, string(o.Status), o.CreatedAt)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to insert order: %v", err)
		return err
	}

	for _, item := range o.Items {
		queryItem := "INSERT INTO order_items (order_id, variant_id, quantity, price) VALUES ($1, $2, $3, $4)"
		err = r.db.Exec(ctx, queryItem, o.ID, item.VariantID, item.Quantity, item.Price)
		if err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Failed to insert order item: %v", err)
			return err
		}
	}

	logger.DebugCtx(ctx, "Postgres: Successfully inserted order and items for ID: %s", o.ID)
	return nil
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*orders.Order, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.GetByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding order by ID: %s", id)

	queryOrder := `SELECT id, customer_id, customer_name, customer_email, customer_contact_number, total, status, created_at 
                   FROM orders WHERE id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, queryOrder, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query order: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		logger.DebugCtx(ctx, "Postgres: Order not found for ID: %s", id)
		return nil, errors.New("order not found")
	}

	var o orders.Order
	var statusStr string
	if err = rows.Scan(&o.ID, &o.CustomerID, &o.CustomerSnapshot.Name, &o.CustomerSnapshot.Email, &o.CustomerSnapshot.ContactNumber, &o.Total, &statusStr, &o.CreatedAt); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to scan order: %v", err)
		return nil, err
	}
	o.Status = orders.OrderStatus(statusStr)

	// Fetch items
	items, err := r.fetchOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items

	logger.DebugCtx(ctx, "Postgres: Successfully found order for ID: %s", id)
	return &o, nil
}

func (r *PostgresOrderRepository) Update(ctx context.Context, o *orders.Order) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.Update")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating order ID: %s", o.ID)

	queryOrder := `UPDATE orders 
                   SET customer_id = $1, customer_name = $2, customer_email = $3, customer_contact_number = $4, total = $5, status = $6 
                   WHERE id = $7`
	err := r.db.Exec(ctx, queryOrder, o.CustomerID, o.CustomerSnapshot.Name, o.CustomerSnapshot.Email, o.CustomerSnapshot.ContactNumber, o.Total, string(o.Status), o.ID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update order: %v", err)
		return err
	}

	// Delete and re-insert items to reflect replace behavior
	queryDelItems := "DELETE FROM order_items WHERE order_id = $1"
	err = r.db.Exec(ctx, queryDelItems, o.ID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to delete order items: %v", err)
		return err
	}

	for _, item := range o.Items {
		queryItem := "INSERT INTO order_items (order_id, variant_id, quantity, price) VALUES ($1, $2, $3, $4)"
		err = r.db.Exec(ctx, queryItem, o.ID, item.VariantID, item.Quantity, item.Price)
		if err != nil {
			span.RecordError(err)
			logger.ErrorCtx(ctx, "Postgres: Failed to insert order item: %v", err)
			return err
		}
	}

	logger.DebugCtx(ctx, "Postgres: Successfully updated order for ID: %s", o.ID)
	return nil
}

func (r *PostgresOrderRepository) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]orders.Order, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.ListByCustomerID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing orders for customer ID: %s", customerID)

	query := `SELECT id, customer_id, customer_name, customer_email, customer_contact_number, total, status, created_at 
              FROM orders WHERE customer_id = $1 LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, customerID, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to list orders: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ordersList []orders.Order
	for rows.Next() {
		var o orders.Order
		var statusStr string
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.CustomerSnapshot.Name, &o.CustomerSnapshot.Email, &o.CustomerSnapshot.ContactNumber, &o.Total, &statusStr, &o.CreatedAt); err != nil {
			span.RecordError(err)
			return nil, err
		}
		o.Status = orders.OrderStatus(statusStr)
		ordersList = append(ordersList, o)
	}

	for i := range ordersList {
		items, err := r.fetchOrderItems(ctx, ordersList[i].ID)
		if err != nil {
			return nil, err
		}
		ordersList[i].Items = items
	}

	return ordersList, nil
}

func (r *PostgresOrderRepository) ListAll(ctx context.Context, limit, offset int) ([]orders.Order, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.ListAll")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Listing all orders")

	query := `SELECT id, customer_id, customer_name, customer_email, customer_contact_number, total, status, created_at 
              FROM orders LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to list orders: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ordersList []orders.Order
	for rows.Next() {
		var o orders.Order
		var statusStr string
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.CustomerSnapshot.Name, &o.CustomerSnapshot.Email, &o.CustomerSnapshot.ContactNumber, &o.Total, &statusStr, &o.CreatedAt); err != nil {
			span.RecordError(err)
			return nil, err
		}
		o.Status = orders.OrderStatus(statusStr)
		ordersList = append(ordersList, o)
	}

	for i := range ordersList {
		items, err := r.fetchOrderItems(ctx, ordersList[i].ID)
		if err != nil {
			return nil, err
		}
		ordersList[i].Items = items
	}

	return ordersList, nil
}

func (r *PostgresOrderRepository) fetchOrderItems(ctx context.Context, orderID string) ([]orders.OrderItem, error) {
	queryItems := "SELECT variant_id, quantity, price FROM order_items WHERE order_id = $1"
	rows, err := r.db.Query(ctx, queryItems, orderID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []orders.OrderItem
	for rows.Next() {
		var item orders.OrderItem
		if err := rows.Scan(&item.VariantID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *PostgresOrderRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.CountByStatus")
	defer span.End()

	rows, err := r.db.Query(ctx, "SELECT status, COUNT(*) FROM orders GROUP BY status")
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (r *PostgresOrderRepository) SumRevenue(ctx context.Context, since time.Time) (float64, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.SumRevenue")
	defer span.End()

	var query string
	var args []interface{}
	if since.IsZero() {
		query = "SELECT COALESCE(SUM(total), 0) FROM orders"
	} else {
		query = "SELECT COALESCE(SUM(total), 0) FROM orders WHERE created_at >= $1"
		args = append(args, since)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}
	defer func() { _ = rows.Close() }()

	if rows.Next() {
		var total float64
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
		return total, rows.Err()
	}
	return 0, rows.Err()
}

func (r *PostgresOrderRepository) RevenueByDay(ctx context.Context, days int) ([]orders.DailyRevenue, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.RevenueByDay")
	defer span.End()

	since := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	query := `SELECT TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
	                 COALESCE(SUM(total), 0) AS revenue
	          FROM orders
	          WHERE created_at >= $1
	          GROUP BY day
	          ORDER BY day ASC`

	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []orders.DailyRevenue
	for rows.Next() {
		var d orders.DailyRevenue
		if err := rows.Scan(&d.Date, &d.Revenue); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *PostgresOrderRepository) TopProducts(ctx context.Context, n int) ([]orders.ProductSales, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresOrderRepository.TopProducts")
	defer span.End()

	query := `SELECT variant_id, SUM(quantity) AS total_sold
	          FROM order_items
	          GROUP BY variant_id
	          ORDER BY total_sold DESC
	          LIMIT $1`

	rows, err := r.db.Query(ctx, query, n)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []orders.ProductSales
	for rows.Next() {
		var p orders.ProductSales
		if err := rows.Scan(&p.VariantID, &p.TotalSold); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
