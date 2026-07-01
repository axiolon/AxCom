// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import "time"

type ShipmentStatus string

const (
	StatusPending   ShipmentStatus = "pending"
	StatusInTransit ShipmentStatus = "in_transit"
	StatusDelivered ShipmentStatus = "delivered"
	StatusReturned  ShipmentStatus = "returned"
)

type StatusHistoryEntry struct {
	Status    ShipmentStatus `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	Actor     string         `json:"actor"`
}

// Shipment represents the shipping details and tracking for an order.
type Shipment struct {
	ID                  string               `json:"id"`
	OrderID             string               `json:"order_id"`
	Carrier             string               `json:"carrier"`
	TrackingNumber      string               `json:"tracking_number"`
	Status              ShipmentStatus       `json:"status"`
	Weight              float64              `json:"weight"`
	Value               float64              `json:"value"`
	ShippingCost        float64              `json:"shipping_cost"`
	EstimatedDeliveryAt *time.Time           `json:"estimated_delivery_at,omitempty"`
	StatusHistory       []StatusHistoryEntry `json:"status_history,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

// RateRequest is the payload used to request shipping rates.
type RateRequest struct {
	Weight float64 `json:"weight" binding:"required"`
	Value  float64 `json:"value"`
}

// RateResponse is the payload returned containing provider name and calculated rate.
type RateResponse struct {
	ProviderName string  `json:"provider_name"`
	Rate         float64 `json:"rate"`
}

// TrackingResponse is the safe, public DTO used for tracking number lookup.
type TrackingResponse struct {
	TrackingNumber      string         `json:"tracking_number"`
	Carrier             string         `json:"carrier"`
	Status              ShipmentStatus `json:"status"`
	EstimatedDeliveryAt *time.Time     `json:"estimated_delivery_at,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}
