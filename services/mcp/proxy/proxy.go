// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy implements the MCP proxy service facade.
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
	"net/url"

	"github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

// Proxy exposes HTTP-backed services through MCP-friendly tool metadata and
// invocation contracts.
//
// Proxy is safe for concurrent use after construction when the configured
// http.Client is safe for concurrent use.
type Proxy struct {
	config     Config
	baseURL    *url.URL
	logger     *slog.Logger
	httpClient *http.Client
	tools      map[string]types.Tool
}

// New constructs a Proxy service.
func New(config Config, configurers ...Option) (*Proxy, error) {
	normalized := normalizeConfig(config)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(normalized.BaseURL)
	if err != nil {
		return nil, types.InvalidConfigError{Field: "BaseURL", Reason: err.Error()}
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, types.InvalidConfigError{Field: "BaseURL", Reason: "must be an absolute URL"}
	}

	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: normalized.Timeout}
	}

	tools := make(map[string]types.Tool, len(normalized.Tools))
	for _, tool := range normalized.Tools {
		tools[tool.Name] = copyTool(tool)
	}

	return &Proxy{
		config:     normalized,
		baseURL:    baseURL,
		logger:     options.logger,
		httpClient: httpClient,
		tools:      tools,
	}, nil
}

// Config returns a copy of the normalized proxy configuration.
func (p *Proxy) Config() Config {
	if p == nil {
		return Config{}
	}
	config := p.config
	config.DefaultHeaders = copyStringMap(p.config.DefaultHeaders)
	config.Tools = copyTools(p.config.Tools)
	return config
}
