// Package sts implements the domain TokenProvider port against a real STS
// (Security Token Service) endpoint using the OAuth2 client-credentials
// shape: a POST with client_id/client_secret/scope that returns
// {access_token, token_type, expires_in}.
package sts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
)

// tokenResponse mirrors the JSON payload returned by a standard STS token
// endpoint. It intentionally never leaves this package.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// HTTPTokenProvider implements repository.TokenProvider by calling a real
// STS endpoint over HTTP.
type HTTPTokenProvider struct {
	client       *http.Client
	tokenURL     string
	clientID     string
	clientSecret string
	scope        string
}

// NewHTTPTokenProvider builds an HTTPTokenProvider. client may be nil, in
// which case a client with a sane default timeout is used.
func NewHTTPTokenProvider(client *http.Client, tokenURL, clientID, clientSecret, scope string) *HTTPTokenProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &HTTPTokenProvider{
		client:       client,
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		scope:        scope,
	}
}

// FetchToken requests a new access token from the STS using the
// client-credentials grant.
func (p *HTTPTokenProvider) FetchToken(ctx context.Context) (*entity.TokenManagement, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	if p.scope != "" {
		form.Set("scope", p.scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: building request: %v", errs.ErrTokenFetchFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrTokenFetchFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: reading response: %v", errs.ErrTokenFetchFailed, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: STS responded with status %d: %s", errs.ErrTokenFetchFailed, resp.StatusCode, string(body))
	}

	var parsed tokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: decoding response: %v", errs.ErrTokenFetchFailed, err)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("%w: STS response missing access_token", errs.ErrTokenFetchFailed)
	}
	if parsed.TokenType == "" {
		parsed.TokenType = "Bearer"
	}
	if parsed.ExpiresIn <= 0 {
		parsed.ExpiresIn = 300 // conservative fallback: 5 minutes
	}

	return entity.NewTokenManagement(parsed.AccessToken, parsed.TokenType, time.Duration(parsed.ExpiresIn)*time.Second), nil
}
