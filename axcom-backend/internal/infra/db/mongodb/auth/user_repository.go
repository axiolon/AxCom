// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth implements MongoDB persistence adapters for user accounts and token storage.
package auth

import (
	"context"
	"errors"

	"ecom-engine/internal/core/auth"
	"ecom-engine/pkg/logger"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.opentelemetry.io/otel"
)

// MongoUserRepository implements auth.UserRepository using MongoDB.
// Use NewMongoUserRepository to initialize; zero value is invalid.
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a MongoUserRepository utilizing the provided MongoDB database.
func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		collection: db.Collection("users"),
	}
}

// Create inserts a user document into MongoDB.
// It returns auth.ErrEmailAlreadyExists if the email is already taken.
func (r *MongoUserRepository) Create(ctx context.Context, user *auth.User) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoUserRepository.Create")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Inserting user document for ID: %s", user.ID)

	doc := toUserDoc(user)
	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to insert user document: %v", err)
		if mongo.IsDuplicateKeyError(err) {
			return auth.ErrEmailAlreadyExists
		}
		return err
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully inserted user document for ID: %s", user.ID)
	return nil
}

// GetByEmail retrieves a user document matching the specified email address.
// It returns auth.ErrUserNotFound if no document is matched.
func (r *MongoUserRepository) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoUserRepository.GetByEmail")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Finding user by email: %s", email)

	var doc userDoc
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.DebugCtx(ctx, "MongoDB: No user found for email: %s", email)
			return nil, auth.ErrUserNotFound
		}
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to find user by email %s: %v", email, err)
		return nil, err
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully found user by email: %s", email)
	return toDomainUser(&doc), nil
}

// GetByID retrieves a user document matching the specified unique identifier.
// It returns auth.ErrUserNotFound if no document is matched.
func (r *MongoUserRepository) GetByID(ctx context.Context, id string) (*auth.User, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoUserRepository.GetByID")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Finding user by ID: %s", id)

	var doc userDoc
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.DebugCtx(ctx, "MongoDB: No user found for ID: %s", id)
			return nil, auth.ErrUserNotFound
		}
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to find user by ID %s: %v", id, err)
		return nil, err
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully found user by ID: %s", id)
	return toDomainUser(&doc), nil
}

// UpdatePassword updates the hashed password value of the matched user.
// It returns auth.ErrUserNotFound if the user ID does not exist.
func (r *MongoUserRepository) UpdatePassword(ctx context.Context, userID string, hashedPassword string) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoUserRepository.UpdatePassword")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Updating password for user ID: %s", userID)

	res, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"password": hashedPassword}},
	)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to update password for user %s: %v", userID, err)
		return err
	}
	if res.MatchedCount == 0 {
		logger.WarnCtx(ctx, "MongoDB: Password update failed, user ID %s not found", userID)
		return auth.ErrUserNotFound
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully updated password for user ID: %s", userID)
	return nil
}

// Update updates user fields (e.g. lockout details, timestamps).
func (r *MongoUserRepository) Update(ctx context.Context, user *auth.User) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoUserRepository.Update")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Updating user document for ID: %s", user.ID)

	doc := toUserDoc(user)
	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": user.ID}, doc)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to update user document %s: %v", user.ID, err)
		return err
	}
	if res.MatchedCount == 0 {
		return auth.ErrUserNotFound
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully updated user document for ID: %s", user.ID)
	return nil
}

// GetByOIDCSub retrieves a user document matching the specified IdP subject identifier.
// It returns auth.ErrUserNotFound if no document is matched.
func (r *MongoUserRepository) GetByOIDCSub(ctx context.Context, sub string) (*auth.User, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoUserRepository.GetByOIDCSub")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Finding user by OIDC sub: %s", sub)

	var doc userDoc
	err := r.collection.FindOne(ctx, bson.M{"oidc_sub": sub}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.DebugCtx(ctx, "MongoDB: No user found for OIDC sub: %s", sub)
			return nil, auth.ErrUserNotFound
		}
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to find user by OIDC sub %s: %v", sub, err)
		return nil, err
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully found user by OIDC sub: %s", sub)
	return toDomainUser(&doc), nil
}
