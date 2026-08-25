// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements distributed configuration consumer options.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast

import (
	"log/slog"
	"os"
	"strings"

	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

// Option customizes a Hazelcast Client during construction.
type Option func(*options)

type options struct {
	logger       *slog.Logger
	awsClient    *consumeraws.Client
	mapProvider  MapProvider
	sourceLoader SourceLoader
	starter      Starter
}

// WithLogger configures the structured logger used by Client.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithAWSClient configures the AWS consumer used by S3 and Secrets Manager sources.
func WithAWSClient(client *consumeraws.Client) Option {
	return func(options *options) {
		if client != nil {
			options.awsClient = client
		}
	}
}

// WithMapProvider configures an existing map provider.
//
// Use this option in tests or when application code owns the Hazelcast client lifecycle.
func WithMapProvider(provider MapProvider) Option {
	return func(options *options) {
		if provider != nil {
			options.mapProvider = provider
		}
	}
}

// WithSourceLoader configures a custom source loader.
func WithSourceLoader(loader SourceLoader) Option {
	return func(options *options) {
		if loader != nil {
			options.sourceLoader = loader
		}
	}
}

// WithStarter configures the function used to start a Hazelcast map provider.
func WithStarter(starter Starter) Option {
	return func(options *options) {
		if starter != nil {
			options.starter = starter
		}
	}
}

func defaultOptions() options {
	return options{
		logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		starter: startProvider,
	}
}

func normalizeConfig(config Config) Config {
	normalized := config
	normalized.DefaultMap = strings.TrimSpace(config.DefaultMap)
	normalized.Source = normalizeSource(config.Source)
	return normalized
}

func normalizeSource(source Source) Source {
	normalized := source
	normalized.Kind = SourceKind(strings.ToLower(strings.TrimSpace(string(source.Kind))))
	normalized.Path = strings.TrimSpace(source.Path)
	normalized.Bucket = strings.TrimSpace(source.Bucket)
	normalized.Key = strings.TrimSpace(source.Key)
	normalized.SecretID = strings.TrimSpace(source.SecretID)
	normalized.VersionID = strings.TrimSpace(source.VersionID)
	normalized.VersionStage = strings.TrimSpace(source.VersionStage)
	normalized.AWSRegion = strings.TrimSpace(source.AWSRegion)
	return normalized
}

func validateConfig(Config) error {
	return nil
}

func validateSource(source Source) error {
	switch source.Kind {
	case SourceDefault:
		return nil
	case SourceInline:
		if len(source.Data) == 0 {
			return InvalidConfigError{Field: "Source.Data", Reason: "is required for inline source"}
		}
	case SourceFile:
		if source.Path == "" {
			return InvalidConfigError{Field: "Source.Path", Reason: "is required for file source"}
		}
	case SourceS3:
		if source.Bucket == "" {
			return InvalidConfigError{Field: "Source.Bucket", Reason: "is required for s3 source"}
		}
		if source.Key == "" {
			return InvalidConfigError{Field: "Source.Key", Reason: "is required for s3 source"}
		}
	case SourceSecretsManager:
		if source.SecretID == "" {
			return InvalidConfigError{Field: "Source.SecretID", Reason: "is required for secretsmanager source"}
		}
	default:
		return InvalidConfigError{Field: "Source.Kind", Reason: "unsupported value"}
	}
	return nil
}
