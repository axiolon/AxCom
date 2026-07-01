// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"context"
	"errors"
	"time"

	"ecom-engine/internal/core/catalog/features/reviews"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoReviewReply struct {
	UserID    string `bson:"user_id"`
	Comment   string `bson:"comment"`
	CreatedAt int64  `bson:"created_at"`
}

type MongoReview struct {
	ID        string            `bson:"_id"`
	ProductID string            `bson:"product_id"`
	UserID    string            `bson:"user_id"`
	Rating    int               `bson:"rating"`
	Comment   string            `bson:"comment"`
	Reply     *MongoReviewReply `bson:"reply,omitempty"`
	CreatedAt int64             `bson:"created_at"`
}

type MongoReviewRepository struct {
	collection *mongo.Collection
}

// NewMongoRepository creates a new Mongo repository for reviews.
func NewMongoRepository(db *mongo.Database) reviews.Repository {
	return &MongoReviewRepository{
		collection: db.Collection("reviews"),
	}
}

func (r *MongoReviewRepository) CreateReview(ctx context.Context, review *reviews.Review) error {
	doc := toReviewDoc(review)
	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

func (r *MongoReviewRepository) GetReviewsByProductID(ctx context.Context, productID string) ([]reviews.Review, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"product_id": productID})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []MongoReview
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]reviews.Review, len(docs))
	for i, doc := range docs {
		result[i] = *toDomainReview(&doc)
	}
	return result, nil
}

func (r *MongoReviewRepository) GetReviewByID(ctx context.Context, id string) (*reviews.Review, error) {
	var doc MongoReview
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, reviews.ErrReviewNotFound
		}
		return nil, err
	}
	return toDomainReview(&doc), nil
}

func (r *MongoReviewRepository) UpdateReview(ctx context.Context, review *reviews.Review) error {
	doc := toReviewDoc(review)
	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": review.ID}, doc)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return reviews.ErrReviewNotFound
	}
	return nil
}

func (r *MongoReviewRepository) DeleteReview(ctx context.Context, id string) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return reviews.ErrReviewNotFound
	}
	return nil
}

func toReviewDoc(r *reviews.Review) *MongoReview {
	if r == nil {
		return nil
	}

	var reply *MongoReviewReply
	if r.Reply != nil {
		reply = &MongoReviewReply{
			UserID:    r.Reply.UserID,
			Comment:   r.Reply.Comment,
			CreatedAt: r.Reply.CreatedAt.Unix(),
		}
	}

	return &MongoReview{
		ID:        r.ID,
		ProductID: r.ProductID,
		UserID:    r.UserID,
		Rating:    r.Rating,
		Comment:   r.Comment,
		Reply:     reply,
		CreatedAt: r.CreatedAt.Unix(),
	}
}

func toDomainReview(doc *MongoReview) *reviews.Review {
	if doc == nil {
		return nil
	}

	var reply *reviews.ReviewReply
	if doc.Reply != nil {
		reply = &reviews.ReviewReply{
			UserID:    doc.Reply.UserID,
			Comment:   doc.Reply.Comment,
			CreatedAt: time.Unix(doc.Reply.CreatedAt, 0),
		}
	}

	return &reviews.Review{
		ID:        doc.ID,
		ProductID: doc.ProductID,
		UserID:    doc.UserID,
		Rating:    doc.Rating,
		Comment:   doc.Comment,
		Reply:     reply,
		CreatedAt: time.Unix(doc.CreatedAt, 0),
	}
}
