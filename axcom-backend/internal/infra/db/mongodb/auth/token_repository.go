// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"time"

	"ecom-engine/internal/core/auth"
	"ecom-engine/pkg/logger"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/otel"
)

// MongoTokenRepository implements auth.TokenRepository using MongoDB.
// Use NewMongoTokenRepository to initialize; zero value is invalid.
type MongoTokenRepository struct {
	refreshCollection *mongo.Collection
	resetCollection   *mongo.Collection
}

// NewMongoTokenRepository creates a MongoTokenRepository utilizing the provided MongoDB database.
func NewMongoTokenRepository(db *mongo.Database) *MongoTokenRepository {
	return &MongoTokenRepository{
		refreshCollection: db.Collection("refresh_tokens"),
		resetCollection:   db.Collection("password_reset_tokens"),
	}
}

// SaveRefreshToken persists a refresh token document.
func (r *MongoTokenRepository) SaveRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.SaveRefreshToken")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Saving refresh token for user ID: %s", token.UserID)

	doc := toRefreshTokenDoc(token)
	_, err := r.refreshCollection.InsertOne(ctx, doc)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to save refresh token: %v", err)
	} else {
		logger.DebugCtx(ctx, "MongoDB: Successfully saved refresh token")
	}
	return err
}

// GetRefreshToken retrieves a refresh token document matching the specified token value.
// It returns auth.ErrTokenRevoked if the token does not exist in the collection.
func (r *MongoTokenRepository) GetRefreshToken(ctx context.Context, token string) (*auth.RefreshToken, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.GetRefreshToken")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Fetching refresh token")

	var doc refreshTokenDoc
	err := r.refreshCollection.FindOne(ctx, bson.M{"token": token}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.DebugCtx(ctx, "MongoDB: Refresh token not found")
			return nil, auth.ErrTokenRevoked
		}
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to fetch refresh token: %v", err)
		return nil, err
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully fetched refresh token")
	return toDomainRefreshToken(&doc), nil
}

// RevokeRefreshToken marks a refresh token document as revoked.
// It returns auth.ErrTokenRevoked if the token does not exist.
func (r *MongoTokenRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.RevokeRefreshToken")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Revoking refresh token")

	res, err := r.refreshCollection.UpdateOne(
		ctx,
		bson.M{"token": token},
		bson.M{"$set": bson.M{"revoked": true}},
	)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to revoke refresh token: %v", err)
		return err
	}
	if res.MatchedCount == 0 {
		logger.WarnCtx(ctx, "MongoDB: Revoke failed, token not found")
		return auth.ErrTokenRevoked
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully revoked refresh token")
	return nil
}

// RevokeAllUserTokens marks all active refresh token documents belonging to a user as revoked.
func (r *MongoTokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.RevokeAllUserTokens")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Revoking all active tokens for user ID: %s", userID)

	_, err := r.refreshCollection.UpdateMany(
		ctx,
		bson.M{"user_id": userID, "revoked": false},
		bson.M{"$set": bson.M{"revoked": true}},
	)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to revoke user tokens: %v", err)
	} else {
		logger.DebugCtx(ctx, "MongoDB: Successfully revoked all tokens for user ID: %s", userID)
	}
	return err
}

// SavePasswordResetToken persists a password reset token document.
func (r *MongoTokenRepository) SavePasswordResetToken(ctx context.Context, token *auth.PasswordResetToken) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.SavePasswordResetToken")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Saving password reset token for user ID: %s", token.UserID)

	doc := toPasswordResetTokenDoc(token)
	_, err := r.resetCollection.InsertOne(ctx, doc)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to save reset token: %v", err)
	} else {
		logger.DebugCtx(ctx, "MongoDB: Successfully saved reset token")
	}
	return err
}

// GetPasswordResetToken retrieves a password reset token document matching the specified token value.
// It returns auth.ErrResetTokenInvalid if the token does not exist in the collection.
func (r *MongoTokenRepository) GetPasswordResetToken(ctx context.Context, token string) (*auth.PasswordResetToken, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.GetPasswordResetToken")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Fetching password reset token")

	var doc passwordResetTokenDoc
	err := r.resetCollection.FindOne(ctx, bson.M{"token": token}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.DebugCtx(ctx, "MongoDB: Password reset token not found")
			return nil, auth.ErrResetTokenInvalid
		}
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to fetch reset token: %v", err)
		return nil, err
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully fetched reset token")
	return toDomainPasswordResetToken(&doc), nil
}

// MarkPasswordResetUsed marks a password reset token document as used.
// It returns auth.ErrResetTokenInvalid if the token does not exist.
func (r *MongoTokenRepository) MarkPasswordResetUsed(ctx context.Context, token string) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.MarkPasswordResetUsed")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Marking password reset token as used")

	res, err := r.resetCollection.UpdateOne(
		ctx,
		bson.M{"token": token},
		bson.M{"$set": bson.M{"used": true}},
	)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to update reset token status: %v", err)
		return err
	}
	if res.MatchedCount == 0 {
		logger.WarnCtx(ctx, "MongoDB: Update failed, reset token not found")
		return auth.ErrResetTokenInvalid
	}
	logger.DebugCtx(ctx, "MongoDB: Successfully marked reset token as used")
	return nil
}

// GetActiveRefreshTokens retrieves all non-revoked and non-expired refresh tokens for a user.
func (r *MongoTokenRepository) GetActiveRefreshTokens(ctx context.Context, userID string) ([]*auth.RefreshToken, error) {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.GetActiveRefreshTokens")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Fetching active refresh tokens for user ID: %s", userID)

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.refreshCollection.Find(ctx, bson.M{
		"user_id":    userID,
		"revoked":    false,
		"expires_at": bson.M{"$gt": time.Now()},
	}, opts)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []refreshTokenDoc
	if err := cursor.All(ctx, &docs); err != nil {
		span.RecordError(err)
		return nil, err
	}

	tokens := make([]*auth.RefreshToken, len(docs))
	for i, doc := range docs {
		tokens[i] = toDomainRefreshToken(&doc)
	}
	return tokens, nil
}

// InvalidateUserResetTokens marks all active password reset tokens for a user as used/expired.
func (r *MongoTokenRepository) InvalidateUserResetTokens(ctx context.Context, userID string) error {
	ctx, span := otel.Tracer("mongodb").Start(ctx, "MongoTokenRepository.InvalidateUserResetTokens")
	defer span.End()

	logger.DebugCtx(ctx, "MongoDB: Invalidating active reset tokens for user ID: %s", userID)

	_, err := r.resetCollection.UpdateMany(
		ctx,
		bson.M{"user_id": userID, "used": false},
		bson.M{"$set": bson.M{"used": true}},
	)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "MongoDB: Failed to invalidate user reset tokens: %v", err)
	}
	return err
}
