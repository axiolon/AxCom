// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	"ecom-engine/internal/core/inventory/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
)

type Service interface {
	GetLowStockReport(ctx context.Context) ([]*domain.StockItem, error)
	ExportInventoryCSV(ctx context.Context) ([]byte, error)
}

type reportsService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &reportsService{
		repo: repo,
	}
}

func (s *reportsService) GetLowStockReport(ctx context.Context) ([]*domain.StockItem, error) {
	logger.InfoCtx(ctx, "Generating low stock report")
	items, err := s.repo.GetLowStockItems(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve low stock items: %v", err)
		return nil, apperrors.NewInternal("failed to generate low stock report", err)
	}
	return items, nil
}

func (s *reportsService) ExportInventoryCSV(ctx context.Context) ([]byte, error) {
	logger.InfoCtx(ctx, "Exporting inventory to CSV")
	items, err := s.repo.GetAllStockItems(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve all stock items for export: %v", err)
		return nil, apperrors.NewInternal("failed to retrieve stock items for CSV export", err)
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write header
	header := []string{"Variant ID", "Location ID", "Quantity", "Low Stock Threshold", "Allow Backorders", "Backorder Limit"}
	if err := writer.Write(header); err != nil {
		return nil, apperrors.NewInternal("failed to write CSV header", err)
	}

	for _, item := range items {
		row := []string{
			item.VariantID,
			item.LocationID,
			strconv.Itoa(item.Quantity),
			strconv.Itoa(item.LowStockThreshold),
			strconv.FormatBool(item.AllowBackorders),
			strconv.Itoa(item.BackorderLimit),
		}
		if err := writer.Write(row); err != nil {
			return nil, apperrors.NewInternal(fmt.Sprintf("failed to write CSV row for variant %s", item.VariantID), err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, apperrors.NewInternal("CSV writer error during flush", err)
	}

	return buf.Bytes(), nil
}
