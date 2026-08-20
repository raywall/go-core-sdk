// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// observability implements the service facade.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package observability

import (
	"log/slog"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/raywall/go-core-sdk/services/observability/types"
)

// Observability centralizes structured logging and custom metrics.
//
// Observability is safe to share after construction when the configured
// MetricsClient is safe to share.
type Observability struct {
	config  Config
	logger  *slog.Logger
	metrics types.MetricsClient
}

// New constructs an Observability service.
func New(config Config, configurers ...Option) (*Observability, error) {
	normalized := normalizeConfig(config)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}

	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}

	client := options.metricsClient
	if client == nil {
		statsdClient, err := statsd.New(normalized.DatadogAddress)
		if err != nil {
			return nil, err
		}
		client = statsdClient
	}

	return &Observability{
		config:  normalized,
		logger:  newLogger(normalized, options),
		metrics: client,
	}, nil
}

// Config returns a copy of the normalized observability configuration.
func (o *Observability) Config() Config {
	if o == nil {
		return Config{}
	}
	config := o.config
	config.DefaultTags = append([]string(nil), o.config.DefaultTags...)
	return config
}

// Logger returns the structured application logger.
func (o *Observability) Logger() *slog.Logger {
	if o == nil {
		return slog.Default()
	}
	return o.logger
}

// Close releases resources owned by the metrics client when supported.
func (o *Observability) Close() error {
	if o == nil || o.metrics == nil {
		return nil
	}
	closer, ok := o.metrics.(interface{ Close() error })
	if !ok {
		return nil
	}
	return closer.Close()
}
