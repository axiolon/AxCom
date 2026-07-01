// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"errors"
	"sync"

	"ecom-engine/internal/core/auth"
)

type MemAuthRepo struct {
	mu    sync.RWMutex
	users map[string]*auth.User
}

func NewMemAuthRepo() *MemAuthRepo {
	return &MemAuthRepo{users: make(map[string]*auth.User)}
}

func (r *MemAuthRepo) Create(_ context.Context, u *auth.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.Email] = u
	return nil
}

func (r *MemAuthRepo) GetByEmail(_ context.Context, email string) (*auth.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[email]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return u, nil
}

func (r *MemAuthRepo) GetByID(_ context.Context, id string) (*auth.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (r *MemAuthRepo) UpdatePassword(_ context.Context, userID string, hashedPassword string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == userID {
			u.Password = hashedPassword
			return nil
		}
	}
	return auth.ErrUserNotFound
}

func (r *MemAuthRepo) Update(_ context.Context, user *auth.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == user.ID {
			u.Email = user.Email
			u.Password = user.Password
			u.OIDCSub = user.OIDCSub
			u.Role = user.Role
			u.UpdatedAt = user.UpdatedAt
			u.FailedLoginAttempts = user.FailedLoginAttempts
			u.LockedUntil = user.LockedUntil
			return nil
		}
	}
	return auth.ErrUserNotFound
}

func (r *MemAuthRepo) GetByOIDCSub(_ context.Context, sub string) (*auth.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.OIDCSub == sub {
			return u, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

type MemTokenRepo struct {
	mu          sync.RWMutex
	refreshToks map[string]*auth.RefreshToken
	resetToks   map[string]*auth.PasswordResetToken
}

func NewMemTokenRepo() *MemTokenRepo {
	return &MemTokenRepo{
		refreshToks: make(map[string]*auth.RefreshToken),
		resetToks:   make(map[string]*auth.PasswordResetToken),
	}
}

func (r *MemTokenRepo) SaveRefreshToken(_ context.Context, token *auth.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshToks[token.Token] = token
	return nil
}

func (r *MemTokenRepo) GetRefreshToken(_ context.Context, tokenVal string) (*auth.RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.refreshToks[tokenVal]
	if !ok {
		return nil, errors.New("refresh token not found")
	}
	return t, nil
}

func (r *MemTokenRepo) RevokeRefreshToken(_ context.Context, tokenVal string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.refreshToks[tokenVal]
	if !ok {
		return errors.New("refresh token not found")
	}
	t.Revoked = true
	return nil
}

func (r *MemTokenRepo) RevokeAllUserTokens(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.refreshToks {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (r *MemTokenRepo) SavePasswordResetToken(_ context.Context, token *auth.PasswordResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetToks[token.Token] = token
	return nil
}

func (r *MemTokenRepo) GetPasswordResetToken(_ context.Context, tokenVal string) (*auth.PasswordResetToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.resetToks[tokenVal]
	if !ok {
		return nil, errors.New("reset token not found")
	}
	return t, nil
}

func (r *MemTokenRepo) MarkPasswordResetUsed(_ context.Context, tokenVal string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.resetToks[tokenVal]
	if !ok {
		return errors.New("reset token not found")
	}
	t.Used = true
	return nil
}
