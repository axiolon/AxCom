// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
)

type mockCatalogService struct {
	products map[string]*domain.Product
}

func (m *mockCatalogService) GetProduct(_ context.Context, id string) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func TestReviewSubmissionAndStats(t *testing.T) {
	catalogSvc := &mockCatalogService{
		products: map[string]*domain.Product{
			"prod_123": {
				ID:   "prod_123",
				Name: "Test Product",
			},
		},
	}

	repo := NewMemoryRepository()
	svc := NewReviewService(repo, catalogSvc)
	ctx := context.Background()

	// 1. Submit review successfully
	r1 := &Review{
		ProductID: "prod_123",
		UserID:    "user_1",
		Rating:    5,
		Comment:   "Outstanding product!",
	}
	err := svc.AddReview(ctx, r1)
	if err != nil {
		t.Fatalf("expected successful review submission, got error: %v", err)
	}

	// 2. Try submitting with invalid rating
	rInvalidRating := &Review{
		ProductID: "prod_123",
		UserID:    "user_2",
		Rating:    0,
		Comment:   "Bad rating",
	}
	err = svc.AddReview(ctx, rInvalidRating)
	if err == nil {
		t.Fatalf("expected error submitting review with rating 0")
	}

	// 3. Try submitting with empty comment
	rInvalidComment := &Review{
		ProductID: "prod_123",
		UserID:    "user_2",
		Rating:    4,
		Comment:   "",
	}
	err = svc.AddReview(ctx, rInvalidComment)
	if err == nil {
		t.Fatalf("expected error submitting review with empty comment")
	}

	// 4. Try submitting for a non-existent product
	rInvalidProduct := &Review{
		ProductID: "prod_unknown",
		UserID:    "user_2",
		Rating:    4,
		Comment:   "Nice product",
	}
	err = svc.AddReview(ctx, rInvalidProduct)
	if err == nil {
		t.Fatalf("expected error submitting review for non-existent product")
	}

	// 5. Submit more reviews and check average rating stats
	r2 := &Review{
		ProductID: "prod_123",
		UserID:    "user_2",
		Rating:    3,
		Comment:   "Average product.",
	}
	_ = svc.AddReview(ctx, r2)

	r3 := &Review{
		ProductID: "prod_123",
		UserID:    "user_3",
		Rating:    4,
		Comment:   "Pretty good overall.",
	}
	_ = svc.AddReview(ctx, r3)

	reviews, err := svc.GetReviewsByProductID(ctx, "prod_123")
	if err != nil {
		t.Fatalf("failed to retrieve reviews: %v", err)
	}
	if len(reviews) != 3 {
		t.Errorf("expected 3 reviews, got %d", len(reviews))
	}

	avg, count, err := svc.GetAverageRating(ctx, "prod_123")
	if err != nil {
		t.Fatalf("failed to get average rating: %v", err)
	}
	if count != 3 {
		t.Errorf("expected review count 3, got %d", count)
	}
	expectedAvg := 4.0 // (5 + 3 + 4) / 3 = 4.0
	if avg != expectedAvg {
		t.Errorf("expected average rating %f, got %f", expectedAvg, avg)
	}

	// 6. Test DeleteReview authorization & operation
	// Owner can delete
	err = svc.DeleteReview(ctx, r1.ID, "user_1", false)
	if err != nil {
		t.Fatalf("expected owner to be able to delete, got error: %v", err)
	}

	// Deleted review should not be returned
	_, err = repo.GetReviewByID(ctx, r1.ID)
	if !errors.Is(err, ErrReviewNotFound) {
		t.Fatalf("expected review to be deleted from repo, got %v", err)
	}

	// Non-owner cannot delete
	err = svc.DeleteReview(ctx, r2.ID, "user_different", false)
	if err == nil {
		t.Fatalf("expected non-owner deletion to fail")
	}

	// Admin can delete any review
	err = svc.DeleteReview(ctx, r2.ID, "user_different", true)
	if err != nil {
		t.Fatalf("expected admin to be able to delete, got error: %v", err)
	}

	// 7. Test SubmitReply
	// Submitting a valid reply
	reply := &ReviewReply{
		UserID:  "merchant_1",
		Comment: "Thank you for the review!",
	}
	err = svc.SubmitReply(ctx, r3.ID, reply)
	if err != nil {
		t.Fatalf("expected successful reply submission, got error: %v", err)
	}

	updatedRev, err := repo.GetReviewByID(ctx, r3.ID)
	if err != nil {
		t.Fatalf("failed to retrieve review after reply: %v", err)
	}
	if updatedRev.Reply == nil {
		t.Fatalf("expected reply to be attached to review")
	}
	if updatedRev.Reply.Comment != "Thank you for the review!" {
		t.Errorf("expected reply comment 'Thank you for the review!', got '%s'", updatedRev.Reply.Comment)
	}

	// Try replying with empty comment
	invalidReply := &ReviewReply{
		UserID:  "merchant_1",
		Comment: "",
	}
	err = svc.SubmitReply(ctx, r3.ID, invalidReply)
	if err == nil {
		t.Fatalf("expected error submitting empty reply comment")
	}

	// 8. Try submitting a duplicate review for the same user and product (C4-1)
	rDuplicate := &Review{
		ProductID: "prod_123",
		UserID:    "user_3",
		Rating:    5,
		Comment:   "Another review!",
	}
	err = svc.AddReview(ctx, rDuplicate)
	if err == nil {
		t.Fatalf("expected error submitting duplicate review for same user and product, but succeeded")
	}

	// 9. Test UpdateReview (C4-2)
	// Owner can update
	updated, err := svc.UpdateReview(ctx, r3.ID, "user_3", 2, "Decent but has issues.")
	if err != nil {
		t.Fatalf("unexpected error updating review: %v", err)
	}
	if updated.Rating != 2 || updated.Comment != "Decent but has issues." {
		t.Errorf("expected updated rating/comment, got rating=%d comment=%s", updated.Rating, updated.Comment)
	}

	// Non-owner cannot update
	_, err = svc.UpdateReview(ctx, r3.ID, "user_different", 5, "I hacked it!")
	if err == nil {
		t.Fatal("expected error updating review as non-owner, got nil")
	}

	// Validation failure on update
	_, err = svc.UpdateReview(ctx, r3.ID, "user_3", 10, "")
	if err == nil {
		t.Fatal("expected validation error on invalid update payload, got nil")
	}
}
