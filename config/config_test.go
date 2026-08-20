// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// config tests shared runtime configuration loading.
//
// This file is part of the Config bounded context within the Config service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package config_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/config"
)

func TestLoad_EnvAndProjection(t *testing.T) {
	t.Setenv("APP_SERVICE_NAME", "orders-worker")
	t.Setenv("APP_ENVIRONMENT", "test")
	t.Setenv("APP_VERSION", "1.4.0")
	t.Setenv("APP_AWS_REGION", "us-east-1")
	t.Setenv("APP_HTTP_TIMEOUT", "7s")
	t.Setenv("APP_OBSERVABILITY_PREFIX", "orders")
	t.Setenv("APP_DATADOG_ADDR", "localhost:8126")

	cfg, err := config.Load(context.Background(),
		config.WithEnv("APP"),
		config.WithLogger(discardLogger()),
		config.WithObservability(config.ObservabilityConfig{
			DefaultTags: []string{"team:credit"},
		}),
		config.WithCache(config.CacheConfig{
			DefaultTTL:      2 * time.Minute,
			CleanupInterval: 30 * time.Second,
		}),
		config.WithToken("partner-api", config.TokenConfig{
			BaseURL:  "https://sts.example.com",
			Endpoint: "/oauth/token",
			Headers:  map[string]string{"X-App": "orders"},
			SecretID: "orders/partner-api",
		}),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ServiceName() != "orders-worker" || cfg.Environment() != "test" || cfg.AWSRegion() != "us-east-1" {
		t.Fatalf("unexpected cfg: service=%q env=%q region=%q", cfg.ServiceName(), cfg.Environment(), cfg.AWSRegion())
	}
	if cfg.Version() != "1.4.0" {
		t.Fatalf("Version() = %q, want 1.4.0", cfg.Version())
	}
	if got := cfg.Consumer().HTTPTimeout; got != 7*time.Second {
		t.Fatalf("Consumer.HTTPTimeout = %s, want 7s", got)
	}
	if _, ok := cfg.ConfiguredLogger(); !ok {
		t.Fatal("ConfiguredLogger() ok = false, want true")
	}
	observabilityConfig := cfg.Observability()
	if observabilityConfig.ServiceName != "orders-worker" {
		t.Fatalf("Observability.ServiceName = %q, want orders-worker", observabilityConfig.ServiceName)
	}
	if observabilityConfig.Environment != "test" {
		t.Fatalf("Observability.Environment = %q, want test", observabilityConfig.Environment)
	}
	if observabilityConfig.Version != "1.4.0" {
		t.Fatalf("Observability.Version = %q, want 1.4.0", observabilityConfig.Version)
	}
	if observabilityConfig.MetricPrefix != "orders" {
		t.Fatalf("Observability.MetricPrefix = %q, want orders", observabilityConfig.MetricPrefix)
	}
	if observabilityConfig.DatadogAddress != "localhost:8126" {
		t.Fatalf("Observability.DatadogAddress = %q, want localhost:8126", observabilityConfig.DatadogAddress)
	}
	if len(observabilityConfig.DefaultTags) != 1 || observabilityConfig.DefaultTags[0] != "team:credit" {
		t.Fatalf("Observability.DefaultTags = %v, want [team:credit]", observabilityConfig.DefaultTags)
	}
	if got := cfg.Cache().DefaultTTL; got != 2*time.Minute {
		t.Fatalf("Cache.DefaultTTL = %s, want 2m", got)
	}
	tokenConfig, ok := cfg.Token("partner-api")
	if !ok {
		t.Fatal("Token(partner-api) not found")
	}
	if tokenConfig.SecretClientIDKey != "client_id" || tokenConfig.SecretClientSecretKey != "client_secret" {
		t.Fatalf("secret keys = %q/%q", tokenConfig.SecretClientIDKey, tokenConfig.SecretClientSecretKey)
	}
}

func TestLoad_RejectsInvalidDuration(t *testing.T) {
	t.Setenv("APP_HTTP_TIMEOUT", "nope")
	_, err := config.Load(context.Background(), config.WithEnv("APP"))
	if err == nil {
		t.Fatal("Load error = nil, want invalid duration")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
