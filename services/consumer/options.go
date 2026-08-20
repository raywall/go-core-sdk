// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements configuration and construction options.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

const defaultHTTPTimeout = 30 * time.Second

// Config defines default behavior for outbound integrations.
type Config struct {
	// HTTPTimeout is used by the default HTTP client. A zero value uses a safe default.
	HTTPTimeout time.Duration
	// AWSRegion optionally pins the AWS region used when loading default AWS configuration.
	AWSRegion string
}

// Option customizes a Consumer during construction.
type Option func(*options)

type options struct {
	logger        *slog.Logger
	httpClient    *http.Client
	tokenProvider types.TokenProvider
	awsConfig     *aws.Config
	dynamoDB      types.DynamoDBClient
	s3            types.S3Client
	secrets       types.SecretsManagerClient
	sqs           types.SQSClient
}

// WithLogger configures the structured logger used by Consumer.
//
// The default logger writes JSON records to stdout. Passing nil keeps the default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithHTTPClient configures the HTTP client used by REST calls.
//
// This option is useful for tracing transports, proxies and httptest clients.
// Passing nil keeps the default HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(options *options) {
		if client != nil {
			options.httpClient = client
		}
	}
}

// WithTokenProvider configures the token provider used by REST calls with WithToken.
func WithTokenProvider(provider types.TokenProvider) Option {
	return func(options *options) {
		if provider != nil {
			options.tokenProvider = provider
		}
	}
}

// WithAWSConfig configures the AWS SDK configuration used to build default AWS clients.
func WithAWSConfig(config aws.Config) Option {
	return func(options *options) {
		options.awsConfig = &config
	}
}

// WithDynamoDBClient configures the DynamoDB client used by Consumer.
func WithDynamoDBClient(client types.DynamoDBClient) Option {
	return func(options *options) {
		if client != nil {
			options.dynamoDB = client
		}
	}
}

// WithS3Client configures the S3 client used by Consumer.
func WithS3Client(client types.S3Client) Option {
	return func(options *options) {
		if client != nil {
			options.s3 = client
		}
	}
}

// WithSecretsManagerClient configures the Secrets Manager client used by Consumer.
func WithSecretsManagerClient(client types.SecretsManagerClient) Option {
	return func(options *options) {
		if client != nil {
			options.secrets = client
		}
	}
}

// WithSQSClient configures the SQS client used by Consumer.
func WithSQSClient(client types.SQSClient) Option {
	return func(options *options) {
		if client != nil {
			options.sqs = client
		}
	}
}

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func normalizeConfig(config Config) Config {
	normalized := config
	normalized.AWSRegion = strings.TrimSpace(config.AWSRegion)
	if normalized.HTTPTimeout == 0 {
		normalized.HTTPTimeout = defaultHTTPTimeout
	}
	return normalized
}

func validateConfig(config Config) error {
	if config.HTTPTimeout < 0 {
		return types.InvalidConfigError{Field: "HTTPTimeout", Reason: "must not be negative"}
	}
	if config.HTTPTimeout == 0 {
		return types.InvalidConfigError{Field: "HTTPTimeout", Reason: "must be greater than zero"}
	}
	return nil
}
