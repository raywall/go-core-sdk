// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// config implements environment variable loading.
//
// This file is part of the Config bounded context within the Config service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package config

import (
	"context"
	"os"
	"strings"
	"time"
)

// WithEnv loads common configuration from environment variables.
//
// When prefix is APP, WithEnv reads APP_SERVICE_NAME, APP_ENVIRONMENT,
// APP_VERSION, APP_AWS_REGION, APP_HTTP_TIMEOUT, APP_OBSERVABILITY_PREFIX and
// APP_DATADOG_ADDR. AWS_REGION and AWS_DEFAULT_REGION are also considered when
// the prefixed AWS region is absent.
func WithEnv(prefix string) Option {
	return WithLoader(func(_ context.Context, cfg Config) (Config, error) {
		normalized := strings.Trim(strings.ToUpper(strings.TrimSpace(prefix)), "_")
		key := func(name string) string {
			if normalized == "" {
				return name
			}
			return normalized + "_" + name
		}

		cfg.serviceName = firstNonEmpty(os.Getenv(key("SERVICE_NAME")), cfg.serviceName)
		cfg.environment = firstNonEmpty(os.Getenv(key("ENVIRONMENT")), cfg.environment)
		cfg.version = firstNonEmpty(os.Getenv(key("VERSION")), cfg.version)
		cfg.awsRegion = firstNonEmpty(os.Getenv(key("AWS_REGION")), os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), cfg.awsRegion)
		cfg.observability.MetricPrefix = firstNonEmpty(os.Getenv(key("OBSERVABILITY_PREFIX")), cfg.observability.MetricPrefix)
		cfg.observability.DatadogAddress = firstNonEmpty(os.Getenv(key("DATADOG_ADDR")), cfg.observability.DatadogAddress)
		if value := strings.TrimSpace(os.Getenv(key("HTTP_TIMEOUT"))); value != "" {
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return Config{}, InvalidConfigError{Field: key("HTTP_TIMEOUT"), Reason: err.Error()}
			}
			cfg.httpTimeout = parsed
		}
		return cfg, nil
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
