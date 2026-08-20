// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements the public concurrency-safe token entity.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import (
	"strings"
	"sync"
	"time"
)

// TokenSnapshot is an immutable copy of the current token values.
//
// Snapshot values are safe to retain and inspect without synchronisation. They
// are not updated when the manager refreshes the token; callers that need fresh
// values should call Token.Snapshot again.
type TokenSnapshot struct {
	// AccessToken is the bearer credential returned by STS.
	AccessToken string
	// TokenType is the token type returned by STS, usually Bearer.
	TokenType string
	// RequestedAt is the instant when the STS request was sent.
	RequestedAt time.Time
	// ExpiresAt is the instant when the token expires.
	ExpiresAt time.Time
	// ExpiresIn is the token lifetime returned by STS.
	ExpiresIn time.Duration
	// RefreshToken is the refresh token returned by STS when present.
	RefreshToken string
	// Scopes is the normalized list of scopes returned by STS.
	Scopes []string
	// Active indicates whether the token is active. When STS omits the field,
	// the token service treats the token as active.
	Active bool
}

// Token is the concurrency-safe token entity exposed by the token manager.
//
// Token is designed to be shared as a stable pointer. The manager updates this
// object in place after each successful refresh, and callers can read the latest
// values through the methods on Token. The zero value represents an unavailable
// token.
type Token struct {
	mu       sync.RWMutex
	snapshot TokenSnapshot
}

// NewToken constructs a token entity with the provided snapshot.
//
// The snapshot scopes are copied so later changes to the input slice do not
// mutate the token. The returned token is safe for concurrent reads and updates
// through its methods.
func NewToken(snapshot TokenSnapshot) *Token {
	token := &Token{}
	token.Replace(snapshot)
	return token
}

// Replace updates the token entity with a new snapshot.
//
// Replace copies the scopes slice before publishing the snapshot. It is safe to
// call while other goroutines read token values through Token methods.
func (t *Token) Replace(snapshot TokenSnapshot) {
	if t == nil {
		return
	}
	copied := snapshot
	copied.Scopes = append([]string(nil), snapshot.Scopes...)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.snapshot = copied
}

// Snapshot returns an immutable copy of the current token values.
//
// The returned scopes slice is copied. Mutating it does not affect the token
// stored by the manager.
func (t *Token) Snapshot() TokenSnapshot {
	if t == nil {
		return TokenSnapshot{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	copied := t.snapshot
	copied.Scopes = append([]string(nil), t.snapshot.Scopes...)
	return copied
}

// AccessToken returns the current access token value.
func (t *Token) AccessToken() string {
	return t.Snapshot().AccessToken
}

// TokenType returns the current token type value.
func (t *Token) TokenType() string {
	return t.Snapshot().TokenType
}

// RequestedAt returns the instant when the current token was requested.
func (t *Token) RequestedAt() time.Time {
	return t.Snapshot().RequestedAt
}

// ExpiresAt returns the instant when the current token expires.
func (t *Token) ExpiresAt() time.Time {
	return t.Snapshot().ExpiresAt
}

// ExpiresIn returns the lifetime of the current token.
func (t *Token) ExpiresIn() time.Duration {
	return t.Snapshot().ExpiresIn
}

// RefreshToken returns the current refresh token value.
func (t *Token) RefreshToken() string {
	return t.Snapshot().RefreshToken
}

// Scopes returns the current token scopes.
//
// The returned slice is copied and can be safely modified by the caller.
func (t *Token) Scopes() []string {
	return t.Snapshot().Scopes
}

// Active returns whether the current token is active.
func (t *Token) Active() bool {
	return t.Snapshot().Active
}

// ToString returns the Authorization header value for the current token.
//
// For a Bearer token, ToString returns a value in the form
// "Bearer eyJ...signature". If either the token type or access token is empty,
// ToString returns an empty string.
func (t *Token) ToString() string {
	snapshot := t.Snapshot()
	tokenType := strings.TrimSpace(snapshot.TokenType)
	accessToken := strings.TrimSpace(snapshot.AccessToken)
	if tokenType == "" || accessToken == "" {
		return ""
	}
	return tokenType + " " + accessToken
}
