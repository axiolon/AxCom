// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package auth provides user registration, credential validation, session tokens, and password recovery.
package auth

import (
	"context"
	"crypto/rand"
	"ecom-engine/internal/infra/db"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/token"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"ecom-engine/pkg/idgen"

	"golang.org/x/crypto/bcrypt"
)

// Service defines the business logic contract for user registration and authentication.
type Service interface {
	// Register creates a new user account with a hashed password.
	// It returns an AppError with code 409 Conflict if the email is already registered.
	Register(ctx context.Context, email, password, role string) (*User, error)

	// Login validates user credentials and issues a Session containing access + refresh tokens.
	// It returns an AppError with code 401 Unauthorized for bad credentials.
	Login(ctx context.Context, email, password string) (*Session, error)

	// Logout revokes the active refresh token.
	// It returns an AppError with code 401 Unauthorized for bad refresh token.
	Logout(ctx context.Context, refreshToken string) error

	// RefreshSession rotates the session and issues a new access token and refresh token.
	// It returns an AppError with code 401 Unauthorized for bad refresh token.
	RefreshSession(ctx context.Context, refreshToken string) (*Session, error)

	// RequestPasswordReset generates a temporary password reset token.
	RequestPasswordReset(ctx context.Context, email string) (*PasswordResetToken, error)

	// ConfirmPasswordReset changes a user's password using a reset token.
	ConfirmPasswordReset(ctx context.Context, token, newPassword string) error

	// SyncOIDCUser finds or auto-provisions a local user record from an OIDC token.
	// It matches by sub first, then email, then creates a new account if neither matches.
	SyncOIDCUser(ctx context.Context, sub, email, name, role string) (*User, error)
}

type authService struct {
	repo       UserRepository
	tokenRepo  TokenRepository
	jwtManager *token.JWTManager
	txManager  db.TransactionManager
}

// NewAuthService creates and returns an implementation of the Service interface.
func NewAuthService(repo UserRepository, tokenRepo TokenRepository, jwtManager *token.JWTManager, txManager db.TransactionManager) Service {
	return &authService{
		repo:       repo,
		tokenRepo:  tokenRepo,
		jwtManager: jwtManager,
		txManager:  txManager,
	}
}

// generateOpaqueToken generates a random secure hex string.
func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 7 * 24 * time.Hour
)

// Register registers a new user after hashing their password.
func (s *authService) Register(ctx context.Context, email, password, role string) (*User, error) {
	// Role validation: only allow 'customer' and 'merchant' self-registration, unless SKIP_RATE_LIMIT=true (development/load-test mode)
	if role != "customer" && role != "merchant" && (role != "admin" || os.Getenv("SKIP_RATE_LIMIT") != "true") {
		logger.ErrorCtx(ctx, "Registration failed: invalid role: %s", role)
		return nil, apperrors.NewBadRequest("invalid role choice", errors.New("registration role must be customer or merchant"))
	}

	now := time.Now()

	// Verify that the email is not already registered before creating the new user account.
	existingUser, err := s.repo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		logger.ErrorCtx(ctx, "Registration failed: email already exists for %s", email)
		return nil, apperrors.NewConflict("email already registered", ErrEmailAlreadyExists)
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		logger.ErrorCtx(ctx, "Registration failed: database check error: %v", err)
		return nil, apperrors.NewInternal("failed to check email availability", err)
	}

	// Hash the password using bcrypt to ensure credentials are not stored in plain text.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to hash password for %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to process password securely", err)
	}

	uid, err := idgen.Generate("usr_")
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to generate user identifier for %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to process registration", err)
	}

	user := &User{
		ID:        uid,
		Email:     email,
		Password:  string(hashedPassword),
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to create user record for %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to save user record", err)
	}

	logger.InfoCtx(ctx, "User registered successfully: %s", email)
	return user, nil
}

// Login validates user credentials and returns a Session.
func (s *authService) Login(ctx context.Context, email, password string) (*Session, error) {
	now := time.Now()
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		logger.ErrorCtx(ctx, "Login failed for %s: user not found: %v", email, err)
		return nil, apperrors.NewUnauthorized("invalid email or password", ErrInvalidCredentials)
	}

	// Check if account is currently locked out
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		logger.WarnCtx(ctx, "Login failed for %s: account is currently locked out", email)
		return nil, apperrors.NewUnauthorized("account is temporarily locked due to too many failed attempts", errors.New("account locked"))
	}

	// Verify that the password matches the stored bcrypt hash.
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		logger.ErrorCtx(ctx, "Login failed for %s: invalid password", email)
		user.FailedLoginAttempts++
		user.UpdatedAt = now
		if user.FailedLoginAttempts >= 5 {
			lockUntil := now.Add(15 * time.Minute)
			user.LockedUntil = &lockUntil
			logger.WarnCtx(ctx, "Account locked out for %s until %v", email, lockUntil)
		}
		if updateErr := s.repo.Update(ctx, user); updateErr != nil {
			logger.ErrorCtx(ctx, "Login failed: failed to update failed login attempts/lockout state for %s: %v", email, updateErr)
			return nil, apperrors.NewInternal("failed to update login attempts", updateErr)
		}
		return nil, apperrors.NewUnauthorized("invalid email or password", ErrInvalidCredentials)
	}

	// Reset lockout state on successful login
	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		user.UpdatedAt = now
		if updateErr := s.repo.Update(ctx, user); updateErr != nil {
			logger.ErrorCtx(ctx, "Login failed: failed to reset failed login attempts/lockout state for %s: %v", email, updateErr)
			return nil, apperrors.NewInternal("failed to reset login attempts", updateErr)
		}
	}

	accessToken, err := s.jwtManager.Generate(user.ID, user.Role, accessTokenDuration)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to generate JWT access token for %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to issue access token", err)
	}

	// Refresh Token has a validity duration of 7 days.
	refreshTokenVal, err := generateOpaqueToken()
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to generate refresh token for %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to generate secure refresh token", err)
	}

	shortLen := len(refreshTokenVal)
	if shortLen > 12 {
		shortLen = 12
	}
	rfToken := &RefreshToken{
		ID:        "rt_" + refreshTokenVal[:shortLen],
		UserID:    user.ID,
		Token:     refreshTokenVal,
		ExpiresAt: now.Add(refreshTokenDuration),
		CreatedAt: now,
		Revoked:   false,
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.enforceMaxSessions(txCtx, user.ID); innerErr != nil {
			return innerErr
		}
		return s.tokenRepo.SaveRefreshToken(txCtx, rfToken)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to save refresh token in repository for %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to persist session", err)
	}

	logger.InfoCtx(ctx, "User logged in successfully: %s", email)
	return &Session{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenVal,
		ExpiresAt:    now.Add(accessTokenDuration),
	}, nil
}

// Logout revokes the provided refresh token.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	err := s.tokenRepo.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenRevoked) || errors.Is(err, ErrTokenExpired) {
			logger.InfoCtx(ctx, "Logout: Token already revoked or expired: %v", err)
			return nil
		}
		logger.ErrorCtx(ctx, "Logout failed: %v", err)
		return apperrors.NewInternal("failed to revoke session token", err)
	}
	logger.InfoCtx(ctx, "Successfully logged out/revoked refresh token")
	return nil
}

// RefreshSession validates the refresh token and issues a new access token + refresh token.
func (s *authService) RefreshSession(ctx context.Context, refreshToken string) (*Session, error) {
	now := time.Now()
	rfToken, err := s.tokenRepo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to get refresh token: %v", err)
		return nil, apperrors.NewUnauthorized("invalid or expired refresh token", err)
	}

	shortLogToken := refreshToken
	if len(shortLogToken) > 12 {
		shortLogToken = shortLogToken[:12]
	}

	if rfToken.Revoked {
		logger.ErrorCtx(ctx, "Refresh failed: token revoked: %s", shortLogToken)
		return nil, apperrors.NewUnauthorized("refresh token has been revoked", ErrTokenRevoked)
	}

	if now.After(rfToken.ExpiresAt) {
		logger.ErrorCtx(ctx, "Refresh failed: token expired: %s", shortLogToken)
		return nil, apperrors.NewUnauthorized("refresh token has expired", ErrTokenExpired)
	}

	user, err := s.repo.GetByID(ctx, rfToken.UserID)
	if err != nil {
		logger.ErrorCtx(ctx, "User not found for refresh token: %s", rfToken.UserID)
		return nil, apperrors.NewUnauthorized("user not found", err)
	}

	// Rotate the session by generating new token first, saving it, and then revoking the old token.
	// This ensures a crash or db failure does not leave the user with a revoked token and no new token.
	accessToken, err := s.jwtManager.Generate(user.ID, user.Role, accessTokenDuration)
	if err != nil {
		return nil, apperrors.NewInternal("failed to issue access token", err)
	}

	// New Refresh Token has a validity duration of 7 days.
	newRefreshTokenVal, err := generateOpaqueToken()
	if err != nil {
		return nil, apperrors.NewInternal("failed to generate new refresh token", err)
	}

	newShortLen := len(newRefreshTokenVal)
	if newShortLen > 12 {
		newShortLen = 12
	}
	newRfToken := &RefreshToken{
		ID:        "rt_" + newRefreshTokenVal[:newShortLen],
		UserID:    user.ID,
		Token:     newRefreshTokenVal,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		Revoked:   false,
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.enforceMaxSessions(txCtx, user.ID); innerErr != nil {
			return innerErr
		}
		if innerErr := s.tokenRepo.SaveRefreshToken(txCtx, newRfToken); innerErr != nil {
			return innerErr
		}
		return s.tokenRepo.RevokeRefreshToken(txCtx, refreshToken)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to rotate refresh token: %v", err)
		return nil, apperrors.NewInternal("failed to rotate refresh token", err)
	}

	logger.InfoCtx(ctx, "Session refreshed successfully for user ID: %s", user.ID)
	return &Session{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenVal,
		ExpiresAt:    now.Add(accessTokenDuration),
	}, nil
}

// RequestPasswordReset generates a temporary password reset token for the email if found.
func (s *authService) RequestPasswordReset(ctx context.Context, email string) (*PasswordResetToken, error) {
	now := time.Now()
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			logger.InfoCtx(ctx, "Password reset request: email not registered but returning generic success to prevent user enumeration: %s", email)
			// Return a dummy reset token structure with an expiration time to satisfy the generic handler
			// and generic success messages.
			return &PasswordResetToken{
				ID:        "pr_dummy",
				UserID:    "usr_dummy",
				Token:     "dummy-token",
				ExpiresAt: now.Add(1 * time.Hour),
				Used:      false,
			}, nil
		}
		logger.ErrorCtx(ctx, "Password reset request failed: database error: %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to process password reset request", err)
	}

	resetTokenVal, err := generateOpaqueToken()
	if err != nil {
		return nil, apperrors.NewInternal("failed to generate reset token", err)
	}

	shortResetLen := len(resetTokenVal)
	if shortResetLen > 12 {
		shortResetLen = 12
	}
	resetToken := &PasswordResetToken{
		ID:        "pr_" + resetTokenVal[:shortResetLen],
		UserID:    user.ID,
		Token:     resetTokenVal,
		ExpiresAt: now.Add(1 * time.Hour), // Reset token is valid for 1 hour.
		Used:      false,
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.tokenRepo.InvalidateUserResetTokens(txCtx, user.ID); innerErr != nil {
			return innerErr
		}
		return s.tokenRepo.SavePasswordResetToken(txCtx, resetToken)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to save password reset token: %v", err)
		return nil, apperrors.NewInternal("failed to save reset token", err)
	}

	logger.InfoCtx(ctx, "Password reset token generated for user: %s", email)
	logger.InfoCtx(ctx, "[SIMULATED EMAIL] Sending password reset link to %s. Token: %s", email, resetTokenVal)
	return resetToken, nil
}

// ConfirmPasswordReset resets the user's password using the token and the new password.
func (s *authService) ConfirmPasswordReset(ctx context.Context, tokenVal, newPassword string) error {
	now := time.Now()
	resetToken, err := s.tokenRepo.GetPasswordResetToken(ctx, tokenVal)
	if err != nil {
		logger.ErrorCtx(ctx, "Reset confirmation failed: token not found: %v", err)
		return apperrors.NewBadRequest("invalid password reset token", ErrResetTokenInvalid)
	}

	if resetToken.Used {
		logger.ErrorCtx(ctx, "Reset confirmation failed: token already used")
		return apperrors.NewBadRequest("password reset token already used", ErrResetTokenUsed)
	}

	if now.After(resetToken.ExpiresAt) {
		logger.ErrorCtx(ctx, "Reset confirmation failed: token expired")
		return apperrors.NewBadRequest("password reset token expired", ErrTokenExpired)
	}

	// Securely hash the new password using bcrypt before updating storage.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.NewInternal("failed to hash password", err)
	}

	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.repo.UpdatePassword(txCtx, resetToken.UserID, string(hashedPassword)); innerErr != nil {
			return innerErr
		}
		if innerErr := s.tokenRepo.MarkPasswordResetUsed(txCtx, tokenVal); innerErr != nil {
			return innerErr
		}
		return s.tokenRepo.RevokeAllUserTokens(txCtx, resetToken.UserID)
	})
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to confirm password reset: %v", err)
		return apperrors.NewInternal("failed to secure password reset", err)
	}

	logger.InfoCtx(ctx, "Password reset successfully confirmed for user ID: %s", resetToken.UserID)
	return nil
}

func (s *authService) enforceMaxSessions(ctx context.Context, userID string) error {
	activeTokens, err := s.tokenRepo.GetActiveRefreshTokens(ctx, userID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to fetch active refresh tokens for user %s: %v", userID, err)
		return err
	}

	if len(activeTokens) >= 5 {
		// activeTokens is already sorted by created_at ASC from the DB query.
		excessCount := len(activeTokens) - 4
		for i := 0; i < excessCount; i++ {
			logger.InfoCtx(ctx, "Revoking oldest active session %s for user %s due to max session limit", activeTokens[i].ID, userID)
			if err := s.tokenRepo.RevokeRefreshToken(ctx, activeTokens[i].Token); err != nil {
				logger.ErrorCtx(ctx, "Failed to revoke oldest active session for user %s: %v", userID, err)
				return err
			}
		}
	}
	return nil
}

// SyncOIDCUser finds or auto-provisions a local user record from an OIDC token claims.
func (s *authService) SyncOIDCUser(ctx context.Context, sub, email, _, role string) (*User, error) {
	now := time.Now()
	// 1. Try finding user by OIDC sub
	user, err := s.repo.GetByOIDCSub(ctx, sub)
	if err == nil && user != nil {
		logger.DebugCtx(ctx, "SyncOIDCUser: Found user by OIDC sub %s", sub)
		return user, nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		logger.ErrorCtx(ctx, "SyncOIDCUser: Failed checking database for OIDC sub %s: %v", sub, err)
		return nil, apperrors.NewInternal("failed to sync user identity", err)
	}

	// 2. Try finding user by Email to link OIDC provider subject ID
	user, err = s.repo.GetByEmail(ctx, email)
	if err == nil && user != nil {
		logger.InfoCtx(ctx, "SyncOIDCUser: Linking existing user by email %s with OIDC sub %s", email, sub)
		user.OIDCSub = sub
		user.UpdatedAt = now
		err = s.repo.Update(ctx, user)
		if err != nil {
			logger.ErrorCtx(ctx, "SyncOIDCUser: Failed updating user sub link: %v", err)
			return nil, apperrors.NewInternal("failed to sync user identity association", err)
		}
		return user, nil
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		logger.ErrorCtx(ctx, "SyncOIDCUser: Failed checking database for email %s: %v", email, err)
		return nil, apperrors.NewInternal("failed to sync user identity", err)
	}

	// 3. Neither found, auto-provision user
	uid, err := idgen.Generate("usr_")
	if err != nil {
		logger.ErrorCtx(ctx, "SyncOIDCUser: Failed generating UUID: %v", err)
		return nil, apperrors.NewInternal("failed to provision user identity", err)
	}

	// Default to customer role if not provided or invalid
	if role == "" {
		role = "customer"
	}

	newUser := &User{
		ID:        uid,
		Email:     email,
		Password:  "", // No local password for OIDC users
		OIDCSub:   sub,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.repo.Create(ctx, newUser)
	if err != nil {
		logger.ErrorCtx(ctx, "SyncOIDCUser: Failed to create user: %v", err)
		return nil, apperrors.NewInternal("failed to create synchronized user record", err)
	}

	logger.InfoCtx(ctx, "SyncOIDCUser: Successfully provisioned new user %s with OIDC sub %s", email, sub)
	return newUser, nil
}
