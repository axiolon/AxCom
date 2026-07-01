// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package orders

import (
	"context"
	"errors"
	"time"

	"ecom-engine/internal/core/orders"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoOrderItem struct {
	VariantID string  `bson:"variant_id"`
	Quantity  int     `bson:"quantity"`
	Price     float64 `bson:"price"`
}

type MongoOrderCustomerSnapshot struct {
	Name          string `bson:"name"`
	Email         string `bson:"email"`
	ContactNumber string `bson:"contact_number"`
}

type MongoOrder struct {
	ID               string                     `bson:"_id"`
	CustomerID       string                     `bson:"customer_id"`
	CustomerSnapshot MongoOrderCustomerSnapshot `bson:"customer_snapshot"`
	Items            []MongoOrderItem           `bson:"items"`
	Total            float64                    `bson:"total"`
	Status           string                     `bson:"status"`
	CreatedAt        time.Time                  `bson:"created_at"`
}

type MongoOrderRepository struct {
	collection *mongo.Collection
}

func NewMongoOrderRepository(db *mongo.Database) *MongoOrderRepository {
	return &MongoOrderRepository{
		collection: db.Collection("orders"),
	}
}

// Convert from domain Order to MongoDB Order doc
func toOrderDoc(o *orders.Order) *MongoOrder {
	if o == nil {
		return nil
	}
	items := make([]MongoOrderItem, len(o.Items))
	for i, item := range o.Items {
		items[i] = MongoOrderItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	return &MongoOrder{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		CustomerSnapshot: MongoOrderCustomerSnapshot{
			Name:          o.CustomerSnapshot.Name,
			Email:         o.CustomerSnapshot.Email,
			ContactNumber: o.CustomerSnapshot.ContactNumber,
		},
		Items:     items,
		Total:     o.Total,
		Status:    string(o.Status),
		CreatedAt: o.CreatedAt,
	}
}

// Convert from MongoDB Order doc to domain Order
func toDomainOrder(doc *MongoOrder) *orders.Order {
	if doc == nil {
		return nil
	}
	items := make([]orders.OrderItem, len(doc.Items))
	for i, item := range doc.Items {
		items[i] = orders.OrderItem{
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	return &orders.Order{
		ID:         doc.ID,
		CustomerID: doc.CustomerID,
		CustomerSnapshot: orders.OrderCustomerSnapshot{
			Name:          doc.CustomerSnapshot.Name,
			Email:         doc.CustomerSnapshot.Email,
			ContactNumber: doc.CustomerSnapshot.ContactNumber,
		},
		Items:     items,
		Total:     doc.Total,
		Status:    orders.OrderStatus(doc.Status),
		CreatedAt: doc.CreatedAt,
	}
}

func (r *MongoOrderRepository) Create(ctx context.Context, o *orders.Order) error {
	doc := toOrderDoc(o)
	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

func (r *MongoOrderRepository) GetByID(ctx context.Context, id string) (*orders.Order, error) {
	var doc MongoOrder
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return toDomainOrder(&doc), nil
}

func (r *MongoOrderRepository) Update(ctx context.Context, o *orders.Order) error {
	doc := toOrderDoc(o)
	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": o.ID}, doc)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("order not found")
	}
	return nil
}

func (r *MongoOrderRepository) ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]orders.Order, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{"customer_id": customerID}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []MongoOrder
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	ordersList := make([]orders.Order, len(docs))
	for i, doc := range docs {
		ordersList[i] = *toDomainOrder(&doc)
	}
	return ordersList, nil
}

func (r *MongoOrderRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	pipeline := bson.A{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	result := make(map[string]int64)
	for cursor.Next(ctx) {
		var row struct {
			Status string `bson:"_id"`
			Count  int64  `bson:"count"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		result[row.Status] = row.Count
	}
	return result, cursor.Err()
}

func (r *MongoOrderRepository) SumRevenue(ctx context.Context, since time.Time) (float64, error) {
	var matchStage bson.D
	if !since.IsZero() {
		matchStage = bson.D{{Key: "$match", Value: bson.D{
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: since}}},
		}}}
	} else {
		matchStage = bson.D{{Key: "$match", Value: bson.D{}}}
	}

	pipeline := bson.A{
		matchStage,
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$total"}}},
		}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	if cursor.Next(ctx) {
		var row struct {
			Total float64 `bson:"total"`
		}
		if err := cursor.Decode(&row); err != nil {
			return 0, err
		}
		return row.Total, cursor.Err()
	}
	return 0, cursor.Err()
}

func (r *MongoOrderRepository) RevenueByDay(ctx context.Context, days int) ([]orders.DailyRevenue, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: since}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
				{Key: "format", Value: "%Y-%m-%d"},
				{Key: "date", Value: "$created_at"},
			}}}},
			{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$total"}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var result []orders.DailyRevenue
	for cursor.Next(ctx) {
		var row struct {
			Date    string  `bson:"_id"`
			Revenue float64 `bson:"revenue"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		result = append(result, orders.DailyRevenue{Date: row.Date, Revenue: row.Revenue})
	}
	return result, cursor.Err()
}

func (r *MongoOrderRepository) TopProducts(ctx context.Context, n int) ([]orders.ProductSales, error) {
	pipeline := bson.A{
		bson.D{{Key: "$unwind", Value: "$items"}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$items.variant_id"},
			{Key: "total_sold", Value: bson.D{{Key: "$sum", Value: "$items.quantity"}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "total_sold", Value: -1}}}},
		bson.D{{Key: "$limit", Value: n}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var result []orders.ProductSales
	for cursor.Next(ctx) {
		var row struct {
			VariantID string `bson:"_id"`
			TotalSold int64  `bson:"total_sold"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		result = append(result, orders.ProductSales{VariantID: row.VariantID, TotalSold: row.TotalSold})
	}
	return result, cursor.Err()
}

func (r *MongoOrderRepository) ListAll(ctx context.Context, limit, offset int) ([]orders.Order, error) {
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []MongoOrder
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	ordersList := make([]orders.Order, len(docs))
	for i, doc := range docs {
		ordersList[i] = *toDomainOrder(&doc)
	}
	return ordersList, nil
}
