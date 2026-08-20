// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// observability implements configuration and construction options.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/raywall/go-core-sdk/services/observability/types"
)

const (
	defaultDatadogAddress = "127.0.0.1:8125"
	defaultEnvironment    = "local"
	defaultMetricRate     = 1.0
)

// Config defines logging and metrics defaults.
type Config struct {
	// ServiceName is added to structured logs when configured.
	ServiceName string
	// Environment is added to structured logs and emitted as the env metric tag.
	Environment string
	// Version is added to structured logs when configured.
	Version string
	// MetricPrefix is prepended to every metric name.
	MetricPrefix string
	// DatadogAddress is the DogStatsD address used when no custom client is supplied.
	DatadogAddress string
	// DefaultTags are added to every metric.
	DefaultTags []string
	// LogLevel controls the minimum level for the default JSON logger.
	LogLevel slog.Level
}

// Option customizes Observability during construction.
type Option func(*options)

type options struct {
	metricsClient types.MetricsClient
	logger        *slog.Logger
	writer        io.Writer
}

// WithMetricsClient sets a custom metrics client.
func WithMetricsClient(client types.MetricsClient) Option {
	return func(options *options) {
		if client != nil {
			options.metricsClient = client
		}
	}
}

// WithLogger sets a custom structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithWriter sets the writer used by the default JSON logger.
func WithWriter(writer io.Writer) Option {
	return func(options *options) {
		if writer != nil {
			options.writer = writer
		}
	}
}

func defaultOptions() options {
	return options{
		writer: os.Stdout,
	}
}

func normalizeConfig(config Config) Config {
	config.ServiceName = strings.TrimSpace(config.ServiceName)
	config.Environment = defaultIfEmpty(config.Environment, defaultEnvironment)
	config.Version = strings.TrimSpace(config.Version)
	config.MetricPrefix = strings.Trim(strings.TrimSpace(config.MetricPrefix), ".")
	config.DatadogAddress = defaultIfEmpty(config.DatadogAddress, defaultDatadogAddress)
	config.DefaultTags = normalizeDefaultTags(config.DefaultTags)
	return config
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.DatadogAddress) == "" {
		return types.InvalidConfigError{Field: "DatadogAddress", Reason: "is required"}
	}
	return nil
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func normalizeDefaultTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			normalized = append(normalized, tag)
		}
	}
	return normalized
}
