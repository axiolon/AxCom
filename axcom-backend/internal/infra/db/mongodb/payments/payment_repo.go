// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	"errors"
	"time"

	"ecom-engine/internal/core/payments"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoPayment struct {
	ID               string     `bson:"_id"`
	OrderID          string     `bson:"order_id"`
	CustomerID       string     `bson:"customer_id"`
	Amount           float64    `bson:"amount"`
	Currency         string     `bson:"currency"`
	Provider         string     `bson:"provider"`
	ProviderIntentID string     `bson:"provider_intent_id"`
	Status           string     `bson:"status"`
	IdempotencyKey   string     `bson:"idempotency_key"`
	FailureReason    string     `bson:"failure_reason,omitempty"`
	CreatedAt        time.Time  `bson:"created_at"`
	UpdatedAt        time.Time  `bson:"updated_at"`
	RefundedAt       *time.Time `bson:"refunded_at,omitempty"`
}

type MongoPaymentRepository struct {
	collection *mongo.Collection
}

func NewMongoPaymentRepository(db *mongo.Database) *MongoPaymentRepository {
	return &MongoPaymentRepository{
		collection: db.Collection("payments"),
	}
}

func toPaymentDoc(p *payments.Payment) *MongoPayment {
	if p == nil {
		return nil
	}
	return &MongoPayment{
		ID:               p.ID,
		OrderID:          p.OrderID,
		CustomerID:       p.CustomerID,
		Amount:           p.Amount,
		Currency:         p.Currency,
		Provider:         p.Provider,
		ProviderIntentID: p.ProviderIntentID,
		Status:           string(p.Status),
		IdempotencyKey:   p.IdempotencyKey,
		FailureReason:    p.FailureReason,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		RefundedAt:       p.RefundedAt,
	}
}

func toDomainPayment(doc *MongoPayment) *payments.Payment {
	if doc == nil {
		return nil
	}
	return &payments.Payment{
		ID:               doc.ID,
		OrderID:          doc.OrderID,
		CustomerID:       doc.CustomerID,
		Amount:           doc.Amount,
		Currency:         doc.Currency,
		Provider:         doc.Provider,
		ProviderIntentID: doc.ProviderIntentID,
		Status:           payments.PaymentStatus(doc.Status),
		IdempotencyKey:   doc.IdempotencyKey,
		FailureReason:    doc.FailureReason,
		CreatedAt:        doc.CreatedAt,
		UpdatedAt:        doc.UpdatedAt,
		RefundedAt:       doc.RefundedAt,
	}
}

func (r *MongoPaymentRepository) Create(ctx context.Context, p *payments.Payment) error {
	doc := toPaymentDoc(p)
	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return payments.ErrDuplicatePayment
		}
		return err
	}
	return nil
}

func (r *MongoPaymentRepository) GetByID(ctx context.Context, id string) (*payments.Payment, error) {
	var doc MongoPayment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, payments.ErrNotFound
		}
		return nil, err
	}
	return toDomainPayment(&doc), nil
}

func (r *MongoPaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*payments.Payment, error) {
	var doc MongoPayment
	err := r.collection.FindOne(ctx, bson.M{"order_id": orderID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, payments.ErrNotFound
		}
		return nil, err
	}
	return toDomainPayment(&doc), nil
}

func (r *MongoPaymentRepository) GetByProviderIntentID(ctx context.Context, provider string, intentID string) (*payments.Payment, error) {
	var doc MongoPayment
	err := r.collection.FindOne(ctx, bson.M{"provider": provider, "provider_intent_id": intentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, payments.ErrNotFound
		}
		return nil, err
	}
	return toDomainPayment(&doc), nil
}

func (r *MongoPaymentRepository) Update(ctx context.Context, p *payments.Payment) error {
	doc := toPaymentDoc(p)
	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": p.ID}, doc)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return payments.ErrNotFound
	}
	return nil
}

func (r *MongoPaymentRepository) ListAll(ctx context.Context, limit, offset int) ([]payments.Payment, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []MongoPayment
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	paymentsList := make([]payments.Payment, len(docs))
	for i, doc := range docs {
		paymentsList[i] = *toDomainPayment(&doc)
	}
	return paymentsList, nil
}

func (r *MongoPaymentRepository) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]payments.Payment, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{"customer_id": customerID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []MongoPayment
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	paymentsList := make([]payments.Payment, len(docs))
	for i, doc := range docs {
		paymentsList[i] = *toDomainPayment(&doc)
	}
	return paymentsList, nil
}
