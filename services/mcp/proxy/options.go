// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy implements configuration and construction options.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package proxy

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

const (
	defaultTimeout       = 30 * time.Second
	maxBackendErrorBytes = 4096
)

// Config defines the downstream HTTP service and MCP tool mappings.
type Config struct {
	// BaseURL is the base URL of the HTTP service behind API Gateway, a Lambda
	// URL, ECS, or a load balancer.
	BaseURL string
	// DefaultHeaders are sent with every downstream invocation.
	DefaultHeaders map[string]string
	// Timeout configures the default HTTP client timeout. A zero value uses a
	// safe default.
	Timeout time.Duration
	// Tools contains the MCP-friendly tool mappings exposed by this proxy.
	Tools []types.Tool
}

// Option customizes Proxy during construction.
type Option func(*options)

type options struct {
	logger     *slog.Logger
	httpClient *http.Client
}

// WithLogger configures the structured logger used by Proxy.
//
// The default logger writes JSON records to stdout. Passing nil keeps the
// default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithHTTPClient configures the client used for downstream service calls.
//
// Passing nil keeps the default client configured from Config.Timeout.
func WithHTTPClient(client *http.Client) Option {
	return func(options *options) {
		if client != nil {
			options.httpClient = client
		}
	}
}

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func normalizeConfig(config Config) Config {
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}
	config.DefaultHeaders = copyStringMap(config.DefaultHeaders)
	config.Tools = copyTools(config.Tools)
	for index := range config.Tools {
		config.Tools[index].Name = strings.TrimSpace(config.Tools[index].Name)
		config.Tools[index].Description = strings.TrimSpace(config.Tools[index].Description)
		config.Tools[index].Method = strings.ToUpper(strings.TrimSpace(config.Tools[index].Method))
		config.Tools[index].Path = strings.TrimSpace(config.Tools[index].Path)
		config.Tools[index].Headers = copyStringMap(config.Tools[index].Headers)
	}
	return config
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.BaseURL) == "" {
		return types.InvalidConfigError{Field: "BaseURL", Reason: "is required"}
	}
	if config.Timeout < 0 {
		return types.InvalidConfigError{Field: "Timeout", Reason: "must not be negative"}
	}
	if config.Timeout == 0 {
		return types.InvalidConfigError{Field: "Timeout", Reason: "must be greater than zero"}
	}
	seen := make(map[string]struct{}, len(config.Tools))
	for _, tool := range config.Tools {
		if tool.Name == "" {
			return types.InvalidConfigError{Field: "Tools.Name", Reason: "is required"}
		}
		if _, ok := seen[tool.Name]; ok {
			return types.InvalidConfigError{Field: "Tools.Name", Reason: "must be unique"}
		}
		seen[tool.Name] = struct{}{}
		if tool.Method == "" {
			return types.InvalidConfigError{Field: "Tools.Method", Reason: "is required"}
		}
		if tool.Path == "" {
			return types.InvalidConfigError{Field: "Tools.Path", Reason: "is required"}
		}
		if !jsonSchemaLooksValid(tool.InputSchema) {
			return types.InvalidConfigError{Field: "Tools.InputSchema", Reason: "must be valid JSON when configured"}
		}
	}
	return nil
}
