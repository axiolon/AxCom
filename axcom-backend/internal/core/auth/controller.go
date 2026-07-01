// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth provides user registration, credential validation, session tokens, and password recovery.
package auth

import (
	"ecom-engine/internal/core/auth/dto"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/response"
	"os"

	"github.com/gin-gonic/gin"
)

const errInvalidPayload = "Invalid request payload"

// Controller handles HTTP requests for user authentication and registration.
// Use NewController to initialize; zero value is invalid.
type Controller struct {
	service Service
}

// NewController creates a new Controller instance with the injected Service.
func NewController(service Service) *Controller {
	return &Controller{service: service}
}

// Register processes HTTP requests to register a new user.
func (c *Controller) Register(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Register request")

	var req dto.AuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(errInvalidPayload, err))
		return
	}

	if err := req.Validate(); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	if req.Role == "" {
		req.Role = "customer"
	}

	user, err := c.service.Register(ctx.Request.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, user)
}

// Login processes HTTP login requests.
func (c *Controller) Login(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Login request")

	var req dto.AuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(errInvalidPayload, err))
		return
	}

	if err := req.Validate(); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	session, err := c.service.Login(ctx.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, session)
}

// Logout processes HTTP logout requests by revoking the refresh token.
func (c *Controller) Logout(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Logout request")

	var req dto.LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(errInvalidPayload, err))
		return
	}

	if err := req.Validate(); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	err := c.service.Logout(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, map[string]string{"message": "logged out successfully"})
}

// Refresh processes HTTP token refresh requests.
func (c *Controller) Refresh(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Refresh request")

	var req dto.RefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(errInvalidPayload, err))
		return
	}

	if err := req.Validate(); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	session, err := c.service.RefreshSession(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, session)
}

// RequestPasswordReset processes requests to start a password reset flow.
func (c *Controller) RequestPasswordReset(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Password Reset Request")

	var req dto.PasswordResetRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(errInvalidPayload, err))
		return
	}

	if err := req.Validate(); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	resetToken, err := c.service.RequestPasswordReset(ctx.Request.Context(), req.Email)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	res := map[string]interface{}{
		"message":    "If the email address exists in our system, a password reset link has been generated. In production, this would be emailed.",
		"expires_at": resetToken.ExpiresAt,
	}

	// Gate token exposure by environment setup (only local/development)
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "local" || appEnv == "development" || appEnv == "" {
		res["reset_token"] = resetToken.Token
	}

	response.GinOK(ctx, res)
}

// ConfirmPasswordReset processes password reset completion.
func (c *Controller) ConfirmPasswordReset(ctx *gin.Context) {
	logger.InfoCtx(ctx.Request.Context(), "Received Password Reset Confirmation")

	var req dto.PasswordResetConfirmRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(errInvalidPayload, err))
		return
	}

	if err := req.Validate(); err != nil {
		response.GinWriteError(ctx, apperrors.NewBadRequest(err.Error(), err))
		return
	}

	err := c.service.ConfirmPasswordReset(ctx.Request.Context(), req.Token, req.NewPassword)
	if err != nil {
		response.GinWriteError(ctx, err)
		return
	}

	response.GinOK(ctx, map[string]string{"message": "password has been reset successfully"})
}
