// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package shipping provides core shipping management, rating, and tracking services.
package shipping

import (
	"context"
	"ecom-engine/internal/events"
	"ecom-engine/internal/infra/db"
	modulesshipping "ecom-engine/internal/modules/shipping"
	"errors"
	"fmt"
	"time"

	"ecom-engine/pkg/logger"

	"ecom-engine/pkg/idgen"
)

var (
	// ErrShipmentNotFound is returned when a requested shipment cannot be found.
	ErrShipmentNotFound = errors.New("shipment not found")

	// ErrTrackingNumberNotFound is returned when a shipment with the specified tracking number is not found.
	ErrTrackingNumberNotFound = errors.New("tracking number not found")
)

// Service defines the business logic for shipping rate calculations and shipment tracking.
type Service interface {
	// CalculateRates queries registered shipping providers to fetch estimated costs.
	CalculateRates(ctx context.Context, req RateRequest) ([]RateResponse, error)

	// CreateShipment generates a new shipment record for an order and persists it.
	CreateShipment(ctx context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*Shipment, error)

	// UpdateShipmentStatus updates tracking details and status for a specific shipment.
	UpdateShipmentStatus(ctx context.Context, id string, status ShipmentStatus, trackingNumber string) (*Shipment, error)

	// GetShipmentByOrderID retrieves the shipment details corresponding to the given order ID.
	GetShipmentByOrderID(ctx context.Context, orderID string) (*Shipment, error)

	// ListAllShipments returns a list of all shipment records in the system.
	ListAllShipments(ctx context.Context, limit, offset int) ([]Shipment, error)

	// GetShipmentByTrackingNumber retrieves shipment details by carrier tracking number.
	GetShipmentByTrackingNumber(ctx context.Context, trackingNumber string) (*Shipment, error)

	// TrackShipment retrieves shipment details for public lookup by tracking number.
	TrackShipment(ctx context.Context, trackingNumber string) (*Shipment, error)

	// DeleteShipment cancels/removes a shipment record.
	DeleteShipment(ctx context.Context, id string) error
}

type shipmentService struct {
	repo      Repository
	providers []modulesshipping.ShippingProvider
	eventBus  events.EventBus
	txManager db.TransactionManager
	outbox    events.OutboxRepository
}

// NewShipmentService initializes and returns a new shipping Service instance.
func NewShipmentService(
	repo Repository,
	providers []modulesshipping.ShippingProvider,
	eventBus events.EventBus,
	txManager db.TransactionManager,
	outbox events.OutboxRepository,
) Service {
	return &shipmentService{
		repo:      repo,
		providers: providers,
		eventBus:  eventBus,
		txManager: txManager,
		outbox:    outbox,
	}
}

// CalculateRates implements the Service interface.
func (s *shipmentService) CalculateRates(ctx context.Context, req RateRequest) ([]RateResponse, error) {
	logger.InfoCtx(ctx, "Calculating shipping rates for package: weight=%.2f, value=%.2f", req.Weight, req.Value)

	pkg := modulesshipping.Package{
		Weight: req.Weight,
		Value:  req.Value,
	}

	var responses []RateResponse
	var errs []error
	for _, provider := range s.providers {
		rate, err := provider.CalculateRate(pkg)
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to calculate rate for provider %s: %v", provider.GetName(), err)
			errs = append(errs, err)
			continue
		}
		responses = append(responses, RateResponse{
			ProviderName: provider.GetName(),
			Rate:         rate,
		})
	}

	if len(s.providers) > 0 && len(responses) == 0 {
		return nil, fmt.Errorf("all providers failed to calculate rate: %v", errs)
	}

	return responses, nil
}

// CreateShipment implements the Service interface.
func (s *shipmentService) CreateShipment(ctx context.Context, orderID string, carrier string, trackingNumber string, weight float64, value float64) (*Shipment, error) {
	logger.InfoCtx(ctx, "Creating shipment for order %s, carrier %s", orderID, carrier)

	// Determine cost if one of the providers matches
	cost := 0.0
	pkg := modulesshipping.Package{
		Weight: weight,
		Value:  value,
	}

	for _, provider := range s.providers {
		if provider.GetName() == carrier {
			calculated, err := provider.CalculateRate(pkg)
			if err != nil {
				logger.ErrorCtx(ctx, "Failed to calculate shipping rate from carrier %s: %v", carrier, err)
				return nil, fmt.Errorf("failed to calculate shipping rate: %w", err)
			}
			cost = calculated
			break
		}
	}

	status := StatusPending
	if trackingNumber != "" {
		status = StatusInTransit
	}

	// Generate full UUID to prevent collisions
	uid, err := idgen.Generate("shpm_")
	if err != nil {
		return nil, fmt.Errorf("failed to generate shipment ID: %w", err)
	}

	est := time.Now().Add(3 * 24 * time.Hour)

	shipment := &Shipment{
		ID:                  uid,
		OrderID:             orderID,
		Carrier:             carrier,
		TrackingNumber:      trackingNumber,
		Status:              status,
		Weight:              weight,
		Value:               value,
		ShippingCost:        cost,
		EstimatedDeliveryAt: &est,
		StatusHistory: []StatusHistoryEntry{
			{Status: status, Timestamp: time.Now(), Actor: "system"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var shippedEvt *events.Event
	if status == StatusInTransit {
		e := events.NewEventFromCtx(ctx, events.OrderShippedTopic, "shipping",
			events.OrderShippedEventPayload{
				OrderID:        orderID,
				TrackingNumber: trackingNumber,
				Carrier:        carrier,
			})
		shippedEvt = &e
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if existing, innerErr := s.repo.GetByOrderID(txCtx, orderID); innerErr == nil && existing != nil {
			logger.ErrorCtx(txCtx, "Shipment already exists for order %s: ID %s", orderID, existing.ID)
			return errors.New("shipment already exists for this order")
		}
		if innerErr := s.repo.Create(txCtx, shipment); innerErr != nil {
			logger.ErrorCtx(txCtx, "Failed to persist shipment: %v", innerErr)
			return innerErr
		}
		if shippedEvt != nil && s.outbox != nil {
			return s.outbox.Store(txCtx, *shippedEvt)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if shippedEvt != nil && s.outbox == nil {
		s.eventBus.Publish(*shippedEvt)
	}

	return shipment, nil
}

// UpdateShipmentStatus implements the Service interface.
func (s *shipmentService) UpdateShipmentStatus(ctx context.Context, id string, status ShipmentStatus, trackingNumber string) (*Shipment, error) {
	logger.InfoCtx(ctx, "Updating shipment %s status to %s, tracking=%s", id, status, trackingNumber)

	shipment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve shipment: %v", err)
		return nil, ErrShipmentNotFound
	}

	oldStatus := shipment.Status

	// Status transition validation state machine:
	// pending -> in_transit -> delivered / returned
	if oldStatus == StatusDelivered || oldStatus == StatusReturned {
		return nil, fmt.Errorf("cannot update shipment from terminal status %s", oldStatus)
	}
	if oldStatus == StatusInTransit && status == StatusPending {
		return nil, fmt.Errorf("cannot revert status from %s to %s", oldStatus, status)
	}

	shipment.Status = status
	if trackingNumber != "" {
		shipment.TrackingNumber = trackingNumber
	}
	shipment.StatusHistory = append(shipment.StatusHistory, StatusHistoryEntry{
		Status:    status,
		Timestamp: time.Now(),
		Actor:     "admin",
	})
	shipment.UpdatedAt = time.Now()

	var shippedEvt *events.Event
	if oldStatus == StatusPending && status == StatusInTransit {
		e := events.NewEventFromCtx(ctx, events.OrderShippedTopic, "shipping",
			events.OrderShippedEventPayload{
				OrderID:        shipment.OrderID,
				TrackingNumber: shipment.TrackingNumber,
				Carrier:        shipment.Carrier,
			})
		shippedEvt = &e
	}

	if shippedEvt != nil && s.outbox != nil && s.txManager != nil {
		if err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if err := s.repo.Update(txCtx, shipment); err != nil {
				return err
			}
			return s.outbox.Store(txCtx, *shippedEvt)
		}); err != nil {
			logger.ErrorCtx(ctx, "Failed to update shipment record: %v", err)
			return nil, err
		}
	} else {
		if err := s.repo.Update(ctx, shipment); err != nil {
			logger.ErrorCtx(ctx, "Failed to update shipment record: %v", err)
			return nil, err
		}
		if shippedEvt != nil {
			s.eventBus.Publish(*shippedEvt)
		}
	}

	return shipment, nil
}

// GetShipmentByOrderID implements the Service interface.
func (s *shipmentService) GetShipmentByOrderID(ctx context.Context, orderID string) (*Shipment, error) {
	shipment, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, ErrShipmentNotFound
	}
	return shipment, nil
}

// ListAllShipments implements the Service interface.
func (s *shipmentService) ListAllShipments(ctx context.Context, limit, offset int) ([]Shipment, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAll(ctx, limit, offset)
}

// GetShipmentByTrackingNumber implements the Service interface.
func (s *shipmentService) GetShipmentByTrackingNumber(ctx context.Context, trackingNumber string) (*Shipment, error) {
	return s.repo.GetByTrackingNumber(ctx, trackingNumber)
}

// TrackShipment implements the Service interface.
func (s *shipmentService) TrackShipment(ctx context.Context, trackingNumber string) (*Shipment, error) {
	shipment, err := s.repo.GetByTrackingNumber(ctx, trackingNumber)
	if err != nil {
		return nil, ErrTrackingNumberNotFound
	}
	return shipment, nil
}

// DeleteShipment implements the Service interface.
func (s *shipmentService) DeleteShipment(ctx context.Context, id string) error {
	shipment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrShipmentNotFound
	}
	if shipment.Status != StatusPending {
		return fmt.Errorf("cannot delete shipment in status %s: only pending shipments can be deleted", shipment.Status)
	}
	return s.repo.Delete(ctx, id)
}
