// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// token implements configuration and options for the token manager.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package token

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	tokeninternal "github.com/raywall/go-core-sdk/services/token/internal"
	"github.com/raywall/go-core-sdk/services/token/types"
)

const (
	defaultRequestTimeout = 10 * time.Second
	defaultRefreshBefore  = 30 * time.Second
	defaultRetryBackoff   = 500 * time.Millisecond
	defaultMaxRetries     = 3
)

// Config defines how Manager connects to the STS token endpoint.
//
// Config is copied when NewManager is called. Later changes to the original
// value do not affect the manager.
type Config struct {
	// BaseURL is the STS base URL, such as https://sts.example.com.
	BaseURL string
	// Endpoint is the token endpoint path, such as /oauth/token.
	Endpoint string
	// Headers contains additional headers sent with every STS request.
	Headers map[string]string
	// ValidateSSL controls TLS certificate verification for HTTPS requests.
	ValidateSSL bool
	// ClientID is the OAuth client identifier used by client_credentials.
	ClientID string
	// ClientSecret is the OAuth client secret used by client_credentials.
	ClientSecret string
	// RefreshBefore is how long before expiry the manager should renew the
	// token. A zero value uses a safe default.
	RefreshBefore time.Duration
	// RequestTimeout is the timeout used by the default HTTP client. A zero
	// value uses a safe default.
	RequestTimeout time.Duration
	// MaxRetries is the number of retry attempts after an STS request failure.
	// A zero value uses a safe default.
	MaxRetries int
	// RetryBackoff is the initial delay between retry attempts. A zero value
	// uses a safe default, and later retries apply exponential backoff.
	RetryBackoff time.Duration
}

// Option customizes a Manager during construction.
type Option func(*managerOptions)

type managerOptions struct {
	logger     *slog.Logger
	httpClient *http.Client
	clock      tokeninternal.Clock
}

// WithLogger configures the structured logger used by Manager.
//
// The default logger writes JSON records to stdout. Passing nil keeps the
// default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *managerOptions) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithHTTPClient configures the HTTP client used for STS calls.
//
// This option is useful when applications need a custom transport, proxy,
// tracing middleware or test server client. Passing nil keeps the default HTTP
// client built from Config.
func WithHTTPClient(client *http.Client) Option {
	return func(options *managerOptions) {
		if client != nil {
			options.httpClient = client
		}
	}
}

func defaultOptions() managerOptions {
	return managerOptions{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		clock:  tokeninternal.RealClock{},
	}
}

func withClock(clock tokeninternal.Clock) Option {
	return func(options *managerOptions) {
		if clock != nil {
			options.clock = clock
		}
	}
}

func normalizeConfig(config Config) Config {
	normalized := config
	normalized.BaseURL = strings.TrimSpace(config.BaseURL)
	normalized.Endpoint = strings.TrimSpace(config.Endpoint)
	normalized.ClientID = strings.TrimSpace(config.ClientID)
	normalized.ClientSecret = strings.TrimSpace(config.ClientSecret)
	normalized.Headers = copyHeaders(config.Headers)
	if normalized.RequestTimeout <= 0 {
		normalized.RequestTimeout = defaultRequestTimeout
	}
	if normalized.RefreshBefore <= 0 {
		normalized.RefreshBefore = defaultRefreshBefore
	}
	if normalized.MaxRetries <= 0 {
		normalized.MaxRetries = defaultMaxRetries
	}
	if normalized.RetryBackoff <= 0 {
		normalized.RetryBackoff = defaultRetryBackoff
	}
	return normalized
}

func validateConfig(config Config) error {
	switch {
	case config.BaseURL == "":
		return types.InvalidConfigError{Field: "BaseURL", Reason: "is required"}
	case config.Endpoint == "":
		return types.InvalidConfigError{Field: "Endpoint", Reason: "is required"}
	case config.ClientID == "":
		return types.InvalidConfigError{Field: "ClientID", Reason: "is required"}
	case config.ClientSecret == "":
		return types.InvalidConfigError{Field: "ClientSecret", Reason: "is required"}
	case config.RefreshBefore < 0:
		return types.InvalidConfigError{Field: "RefreshBefore", Reason: "must not be negative"}
	case config.RequestTimeout < 0:
		return types.InvalidConfigError{Field: "RequestTimeout", Reason: "must not be negative"}
	case config.MaxRetries < 0:
		return types.InvalidConfigError{Field: "MaxRetries", Reason: "must not be negative"}
	case config.RetryBackoff < 0:
		return types.InvalidConfigError{Field: "RetryBackoff", Reason: "must not be negative"}
	default:
		return nil
	}
}

func copyHeaders(headers map[string]string) map[string]string {
	copied := make(map[string]string, len(headers))
	for key, value := range headers {
		copied[key] = value
	}
	return copied
}
