// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import (
	"context"
	"ecom-engine/internal/events"
	modulesshipping "ecom-engine/internal/modules/shipping"
	"errors"
	"testing"
)

// MockRepository implements Repository
type MockRepository struct {
	shipments map[string]*Shipment
}

func (m *MockRepository) Create(_ context.Context, s *Shipment) error {
	m.shipments[s.ID] = s
	return nil
}

func (m *MockRepository) GetByID(_ context.Context, id string) (*Shipment, error) {
	s, exists := m.shipments[id]
	if !exists {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *MockRepository) GetByOrderID(_ context.Context, orderID string) (*Shipment, error) {
	for _, s := range m.shipments {
		if s.OrderID == orderID {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) Update(_ context.Context, s *Shipment) error {
	m.shipments[s.ID] = s
	return nil
}

func (m *MockRepository) ListAll(_ context.Context, limit, offset int) ([]Shipment, error) {
	var list []Shipment
	for _, s := range m.shipments {
		list = append(list, *s)
	}
	if offset > len(list) {
		return []Shipment{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (m *MockRepository) GetByTrackingNumber(_ context.Context, trackingNumber string) (*Shipment, error) {
	for _, s := range m.shipments {
		if s.TrackingNumber == trackingNumber {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *MockRepository) Delete(_ context.Context, id string) error {
	if _, exists := m.shipments[id]; !exists {
		return errors.New("shipment not found")
	}
	delete(m.shipments, id)
	return nil
}

// MockShippingProvider implements modulesshipping.ShippingProvider
type MockShippingProvider struct {
	name string
	rate float64
	err  error
}

func (m *MockShippingProvider) CalculateRate(_ modulesshipping.Package) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.rate, nil
}

func (m *MockShippingProvider) GetName() string {
	return m.name
}

type MockTxManager struct{}

func (m *MockTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCalculateRates(t *testing.T) {
	repo := &MockRepository{shipments: make(map[string]*Shipment)}
	bus := events.NewLocalEventBus()
	prov1 := &MockShippingProvider{name: "Express Carrier", rate: 15.50}
	prov2 := &MockShippingProvider{name: "Postal Service", rate: 5.25}

	service := NewShipmentService(repo, []modulesshipping.ShippingProvider{prov1, prov2}, bus, &MockTxManager{}, nil)

	t.Run("calculates correct rates", func(t *testing.T) {
		rates, err := service.CalculateRates(context.Background(), RateRequest{Weight: 2.5, Value: 10.0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(rates) != 2 {
			t.Fatalf("expected 2 rates, got %d", len(rates))
		}

		if rates[0].ProviderName != "Express Carrier" || rates[0].Rate != 15.50 {
			t.Errorf("unexpected first rate: %v", rates[0])
		}

		if rates[1].ProviderName != "Postal Service" || rates[1].Rate != 5.25 {
			t.Errorf("unexpected second rate: %v", rates[1])
		}
	})
}

func TestCreateAndUpdateShipment(t *testing.T) {
	repo := &MockRepository{shipments: make(map[string]*Shipment)}
	bus := events.NewLocalEventBus()
	prov := &MockShippingProvider{name: "Express Carrier", rate: 15.50}

	service := NewShipmentService(repo, []modulesshipping.ShippingProvider{prov}, bus, &MockTxManager{}, nil)

	t.Run("creates shipment correctly", func(t *testing.T) {
		s, err := service.CreateShipment(context.Background(), "order_1", "Express Carrier", "track_123", 2.0, 50.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.OrderID != "order_1" {
			t.Errorf("expected OrderID to be order_1, got %s", s.OrderID)
		}
		if s.Carrier != "Express Carrier" {
			t.Errorf("expected carrier to be Express Carrier, got %s", s.Carrier)
		}
		if s.Status != StatusInTransit {
			t.Errorf("expected status to be in_transit, got %s", s.Status)
		}
		if s.ShippingCost != 15.50 {
			t.Errorf("expected shipping cost to be 15.50, got %.2f", s.ShippingCost)
		}
	})

	t.Run("updates shipment status correctly", func(t *testing.T) {
		// Get ID of the created shipment
		var id string
		for k := range repo.shipments {
			id = k
		}

		s, err := service.UpdateShipmentStatus(context.Background(), id, StatusDelivered, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Status != StatusDelivered {
			t.Errorf("expected status to be delivered, got %s", s.Status)
		}
	})

	t.Run("gets shipment by tracking number", func(t *testing.T) {
		s, err := service.GetShipmentByTrackingNumber(context.Background(), "track_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.TrackingNumber != "track_123" {
			t.Errorf("expected tracking number track_123, got %s", s.TrackingNumber)
		}
	})

	t.Run("deletes shipment correctly", func(t *testing.T) {
		var id string
		for k := range repo.shipments {
			id = k
		}

		repo.shipments[id].Status = StatusPending
		err := service.DeleteShipment(context.Background(), id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = repo.GetByID(context.Background(), id)
		if err == nil {
			t.Fatal("expected shipment to be deleted, but it was found")
		}
	})
}

func TestTrackShipment(t *testing.T) {
	repo := &MockRepository{shipments: make(map[string]*Shipment)}
	bus := events.NewLocalEventBus()
	service := NewShipmentService(repo, nil, bus, &MockTxManager{}, nil)

	// Seed a shipment
	shipment := &Shipment{
		ID:             "shpm_123",
		OrderID:        "order_123",
		Carrier:        "USPS",
		TrackingNumber: "TRK12345",
		Status:         StatusInTransit,
	}
	repo.shipments[shipment.ID] = shipment

	t.Run("found", func(t *testing.T) {
		s, err := service.TrackShipment(context.Background(), "TRK12345")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.ID != "shpm_123" {
			t.Errorf("expected shipment ID shpm_123, got %s", s.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := service.TrackShipment(context.Background(), "TRK99999")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err != ErrTrackingNumberNotFound {
			t.Errorf("expected ErrTrackingNumberNotFound, got %v", err)
		}
	})
}
