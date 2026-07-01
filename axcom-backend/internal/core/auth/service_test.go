// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"ecom-engine/pkg/token"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository implements UserRepository
type MockUserRepository struct {
	mu           sync.RWMutex
	usersByID    map[string]*User
	usersByEmail map[string]*User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		usersByID:    make(map[string]*User),
		usersByEmail: make(map[string]*User),
	}
}

func (m *MockUserRepository) Create(_ context.Context, user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usersByID[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *MockUserRepository) GetByEmail(_ context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, exists := m.usersByEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *MockUserRepository) GetByID(_ context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, exists := m.usersByID[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *MockUserRepository) UpdatePassword(_ context.Context, userID string, hashedPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, exists := m.usersByID[userID]
	if !exists {
		return ErrUserNotFound
	}
	u.Password = hashedPassword
	return nil
}

func (m *MockUserRepository) Update(_ context.Context, user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, exists := m.usersByID[user.ID]
	if !exists {
		return ErrUserNotFound
	}
	u.Email = user.Email
	u.Password = user.Password
	u.OIDCSub = user.OIDCSub
	u.Role = user.Role
	u.UpdatedAt = user.UpdatedAt
	u.FailedLoginAttempts = user.FailedLoginAttempts
	u.LockedUntil = user.LockedUntil
	return nil
}

func (m *MockUserRepository) GetByOIDCSub(_ context.Context, sub string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.usersByID {
		if u.OIDCSub == sub {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

// MockTokenRepository implements TokenRepository
type MockTokenRepository struct {
	mu                  sync.RWMutex
	refreshTokens       map[string]*RefreshToken
	passwordResetTokens map[string]*PasswordResetToken
}

func NewMockTokenRepository() *MockTokenRepository {
	return &MockTokenRepository{
		refreshTokens:       make(map[string]*RefreshToken),
		passwordResetTokens: make(map[string]*PasswordResetToken),
	}
}

func (m *MockTokenRepository) SaveRefreshToken(_ context.Context, token *RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshTokens[token.Token] = token
	return nil
}

func (m *MockTokenRepository) GetRefreshToken(_ context.Context, token string) (*RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, exists := m.refreshTokens[token]
	if !exists {
		return nil, errors.New("token not found")
	}
	return t, nil
}

func (m *MockTokenRepository) RevokeRefreshToken(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, exists := m.refreshTokens[token]
	if !exists {
		return errors.New("token not found")
	}
	t.Revoked = true
	return nil
}

func (m *MockTokenRepository) RevokeAllUserTokens(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.refreshTokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (m *MockTokenRepository) GetActiveRefreshTokens(_ context.Context, userID string) ([]*RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var active []*RefreshToken
	for _, t := range m.refreshTokens {
		if t.UserID == userID && !t.Revoked && t.ExpiresAt.After(time.Now()) {
			active = append(active, t)
		}
	}
	return active, nil
}

func (m *MockTokenRepository) SavePasswordResetToken(_ context.Context, token *PasswordResetToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.passwordResetTokens[token.Token] = token
	return nil
}

func (m *MockTokenRepository) GetPasswordResetToken(_ context.Context, token string) (*PasswordResetToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, exists := m.passwordResetTokens[token]
	if !exists {
		return nil, errors.New("token not found")
	}
	return t, nil
}

func (m *MockTokenRepository) MarkPasswordResetUsed(_ context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, exists := m.passwordResetTokens[token]
	if !exists {
		return errors.New("token not found")
	}
	t.Used = true
	return nil
}

func (m *MockTokenRepository) InvalidateUserResetTokens(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.passwordResetTokens {
		if t.UserID == userID {
			t.Used = true
		}
	}
	return nil
}

type MockTxManager struct{}

func (m *MockTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestRegister(t *testing.T) {
	t.Parallel()

	t.Run("register success", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "user@example.com"
		password := "Password123"
		role := "customer"

		user, err := service.Register(context.Background(), email, password, role)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, role, user.Role)

		// Verify hashed password
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		assert.NoError(t, err)
	})

	t.Run("register duplicate email conflict", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "user@example.com"
		password := "NewPassword123"
		role := "customer"

		_, err := service.Register(context.Background(), email, password, role)
		require.NoError(t, err)

		_, err = service.Register(context.Background(), email, password, role)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email already registered")
	})
}

func TestLogin(t *testing.T) {
	t.Parallel()

	t.Run("login success", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "login@example.com"
		password := "Password123"
		role := "customer"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		err := userRepo.Create(context.Background(), &User{
			ID:        "usr_test_123",
			Email:     email,
			Password:  string(hashedPassword),
			Role:      role,
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		session, err := service.Login(context.Background(), email, password)
		require.NoError(t, err)
		require.NotNil(t, session)
		assert.NotEmpty(t, session.AccessToken)
		assert.NotEmpty(t, session.RefreshToken)

		claims, err := jwtManager.Validate(session.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "usr_test_123", claims.UserID)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("login invalid credentials - wrong password", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "login@example.com"
		password := "Password123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		err := userRepo.Create(context.Background(), &User{
			ID:       "usr_test_123",
			Email:    email,
			Password: string(hashedPassword),
		})
		require.NoError(t, err)

		_, err = service.Login(context.Background(), email, "WrongPassword12")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email or password")
	})

	t.Run("login invalid credentials - wrong email", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		_, err := service.Login(context.Background(), "unknown@example.com", "Password123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email or password")
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("logout success", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		token := "some_refresh_token_string"
		err := tokenRepo.SaveRefreshToken(context.Background(), &RefreshToken{
			ID:        "rt_123",
			UserID:    "usr_test_123",
			Token:     token,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		})
		require.NoError(t, err)

		err = service.Logout(context.Background(), token)
		require.NoError(t, err)

		rt, err := tokenRepo.GetRefreshToken(context.Background(), token)
		require.NoError(t, err)
		assert.True(t, rt.Revoked)
	})
}

func TestRefreshSession(t *testing.T) {
	t.Parallel()

	t.Run("refresh success", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		err := userRepo.Create(context.Background(), &User{
			ID:    "usr_test_123",
			Email: "refresh@example.com",
			Role:  "customer",
		})
		require.NoError(t, err)

		activeToken := "active_refresh_token"
		err = tokenRepo.SaveRefreshToken(context.Background(), &RefreshToken{
			ID:        "rt_active",
			UserID:    "usr_test_123",
			Token:     activeToken,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		})
		require.NoError(t, err)

		session, err := service.RefreshSession(context.Background(), activeToken)
		require.NoError(t, err)
		require.NotNil(t, session)
		assert.NotEmpty(t, session.AccessToken)
		assert.NotEmpty(t, session.RefreshToken)

		oldRt, err := tokenRepo.GetRefreshToken(context.Background(), activeToken)
		require.NoError(t, err)
		assert.True(t, oldRt.Revoked)

		newRt, err := tokenRepo.GetRefreshToken(context.Background(), session.RefreshToken)
		require.NoError(t, err)
		assert.NotNil(t, newRt)
	})

	t.Run("refresh fail - revoked token", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		revokedToken := "revoked_refresh_token"
		err := tokenRepo.SaveRefreshToken(context.Background(), &RefreshToken{
			ID:        "rt_revoked",
			UserID:    "usr_test_123",
			Token:     revokedToken,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   true,
		})
		require.NoError(t, err)

		_, err = service.RefreshSession(context.Background(), revokedToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refresh token has been revoked")
	})

	t.Run("refresh fail - expired token", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		expiredToken := "expired_refresh_token"
		err := tokenRepo.SaveRefreshToken(context.Background(), &RefreshToken{
			ID:        "rt_expired",
			UserID:    "usr_test_123",
			Token:     expiredToken,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			Revoked:   false,
		})
		require.NoError(t, err)

		_, err = service.RefreshSession(context.Background(), expiredToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refresh token has expired")
	})
}

func TestPasswordReset(t *testing.T) {
	t.Parallel()

	t.Run("request reset success", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "reset@example.com"
		err := userRepo.Create(context.Background(), &User{ //nolint:gosec
			ID:       "usr_test_123",
			Email:    email,
			Password: "old_hashed_password", //nolint:gosec // This is a mock password for testing
			Role:     "customer",
		})
		require.NoError(t, err)

		rt, err := service.RequestPasswordReset(context.Background(), email)
		require.NoError(t, err)
		require.NotNil(t, rt)
		assert.NotEmpty(t, rt.Token)
		assert.Equal(t, "usr_test_123", rt.UserID)
	})

	t.Run("request reset generic success - unknown email", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		res, err := service.RequestPasswordReset(context.Background(), "unknown@example.com")
		require.NoError(t, err)
		assert.Equal(t, "dummy-token", res.Token)
	})

	t.Run("confirm reset success", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "reset2@example.com"
		err := userRepo.Create(context.Background(), &User{ //nolint:gosec
			ID:       "usr_test_123",
			Email:    email,
			Password: "old_hashed_password", //nolint:gosec // This is a mock password for testing
			Role:     "customer",
		})
		require.NoError(t, err)

		rt, err := service.RequestPasswordReset(context.Background(), email)
		require.NoError(t, err)

		newPassword := "NewPassword123"
		err = service.ConfirmPasswordReset(context.Background(), rt.Token, newPassword)
		require.NoError(t, err)

		user, err := userRepo.GetByID(context.Background(), "usr_test_123")
		require.NoError(t, err)
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPassword))
		assert.NoError(t, err)

		usedToken, err := tokenRepo.GetPasswordResetToken(context.Background(), rt.Token)
		require.NoError(t, err)
		assert.True(t, usedToken.Used)
	})

	t.Run("confirm reset fail - token already used", func(t *testing.T) {
		t.Parallel()
		userRepo := NewMockUserRepository()
		tokenRepo := NewMockTokenRepository()
		jwtManager := token.NewJWTManager("secret_key_123")
		service := NewAuthService(userRepo, tokenRepo, jwtManager, &MockTxManager{})

		email := "reset3@example.com"
		err := userRepo.Create(context.Background(), &User{ //nolint:gosec
			ID:       "usr_test_123",
			Email:    email,
			Password: "old_hashed_password", //nolint:gosec // This is a mock password for testing
			Role:     "customer",
		})
		require.NoError(t, err)

		rt, err := service.RequestPasswordReset(context.Background(), email)
		require.NoError(t, err)

		err = service.ConfirmPasswordReset(context.Background(), rt.Token, "NewPassword123")
		require.NoError(t, err)

		// Try again
		err = service.ConfirmPasswordReset(context.Background(), rt.Token, "SomeOtherPassword123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password reset token already used")
	})
}
