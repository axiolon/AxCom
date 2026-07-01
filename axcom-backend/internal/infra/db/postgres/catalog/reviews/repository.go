// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"context"
	"database/sql"

	"ecom-engine/internal/core/catalog/features/reviews"
	"ecom-engine/internal/infra/db"
	"ecom-engine/pkg/logger"

	"go.opentelemetry.io/otel"
)

type PostgresReviewRepository struct {
	db db.Database
}

func NewPostgresRepository(database db.Database) reviews.Repository {
	return &PostgresReviewRepository{
		db: database,
	}
}

func (r *PostgresReviewRepository) CreateReview(ctx context.Context, review *reviews.Review) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresReviewRepository.CreateReview")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Creating review for product ID: %s", review.ProductID)

	var replyUser sql.NullString
	var replyComment sql.NullString
	var replyCreatedAt sql.NullTime
	if review.Reply != nil {
		replyUser = sql.NullString{String: review.Reply.UserID, Valid: true}
		replyComment = sql.NullString{String: review.Reply.Comment, Valid: true}
		replyCreatedAt = sql.NullTime{Time: review.Reply.CreatedAt, Valid: true}
	}

	query := `INSERT INTO reviews (id, product_id, user_id, rating, comment, reply_user_id, reply_comment, reply_created_at, created_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	err := r.db.Exec(ctx, query, review.ID, review.ProductID, review.UserID, review.Rating, review.Comment, replyUser, replyComment, replyCreatedAt, review.CreatedAt)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to insert review: %v", err)
		return err
	}
	return nil
}

func (r *PostgresReviewRepository) GetReviewsByProductID(ctx context.Context, productID string) ([]reviews.Review, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresReviewRepository.GetReviewsByProductID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding reviews for product ID: %s", productID)

	query := `SELECT id, product_id, user_id, rating, comment, reply_user_id, reply_comment, reply_created_at, created_at 
              FROM reviews WHERE product_id = $1`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to query reviews: %v", err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []reviews.Review
	for rows.Next() {
		rev, err := r.scanReview(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *rev)
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Error iterating reviews: %v", err)
		return nil, err
	}
	return result, nil
}

func (r *PostgresReviewRepository) GetReviewByID(ctx context.Context, id string) (*reviews.Review, error) {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresReviewRepository.GetReviewByID")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Finding review ID: %s", id)

	query := `SELECT id, product_id, user_id, rating, comment, reply_user_id, reply_comment, reply_created_at, created_at 
              FROM reviews WHERE id = $1 LIMIT 1`
	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, reviews.ErrReviewNotFound
	}

	return r.scanReview(rows)
}

func (r *PostgresReviewRepository) UpdateReview(ctx context.Context, review *reviews.Review) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresReviewRepository.UpdateReview")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Updating review ID: %s", review.ID)

	var replyUser sql.NullString
	var replyComment sql.NullString
	var replyCreatedAt sql.NullTime
	if review.Reply != nil {
		replyUser = sql.NullString{String: review.Reply.UserID, Valid: true}
		replyComment = sql.NullString{String: review.Reply.Comment, Valid: true}
		replyCreatedAt = sql.NullTime{Time: review.Reply.CreatedAt, Valid: true}
	}

	query := `UPDATE reviews 
              SET rating = $1, comment = $2, reply_user_id = $3, reply_comment = $4, reply_created_at = $5 
              WHERE id = $6`
	res, err := r.db.ExecResult(ctx, query, review.Rating, review.Comment, replyUser, replyComment, replyCreatedAt, review.ID)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to update review: %v", err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		return reviews.ErrReviewNotFound
	}
	return nil
}

func (r *PostgresReviewRepository) DeleteReview(ctx context.Context, id string) error {
	ctx, span := otel.Tracer("postgres").Start(ctx, "PostgresReviewRepository.DeleteReview")
	defer span.End()

	logger.DebugCtx(ctx, "Postgres: Deleting review ID: %s", id)

	query := "DELETE FROM reviews WHERE id = $1"
	res, err := r.db.ExecResult(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		logger.ErrorCtx(ctx, "Postgres: Failed to delete review: %v", err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		return err
	}
	if rowsAffected == 0 {
		return reviews.ErrReviewNotFound
	}
	return nil
}

func (r *PostgresReviewRepository) scanReview(rows db.Rows) (*reviews.Review, error) {
	var rev reviews.Review
	var replyUser sql.NullString
	var replyComment sql.NullString
	var replyCreatedAt sql.NullTime

	err := rows.Scan(&rev.ID, &rev.ProductID, &rev.UserID, &rev.Rating, &rev.Comment, &replyUser, &replyComment, &replyCreatedAt, &rev.CreatedAt)
	if err != nil {
		return nil, err
	}

	if replyUser.Valid {
		rev.Reply = &reviews.ReviewReply{
			UserID:    replyUser.String,
			Comment:   replyComment.String,
			CreatedAt: replyCreatedAt.Time,
		}
	}
	return &rev, nil
}
