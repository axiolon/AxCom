// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package shipping

import (
	"context"
	"errors"
	"time"

	"ecom-engine/internal/core/shipping"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoStatusHistoryEntry struct {
	Status    string    `bson:"status"`
	Timestamp time.Time `bson:"timestamp"`
	Actor     string    `bson:"actor"`
}

type MongoShipment struct {
	ID                  string                    `bson:"_id"`
	OrderID             string                    `bson:"order_id"`
	Carrier             string                    `bson:"carrier"`
	TrackingNumber      string                    `bson:"tracking_number"`
	Status              string                    `bson:"status"`
	Weight              float64                   `bson:"weight"`
	Value               float64                   `bson:"value"`
	ShippingCost        float64                   `bson:"shipping_cost"`
	EstimatedDeliveryAt *time.Time                `bson:"estimated_delivery_at,omitempty"`
	StatusHistory       []MongoStatusHistoryEntry `bson:"status_history,omitempty"`
	CreatedAt           time.Time                 `bson:"created_at"`
	UpdatedAt           time.Time                 `bson:"updated_at"`
}

type MongoShipmentRepository struct {
	collection *mongo.Collection
}

func NewMongoShipmentRepository(db *mongo.Database) *MongoShipmentRepository {
	return &MongoShipmentRepository{
		collection: db.Collection("shipments"),
	}
}

func toShipmentDoc(s *shipping.Shipment) *MongoShipment {
	if s == nil {
		return nil
	}
	history := make([]MongoStatusHistoryEntry, len(s.StatusHistory))
	for i, h := range s.StatusHistory {
		history[i] = MongoStatusHistoryEntry{
			Status:    string(h.Status),
			Timestamp: h.Timestamp,
			Actor:     h.Actor,
		}
	}
	return &MongoShipment{
		ID:                  s.ID,
		OrderID:             s.OrderID,
		Carrier:             s.Carrier,
		TrackingNumber:      s.TrackingNumber,
		Status:              string(s.Status),
		Weight:              s.Weight,
		Value:               s.Value,
		ShippingCost:        s.ShippingCost,
		EstimatedDeliveryAt: s.EstimatedDeliveryAt,
		StatusHistory:       history,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

func toDomainShipment(doc *MongoShipment) *shipping.Shipment {
	if doc == nil {
		return nil
	}
	history := make([]shipping.StatusHistoryEntry, len(doc.StatusHistory))
	for i, h := range doc.StatusHistory {
		history[i] = shipping.StatusHistoryEntry{
			Status:    shipping.ShipmentStatus(h.Status),
			Timestamp: h.Timestamp,
			Actor:     h.Actor,
		}
	}
	return &shipping.Shipment{
		ID:                  doc.ID,
		OrderID:             doc.OrderID,
		Carrier:             doc.Carrier,
		TrackingNumber:      doc.TrackingNumber,
		Status:              shipping.ShipmentStatus(doc.Status),
		Weight:              doc.Weight,
		Value:               doc.Value,
		ShippingCost:        doc.ShippingCost,
		EstimatedDeliveryAt: doc.EstimatedDeliveryAt,
		StatusHistory:       history,
		CreatedAt:           doc.CreatedAt,
		UpdatedAt:           doc.UpdatedAt,
	}
}

func (r *MongoShipmentRepository) Create(ctx context.Context, s *shipping.Shipment) error {
	doc := toShipmentDoc(s)
	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

func (r *MongoShipmentRepository) GetByID(ctx context.Context, id string) (*shipping.Shipment, error) {
	var doc MongoShipment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return toDomainShipment(&doc), nil
}

func (r *MongoShipmentRepository) GetByOrderID(ctx context.Context, orderID string) (*shipping.Shipment, error) {
	var doc MongoShipment
	err := r.collection.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return toDomainShipment(&doc), nil
}

func (r *MongoShipmentRepository) Update(ctx context.Context, s *shipping.Shipment) error {
	doc := toShipmentDoc(s)
	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": s.ID}, doc)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("shipment not found")
	}
	return nil
}

func (r *MongoShipmentRepository) ListAll(ctx context.Context, limit, offset int) ([]shipping.Shipment, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []MongoShipment
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	shipmentsList := make([]shipping.Shipment, len(docs))
	for i, doc := range docs {
		shipmentsList[i] = *toDomainShipment(&doc)
	}
	return shipmentsList, nil
}

func (r *MongoShipmentRepository) GetByTrackingNumber(ctx context.Context, trackingNumber string) (*shipping.Shipment, error) {
	var doc MongoShipment
	err := r.collection.FindOne(ctx, bson.M{"tracking_number": trackingNumber}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("shipment not found")
		}
		return nil, err
	}
	return toDomainShipment(&doc), nil
}

func (r *MongoShipmentRepository) Delete(ctx context.Context, id string) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("shipment not found")
	}
	return nil
}
