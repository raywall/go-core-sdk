// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent implements construction options.
//
// This file is part of the Agent bounded context within the AI package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package agent

import (
	"log/slog"
	"net/http"
	"os"
)

// Option customizes Agent construction.
type Option func(*options)

type options struct {
	httpClient      *http.Client
	logger          *slog.Logger
	modelClient     ModelClient
	shortTermMemory Memory
	longTermMemory  Memory
	tools           []Tool
}

// WithHTTPClient configures the HTTP client used by the default model client.
func WithHTTPClient(client *http.Client) Option {
	return func(options *options) {
		if client != nil {
			options.httpClient = client
		}
	}
}

// WithLogger configures the structured logger used by Agent.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithModelClient configures a custom model client.
func WithModelClient(client ModelClient) Option {
	return func(options *options) {
		if client != nil {
			options.modelClient = client
		}
	}
}

// WithShortTermMemory configures conversation memory for recent messages.
func WithShortTermMemory(memory Memory) Option {
	return func(options *options) {
		options.shortTermMemory = memory
	}
}

// WithLongTermMemory configures memory for persistent context.
func WithLongTermMemory(memory Memory) Option {
	return func(options *options) {
		options.longTermMemory = memory
	}
}

// WithTools registers tools available to the model.
func WithTools(tools ...Tool) Option {
	return func(options *options) {
		options.tools = append(options.tools, tools...)
	}
}

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}
