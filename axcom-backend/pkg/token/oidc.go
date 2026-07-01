// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// OIDCClaims represents the fields we extract from the external IdP identity token.
type OIDCClaims struct {
	Subject string   `json:"sub"`
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	Roles   []string `json:"roles"`
}

// OIDCValidator fetches JWKS keys from the provider and validates external RS256/ES256 tokens.
type OIDCValidator struct {
	issuer   string
	audience string
	k        keyfunc.Keyfunc
}

// NewOIDCValidator initializes the Keyfunc JWKS cache and returns a validator.
func NewOIDCValidator(issuer, audience, jwksURL string) (*OIDCValidator, error) {
	if issuer == "" || audience == "" || jwksURL == "" {
		return nil, errors.New("OIDC validator requires issuer, audience, and jwksURL configuration")
	}

	// Create keyfunc with a default context that will manage background updates.
	k, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch/parse JWKS from URL %q: %w", jwksURL, err)
	}

	return &OIDCValidator{
		issuer:   issuer,
		audience: audience,
		k:        k,
	}, nil
}

// Validate parses the standard JWT token against the cached JWKS keys, verifying claims.
func (v *OIDCValidator) Validate(tokenString string) (*OIDCClaims, error) {
	var claims struct {
		jwt.RegisteredClaims
		Email string   `json:"email"`
		Name  string   `json:"name"`
		Roles []string `json:"roles"`
	}

	token, err := jwt.ParseWithClaims(tokenString, &claims, v.k.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse/validate external JWT token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("external JWT token is invalid")
	}

	iss, err := claims.GetIssuer()
	if err != nil || iss != v.issuer {
		return nil, fmt.Errorf("issuer mismatch: expected %q, got %q", v.issuer, iss)
	}

	auds, err := claims.GetAudience()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audience claim: %w", err)
	}
	audMatched := false
	for _, aud := range auds {
		if aud == v.audience {
			audMatched = true
			break
		}
	}
	if !audMatched {
		return nil, fmt.Errorf("audience mismatch: expected %q", v.audience)
	}

	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil || exp.Before(time.Now()) {
		return nil, errors.New("external JWT token is expired")
	}

	sub, err := claims.GetSubject()
	if err != nil || sub == "" {
		return nil, errors.New("missing or empty subject ('sub') claim")
	}

	return &OIDCClaims{
		Subject: sub,
		Email:   claims.Email,
		Name:    claims.Name,
		Roles:   claims.Roles,
	}, nil
}
