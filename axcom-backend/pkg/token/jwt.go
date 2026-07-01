// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Claims represents the structure of the token payload.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

// JWTManager is a manager for signing and validating JWT tokens.
type JWTManager struct {
	secret []byte
}

// NewJWTManager creates a new JWTManager instance with a signing secret.
func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{secret: []byte(secret)}
}

// Generate creates a cryptographically signed token containing the userID and role.
// The duration parameter specifies the TTL (Time-To-Live) from the current time.
func (m *JWTManager) Generate(userID string, role string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		Exp:    time.Now().Add(duration).Unix(),
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig := m.sign(payload)
	token := payload + "." + sig
	return token, nil
}

// Validate verifies the token signature and expiration, and extracts the claims.
func (m *JWTManager) Validate(tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}

	payload := parts[0]
	sig := parts[1]

	expectedSig := m.sign(payload)
	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(sig)) != 1 {
		return nil, errors.New("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}

	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

func (m *JWTManager) sign(payload string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
