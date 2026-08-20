// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// internal implements the STS HTTP client.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/raywall/go-core-sdk/services/token/types"
)

const maxErrorBodyBytes = 4096

// Credentials contains the client credentials sent to STS.
type Credentials struct {
	// ClientID is the OAuth client identifier.
	ClientID string
	// ClientSecret is the OAuth client secret.
	ClientSecret string
}

// ClientConfig configures the private STS HTTP client.
type ClientConfig struct {
	// BaseURL is the STS host URL.
	BaseURL string
	// Endpoint is the STS token endpoint path.
	Endpoint string
	// Headers contains additional headers sent with every STS token request.
	Headers map[string]string
	// ValidateSSL controls TLS certificate verification for HTTPS STS calls.
	ValidateSSL bool
	// RequestTimeout is applied to the default HTTP client when no custom
	// client is injected.
	RequestTimeout time.Duration
	// HTTPClient is an optional custom HTTP client used primarily by tests and
	// applications that already own transport configuration.
	HTTPClient *http.Client
}

// TokenResponse represents the JSON payload returned by STS.
type TokenResponse struct {
	// AccessToken is the access token returned by STS.
	AccessToken string `json:"access_token"`
	// TokenType is the token type returned by STS.
	TokenType string `json:"token_type"`
	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int64 `json:"expires_in"`
	// RefreshToken is the optional refresh token returned by STS.
	RefreshToken string `json:"refresh_token"`
	// Scope is the space-delimited list of scopes returned by STS.
	Scope string `json:"scope"`
	// Active indicates whether the returned token is active.
	Active *bool `json:"active"`
}

// Client performs STS token requests.
type Client struct {
	baseURL    *url.URL
	endpoint   string
	headers    map[string]string
	httpClient *http.Client
}

// NewClient constructs an STS HTTP client from configuration.
func NewClient(config ClientConfig) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil {
		return nil, types.InvalidConfigError{Field: "BaseURL", Reason: err.Error()}
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, types.InvalidConfigError{Field: "BaseURL", Reason: "must be an absolute URL"}
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if baseURL.Scheme == "https" && !config.ValidateSSL {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		}
		httpClient = &http.Client{
			Timeout:   config.RequestTimeout,
			Transport: transport,
		}
	}

	return &Client{
		baseURL:    baseURL,
		endpoint:   config.Endpoint,
		headers:    copyHeaders(config.Headers),
		httpClient: httpClient,
	}, nil
}

// FetchToken requests a new STS token using the client credentials grant.
func (c *Client) FetchToken(ctx context.Context, credentials Credentials) (TokenResponse, error) {
	endpointURL, err := c.endpointURL()
	if err != nil {
		return TokenResponse{}, err
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", credentials.ClientID)
	form.Set("client_secret", credentials.ClientSecret)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, types.STSRequestError{Operation: "build_request", Err: err}
	}
	for key, value := range c.headers {
		request.Header.Set(key, value)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return TokenResponse{}, types.STSRequestError{Operation: "post_token", Err: err}
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		return TokenResponse{}, types.STSResponseError{
			StatusCode: response.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		return TokenResponse{}, types.STSResponseError{Err: err}
	}
	if err := tokenResponse.Validate(); err != nil {
		return TokenResponse{}, types.STSResponseError{Err: err}
	}
	return tokenResponse, nil
}

// Validate checks whether the STS token response contains the required fields.
func (r TokenResponse) Validate() error {
	switch {
	case strings.TrimSpace(r.AccessToken) == "":
		return fmt.Errorf("access_token is required")
	case strings.TrimSpace(r.TokenType) == "":
		return fmt.Errorf("token_type is required")
	case r.ExpiresIn <= 0:
		return fmt.Errorf("expires_in must be greater than zero")
	default:
		return nil
	}
}

// ToSnapshot converts the STS response into a public token snapshot.
func (r TokenResponse) ToSnapshot(requestedAt time.Time) types.TokenSnapshot {
	active := true
	if r.Active != nil {
		active = *r.Active
	}
	expiresIn := time.Duration(r.ExpiresIn) * time.Second
	return types.TokenSnapshot{
		AccessToken:  strings.TrimSpace(r.AccessToken),
		TokenType:    strings.TrimSpace(r.TokenType),
		RequestedAt:  requestedAt,
		ExpiresAt:    requestedAt.Add(expiresIn),
		ExpiresIn:    expiresIn,
		RefreshToken: r.RefreshToken,
		Scopes:       strings.Fields(r.Scope),
		Active:       active,
	}
}

func (c *Client) endpointURL() (string, error) {
	endpoint, err := url.Parse(c.endpoint)
	if err != nil {
		return "", types.InvalidConfigError{Field: "Endpoint", Reason: err.Error()}
	}
	return c.baseURL.ResolveReference(endpoint).String(), nil
}

func copyHeaders(headers map[string]string) map[string]string {
	copied := make(map[string]string, len(headers))
	for key, value := range headers {
		copied[key] = value
	}
	return copied
}
