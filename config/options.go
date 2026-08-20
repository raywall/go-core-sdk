// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// config implements loader and resolver options.
//
// This file is part of the Config bounded context within the Config service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package config

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Loader loads configuration from an external or computed source.
type Loader func(context.Context, Config) (Config, error)

// Resolver normalizes or validates a loaded Config.
type Resolver func(context.Context, *Config) error

// Option customizes configuration loading.
type Option func(*loadState)

type loadState struct {
	config    Config
	loaders   []Loader
	resolvers []Resolver
}

// WithServiceName sets the logical application service name.
func WithServiceName(name string) Option {
	return func(state *loadState) {
		state.config.serviceName = name
	}
}

// WithEnvironment sets the runtime environment name.
func WithEnvironment(environment string) Option {
	return func(state *loadState) {
		state.config.environment = environment
	}
}

// WithVersion sets the application version.
func WithVersion(version string) Option {
	return func(state *loadState) {
		state.config.version = version
	}
}

// WithAWSRegion sets the AWS region used by AWS-backed services.
func WithAWSRegion(region string) Option {
	return func(state *loadState) {
		state.config.awsRegion = region
	}
}

// WithHTTPTimeout sets the default outbound HTTP timeout.
func WithHTTPTimeout(timeout time.Duration) Option {
	return func(state *loadState) {
		state.config.httpTimeout = timeout
	}
}

// WithLogger sets the structured logger shared by core services.
func WithLogger(logger *slog.Logger) Option {
	return func(state *loadState) {
		if logger != nil {
			state.config.logger = logger
			state.config.customLogger = true
		}
	}
}

// WithAWSConfig sets a preloaded AWS config.
func WithAWSConfig(config aws.Config) Option {
	return func(state *loadState) {
		state.config.awsConfig = &config
	}
}

// WithAWSDefaultConfig loads AWS config using the AWS SDK default chain.
func WithAWSDefaultConfig() Option {
	return func(state *loadState) {
		state.loaders = append(state.loaders, loadAWSDefaultConfig)
	}
}

// WithCache sets shared cache configuration.
func WithCache(config CacheConfig) Option {
	return func(state *loadState) {
		state.config.cache = config
	}
}

// WithObservability sets shared observability configuration.
func WithObservability(config ObservabilityConfig) Option {
	return func(state *loadState) {
		state.config.observability = config.clone()
	}
}

// WithToken registers a named token manager configuration.
func WithToken(name string, config TokenConfig) Option {
	return func(state *loadState) {
		if state.config.tokens == nil {
			state.config.tokens = make(map[string]TokenConfig)
		}
		name = strings.TrimSpace(name)
		if name != "" {
			state.config.tokens[name] = config.clone()
		}
	}
}

// WithLoader appends a custom configuration loader.
func WithLoader(loader Loader) Option {
	return func(state *loadState) {
		if loader != nil {
			state.loaders = append(state.loaders, loader)
		}
	}
}

// WithResolver appends a custom configuration resolver.
func WithResolver(resolver Resolver) Option {
	return func(state *loadState) {
		if resolver != nil {
			state.resolvers = append(state.resolvers, resolver)
		}
	}
}
