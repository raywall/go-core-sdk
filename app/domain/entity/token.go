// Package entity contains the pure business entities of the STS token
// management library. This package has zero external dependencies: no HTTP,
// no JSON, no framework code — only Go's standard library types that are
// part of the language itself (time).
package entity

import "time"

// TokenManagement is the aggregate that represents an STS access token and
// its lifecycle metadata. It is intentionally immutable from the outside:
// callers receive copies and must go through the domain constructors to
// build new instances.
type TokenManagement struct {
	AccessToken string
	TokenType   string
	IssuedAt    time.Time
	ExpiresAt   time.Time
}

// NewTokenManagement builds a TokenManagement that was just issued and will
// remain valid for expiresIn.
func NewTokenManagement(accessToken, tokenType string, expiresIn time.Duration) *TokenManagement {
	now := time.Now()
	return &TokenManagement{
		AccessToken: accessToken,
		TokenType:   tokenType,
		IssuedAt:    now,
		ExpiresAt:   now.Add(expiresIn),
	}
}

// IsExpired reports whether the token is no longer valid.
func (t *TokenManagement) IsExpired() bool {
	if t == nil {
		return true
	}
	return !time.Now().Before(t.ExpiresAt)
}

// IsNearExpiration reports whether the token will expire within the given
// threshold, i.e. whether it is time to proactively renew it.
func (t *TokenManagement) IsNearExpiration(threshold time.Duration) bool {
	if t == nil {
		return true
	}
	return !time.Now().Add(threshold).Before(t.ExpiresAt)
}

// BearerHeader formats the token as an HTTP Authorization header value.
func (t *TokenManagement) BearerHeader() string {
	tokenType := t.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return tokenType + " " + t.AccessToken
}
