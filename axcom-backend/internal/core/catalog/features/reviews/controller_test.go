// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package reviews

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/pkg/ctxkeys"
	apperrors "ecom-engine/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	addReview             func(ctx context.Context, r *Review) error
	getReviewsByProductID func(ctx context.Context, productID string) ([]Review, error)
	getAverageRating      func(ctx context.Context, productID string) (float64, int, error)
	submitReply           func(ctx context.Context, reviewID string, reply *ReviewReply) error
	deleteReview          func(ctx context.Context, reviewID string, userID string, isAdmin bool) error
	updateReview          func(ctx context.Context, reviewID string, userID string, rating int, comment string) (*Review, error)
}

func (m *mockService) AddReview(ctx context.Context, r *Review) error {
	if m.addReview != nil {
		return m.addReview(ctx, r)
	}
	return nil
}

func (m *mockService) GetReviewsByProductID(ctx context.Context, productID string) ([]Review, error) {
	if m.getReviewsByProductID != nil {
		return m.getReviewsByProductID(ctx, productID)
	}
	return nil, nil
}

func (m *mockService) GetAverageRating(ctx context.Context, productID string) (float64, int, error) {
	if m.getAverageRating != nil {
		return m.getAverageRating(ctx, productID)
	}
	return 0, 0, nil
}

func (m *mockService) SubmitReply(ctx context.Context, reviewID string, reply *ReviewReply) error {
	if m.submitReply != nil {
		return m.submitReply(ctx, reviewID, reply)
	}
	return nil
}

func (m *mockService) DeleteReview(ctx context.Context, reviewID string, userID string, isAdmin bool) error {
	if m.deleteReview != nil {
		return m.deleteReview(ctx, reviewID, userID, isAdmin)
	}
	return nil
}

func (m *mockService) UpdateReview(ctx context.Context, reviewID string, userID string, rating int, comment string) (*Review, error) {
	if m.updateReview != nil {
		return m.updateReview(ctx, reviewID, userID, rating, comment)
	}
	return nil, nil
}

type mockProductVerifier struct {
	products map[string]*domain.Product
}

func (m *mockProductVerifier) GetProduct(_ context.Context, id string) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func setupTestRouter(svc Service, userID string, role string) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")

	// Mock authMiddleware that injects the provided userID and role
	mockAuthMiddleware := func(c *gin.Context) {
		if userID != "" {
			c.Set(string(ctxkeys.UserIDKey), userID)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		if role != "" {
			c.Set(string(ctxkeys.UserRoleKey), role)
		}
		c.Next()
	}

	mockAdminOnlyMiddleware := func(c *gin.Context) {
		roleVal := c.GetString(string(ctxkeys.UserRoleKey))
		if roleVal != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: admin role required"})
			return
		}
		c.Next()
	}

	ctrl := NewController(svc)
	RegisterRoutes(rg, ctrl, mockAuthMiddleware, mockAdminOnlyMiddleware)
	return router
}

// ----------------------------------------------------
// INTEGRATION TESTS (Mocked Service)
// ----------------------------------------------------

func TestController_SubmitReview_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful review submission", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			addReview: func(_ context.Context, r *Review) error {
				assert.Equal(t, "prod_123", r.ProductID)
				assert.Equal(t, "user_123", r.UserID)
				assert.Equal(t, 5, r.Rating)
				assert.Equal(t, "Great product!", r.Comment)
				r.ID = "rev_abc"
				r.CreatedAt = time.Now()
				return nil
			},
		}

		router := setupTestRouter(mockSvc, "user_123", "customer")
		reqBody := CreateReviewRequest{
			Rating:  5,
			Comment: "Great product!",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_123/reviews", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var envelope struct {
			Success bool           `json:"success"`
			Data    ReviewResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, "rev_abc", envelope.Data.ID)
		assert.Equal(t, 5, envelope.Data.Rating)
	})

	t.Run("fails - unauthenticated", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		// Set empty userID to simulate unauthenticated request
		router := setupTestRouter(mockSvc, "", "")
		reqBody := CreateReviewRequest{
			Rating:  5,
			Comment: "Great product!",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_123/reviews", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("fails - bad request payload", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc, "user_123", "customer")

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_123/reviews", bytes.NewBufferString("{bad_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_GetReviews_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful list reviews", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			getReviewsByProductID: func(_ context.Context, productID string) ([]Review, error) {
				assert.Equal(t, "prod_123", productID)
				return []Review{
					{ID: "rev_1", ProductID: "prod_123", UserID: "user_1", Rating: 4, Comment: "Nice"},
				}, nil
			},
		}

		router := setupTestRouter(mockSvc, "", "")
		req, _ := http.NewRequest(http.MethodGet, "/api/products/prod_123/reviews", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var envelope struct {
			Success bool             `json:"success"`
			Data    []ReviewResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Len(t, envelope.Data, 1)
		assert.Equal(t, "rev_1", envelope.Data[0].ID)
	})
}

func TestController_GetRatingSummary_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful rating summary", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			getAverageRating: func(_ context.Context, productID string) (float64, int, error) {
				assert.Equal(t, "prod_123", productID)
				return 4.5, 12, nil
			},
		}

		router := setupTestRouter(mockSvc, "", "")
		req, _ := http.NewRequest(http.MethodGet, "/api/products/prod_123/reviews/rating", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var envelope struct {
			Success bool                  `json:"success"`
			Data    ProductRatingResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, 4.5, envelope.Data.AverageRating)
		assert.Equal(t, 12, envelope.Data.ReviewCount)
	})
}

func TestController_DeleteReview_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful deletion", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			deleteReview: func(_ context.Context, reviewID string, userID string, isAdmin bool) error {
				assert.Equal(t, "rev_abc", reviewID)
				assert.Equal(t, "user_123", userID)
				assert.True(t, isAdmin)
				return nil
			},
		}

		router := setupTestRouter(mockSvc, "user_123", "admin")
		req, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_123/reviews/rev_abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("fails - forbidden service error mapping", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			deleteReview: func(_ context.Context, _ string, _ string, _ bool) error {
				return apperrors.NewForbidden("unauthorized", nil)
			},
		}

		router := setupTestRouter(mockSvc, "user_other", "admin")
		req, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_123/reviews/rev_abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestController_SubmitReply_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful reply submission", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			submitReply: func(_ context.Context, reviewID string, reply *ReviewReply) error {
				assert.Equal(t, "rev_abc", reviewID)
				assert.Equal(t, "user_123", reply.UserID)
				assert.Equal(t, "Thank you!", reply.Comment)
				return nil
			},
		}

		router := setupTestRouter(mockSvc, "user_123", "admin")
		reqBody := SubmitReplyRequest{
			Comment: "Thank you!",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_123/reviews/rev_abc/reply", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var envelope struct {
			Success bool                `json:"success"`
			Data    ReviewReplyResponse `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &envelope)
		require.NoError(t, err)
		assert.True(t, envelope.Success)
		assert.Equal(t, "user_123", envelope.Data.UserID)
		assert.Equal(t, "Thank you!", envelope.Data.Comment)
	})

	t.Run("fails - unauthenticated", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc, "", "")
		reqBody := SubmitReplyRequest{
			Comment: "Thank you!",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_123/reviews/rev_abc/reply", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("fails - forbidden non-admin", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc, "user_123", "customer")
		reqBody := SubmitReplyRequest{
			Comment: "Thank you!",
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_123/reviews/rev_abc/reply", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestReviewFlow_E2E(t *testing.T) {
	t.Parallel()

	// 1. Setup real components
	repo := NewMemoryRepository()
	verifier := &mockProductVerifier{
		products: map[string]*domain.Product{
			"prod_abc": {
				ID:   "prod_abc",
				Name: "E2E Wireless Mouse",
			},
		},
	}
	service := NewReviewService(repo, verifier)

	// We'll use a dynamic router that allows us to change auth credentials easily
	var currentUserID string
	var currentUserRole string

	router := gin.New()
	rg := router.Group("/api")
	mockAuthMiddleware := func(c *gin.Context) {
		if currentUserID != "" {
			c.Set(string(ctxkeys.UserIDKey), currentUserID)
		}
		if currentUserRole != "" {
			c.Set(string(ctxkeys.UserRoleKey), currentUserRole)
		}
		c.Next()
	}
	mockAdminOnlyMiddleware := func(c *gin.Context) {
		if currentUserRole != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: admin role required"})
			return
		}
		c.Next()
	}
	ctrl := NewController(service)
	RegisterRoutes(rg, ctrl, mockAuthMiddleware, mockAdminOnlyMiddleware)

	// Verify initially no reviews
	reviews, err := repo.GetReviewsByProductID(context.Background(), "prod_abc")
	require.NoError(t, err)
	assert.Empty(t, reviews)

	// 2. Submit a review via HTTP
	currentUserID = "user_cust1"
	currentUserRole = "customer"

	submitReq := CreateReviewRequest{
		Rating:  4,
		Comment: "Very nice feel, lightweight.",
	}
	submitBytes, _ := json.Marshal(submitReq)
	req1, _ := http.NewRequest(http.MethodPost, "/api/products/prod_abc/reviews", bytes.NewBuffer(submitBytes))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	var submitEnvelope struct {
		Success bool           `json:"success"`
		Data    ReviewResponse `json:"data"`
	}
	err = json.Unmarshal(w1.Body.Bytes(), &submitEnvelope)
	require.NoError(t, err)
	assert.True(t, submitEnvelope.Success)
	assert.NotEmpty(t, submitEnvelope.Data.ID)
	assert.Equal(t, 4, submitEnvelope.Data.Rating)

	// 3. Submit a second review via HTTP
	currentUserID = "user_cust2"
	currentUserRole = "customer"

	submitReq2 := CreateReviewRequest{
		Rating:  5,
		Comment: "Absolutely amazing!",
	}
	submitBytes2, _ := json.Marshal(submitReq2)
	req2, _ := http.NewRequest(http.MethodPost, "/api/products/prod_abc/reviews", bytes.NewBuffer(submitBytes2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	var submitEnvelope2 struct {
		Success bool           `json:"success"`
		Data    ReviewResponse `json:"data"`
	}
	err = json.Unmarshal(w2.Body.Bytes(), &submitEnvelope2)
	require.NoError(t, err)
	assert.True(t, submitEnvelope2.Success)

	// 4. Get Reviews list
	reqList, _ := http.NewRequest(http.MethodGet, "/api/products/prod_abc/reviews", nil)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	assert.Equal(t, http.StatusOK, wList.Code)

	var listEnvelope struct {
		Success bool             `json:"success"`
		Data    []ReviewResponse `json:"data"`
	}
	err = json.Unmarshal(wList.Body.Bytes(), &listEnvelope)
	require.NoError(t, err)
	assert.True(t, listEnvelope.Success)
	assert.Len(t, listEnvelope.Data, 2)

	// 5. Get Rating Summary
	reqSummary, _ := http.NewRequest(http.MethodGet, "/api/products/prod_abc/reviews/rating", nil)
	wSummary := httptest.NewRecorder()
	router.ServeHTTP(wSummary, reqSummary)
	assert.Equal(t, http.StatusOK, wSummary.Code)

	var summaryEnvelope struct {
		Success bool                  `json:"success"`
		Data    ProductRatingResponse `json:"data"`
	}
	err = json.Unmarshal(wSummary.Body.Bytes(), &summaryEnvelope)
	require.NoError(t, err)
	assert.True(t, summaryEnvelope.Success)
	assert.Equal(t, 4.5, summaryEnvelope.Data.AverageRating)
	assert.Equal(t, 2, summaryEnvelope.Data.ReviewCount)

	// 5a. Submit Reply via HTTP as customer (fails)
	currentUserID = "user_cust1"
	currentUserRole = "customer"

	replyReqCust := SubmitReplyRequest{
		Comment: "Trying to reply as customer",
	}
	replyBytesCust, _ := json.Marshal(replyReqCust)
	reqReplyCust, _ := http.NewRequest(http.MethodPost, "/api/products/prod_abc/reviews/"+submitEnvelope.Data.ID+"/reply", bytes.NewBuffer(replyBytesCust))
	reqReplyCust.Header.Set("Content-Type", "application/json")
	wReplyCust := httptest.NewRecorder()
	router.ServeHTTP(wReplyCust, reqReplyCust)
	assert.Equal(t, http.StatusForbidden, wReplyCust.Code)

	// 5b. Submit Reply via HTTP
	currentUserID = "user_admin"
	currentUserRole = "admin"

	replyReq := SubmitReplyRequest{
		Comment: "Thank you for the wonderful feedback!",
	}
	replyBytes, _ := json.Marshal(replyReq)
	reqReply, _ := http.NewRequest(http.MethodPost, "/api/products/prod_abc/reviews/"+submitEnvelope.Data.ID+"/reply", bytes.NewBuffer(replyBytes))
	reqReply.Header.Set("Content-Type", "application/json")
	wReply := httptest.NewRecorder()
	router.ServeHTTP(wReply, reqReply)
	assert.Equal(t, http.StatusOK, wReply.Code)

	var replyEnvelope struct {
		Success bool                `json:"success"`
		Data    ReviewReplyResponse `json:"data"`
	}
	err = json.Unmarshal(wReply.Body.Bytes(), &replyEnvelope)
	require.NoError(t, err)
	assert.True(t, replyEnvelope.Success)
	assert.Equal(t, "user_admin", replyEnvelope.Data.UserID)
	assert.Equal(t, "Thank you for the wonderful feedback!", replyEnvelope.Data.Comment)

	// 5c. Get Reviews list and verify reply is present
	reqList2, _ := http.NewRequest(http.MethodGet, "/api/products/prod_abc/reviews", nil)
	wList2 := httptest.NewRecorder()
	router.ServeHTTP(wList2, reqList2)
	assert.Equal(t, http.StatusOK, wList2.Code)

	var listEnvelope2 struct {
		Success bool             `json:"success"`
		Data    []ReviewResponse `json:"data"`
	}
	err = json.Unmarshal(wList2.Body.Bytes(), &listEnvelope2)
	require.NoError(t, err)
	assert.True(t, listEnvelope2.Success)
	assert.Len(t, listEnvelope2.Data, 2)

	// Find the first review (submitEnvelope.Data.ID) in the list and verify the reply
	var foundReply bool
	for _, r := range listEnvelope2.Data {
		if r.ID == submitEnvelope.Data.ID {
			assert.NotNil(t, r.Reply)
			assert.Equal(t, "user_admin", r.Reply.UserID)
			assert.Equal(t, "Thank you for the wonderful feedback!", r.Reply.Comment)
			foundReply = true
		}
	}
	assert.True(t, foundReply)

	// 6. Delete review as non-owner (fails)
	currentUserID = "user_cust3"
	currentUserRole = "customer"

	reqDel1, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_abc/reviews/"+submitEnvelope.Data.ID, nil)
	wDel1 := httptest.NewRecorder()
	router.ServeHTTP(wDel1, reqDel1)
	assert.Equal(t, http.StatusForbidden, wDel1.Code)

	// 7. Delete review as owner (succeeds)
	currentUserID = "user_cust1"
	currentUserRole = "customer"

	reqDel2, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_abc/reviews/"+submitEnvelope.Data.ID, nil)
	wDel2 := httptest.NewRecorder()
	router.ServeHTTP(wDel2, reqDel2)
	assert.Equal(t, http.StatusOK, wDel2.Code)

	// 8. Delete reviews as Admin (succeeds for remaining reviews)
	currentUserID = "user_admin"
	currentUserRole = "admin"

	// Review 1 is already deleted by owner, so deleting it again should return 404
	reqDel3, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_abc/reviews/"+submitEnvelope.Data.ID, nil)
	wDel3 := httptest.NewRecorder()
	router.ServeHTTP(wDel3, reqDel3)
	assert.Equal(t, http.StatusNotFound, wDel3.Code)

	// Review 2 is still present, so deleting it should succeed
	reqDel4, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_abc/reviews/"+submitEnvelope2.Data.ID, nil)
	wDel4 := httptest.NewRecorder()
	router.ServeHTTP(wDel4, reqDel4)
	assert.Equal(t, http.StatusOK, wDel4.Code)

	// 9. Verify rating summary is now 0/0
	reqSummaryFinal, _ := http.NewRequest(http.MethodGet, "/api/products/prod_abc/reviews/rating", nil)
	wSummaryFinal := httptest.NewRecorder()
	router.ServeHTTP(wSummaryFinal, reqSummaryFinal)
	assert.Equal(t, http.StatusOK, wSummaryFinal.Code)

	var summaryEnvelopeFinal struct {
		Success bool                  `json:"success"`
		Data    ProductRatingResponse `json:"data"`
	}
	err = json.Unmarshal(wSummaryFinal.Body.Bytes(), &summaryEnvelopeFinal)
	require.NoError(t, err)
	assert.True(t, summaryEnvelopeFinal.Success)
	assert.Equal(t, 0.0, summaryEnvelopeFinal.Data.AverageRating)
	assert.Equal(t, 0, summaryEnvelopeFinal.Data.ReviewCount)
}
