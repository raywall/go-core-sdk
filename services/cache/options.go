// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// cache implements configuration and construction options.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package cache

import (
	"log/slog"
	"os"
	"time"

	cacheinternal "github.com/raywall/go-core-sdk/services/cache/internal"
	"github.com/raywall/go-core-sdk/services/cache/types"
)

const (
	defaultTTL             = 5 * time.Minute
	defaultCleanupInterval = time.Minute
)

// Config defines cache expiration behavior.
type Config struct {
	// DefaultTTL is used when Add does not receive an item-specific TTL. A zero
	// value uses a safe default.
	DefaultTTL time.Duration
	// CleanupInterval controls how often the automatic cleanup loop removes
	// expired records. A zero value uses a safe default.
	CleanupInterval time.Duration
}

// Option customizes a Cache during construction.
type Option func(*options)

type options struct {
	logger *slog.Logger
	clock  cacheinternal.Clock
}

// WithLogger configures the structured logger used by Cache.
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

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		clock:  cacheinternal.RealClock{},
	}
}

func withClock(clock cacheinternal.Clock) Option {
	return func(options *options) {
		if clock != nil {
			options.clock = clock
		}
	}
}

func normalizeConfig(config Config) Config {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = defaultTTL
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = defaultCleanupInterval
	}
	return config
}

func validateConfig(config Config) error {
	if config.DefaultTTL < 0 {
		return types.InvalidConfigError{Field: "DefaultTTL", Reason: "must not be negative"}
	}
	if config.CleanupInterval < 0 {
		return types.InvalidConfigError{Field: "CleanupInterval", Reason: "must not be negative"}
	}
	if config.DefaultTTL == 0 {
		return types.InvalidConfigError{Field: "DefaultTTL", Reason: "must be greater than zero"}
	}
	if config.CleanupInterval == 0 {
		return types.InvalidConfigError{Field: "CleanupInterval", Reason: "must be greater than zero"}
	}
	return nil
}
