// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// rest implements REST client configuration.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package rest

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

const defaultHTTPTimeout = 30 * time.Second

// Config defines default behavior for outbound REST integrations.
type Config struct {
	// HTTPTimeout is used by the default HTTP client. A zero value uses a safe default.
	HTTPTimeout time.Duration
}

// Option customizes a REST Client during construction.
type Option func(*options)

type options struct {
	logger        *slog.Logger
	httpClient    *http.Client
	tokenProvider TokenProvider
}

// WithLogger configures the structured logger used by Client.
//
// The default logger writes JSON records to stdout. Passing nil keeps the default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithHTTPClient configures the HTTP client used by REST calls.
//
// This option is useful for tracing transports, proxies and httptest clients.
// Passing nil keeps the default HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(options *options) {
		if client != nil {
			options.httpClient = client
		}
	}
}

// WithTokenProvider configures the token provider used by REST calls with WithToken.
func WithTokenProvider(provider TokenProvider) Option {
	return func(options *options) {
		if provider != nil {
			options.tokenProvider = provider
		}
	}
}

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func normalizeConfig(config Config) Config {
	normalized := config
	if normalized.HTTPTimeout == 0 {
		normalized.HTTPTimeout = defaultHTTPTimeout
	}
	return normalized
}

func validateConfig(config Config) error {
	if config.HTTPTimeout < 0 {
		return InvalidConfigError{Field: "HTTPTimeout", Reason: "must not be negative"}
	}
	if config.HTTPTimeout == 0 {
		return InvalidConfigError{Field: "HTTPTimeout", Reason: "must be greater than zero"}
	}
	return nil
}
