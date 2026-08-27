// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// aws implements AWS client configuration.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package aws

import (
	"log/slog"
	"os"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
)

// Config defines default behavior for AWS integrations.
type Config struct {
	// Region optionally pins the AWS region used when loading default AWS configuration.
	Region string
	// EndpointURL optionally overrides AWS service endpoints, useful for LocalStack.
	EndpointURL string
	// S3UsePathStyle forces S3 path-style addressing for local S3-compatible endpoints.
	S3UsePathStyle bool
}

// Option customizes an AWS Client during construction.
type Option func(*options)

type options struct {
	logger    *slog.Logger
	awsConfig *awssdk.Config
	dynamoDB  DynamoDBClient
	s3        S3Client
	secrets   SecretsManagerClient
	sqs       SQSClient
}

// WithLogger configures the structured logger used by Client.
//
// The default logger writes JSON records to stdout. Passing nil keeps the default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithConfig configures the AWS SDK configuration used to build default AWS clients.
func WithConfig(config awssdk.Config) Option {
	return func(options *options) {
		options.awsConfig = &config
	}
}

// WithDynamoDBClient configures the DynamoDB client used by Client.
func WithDynamoDBClient(client DynamoDBClient) Option {
	return func(options *options) {
		if client != nil {
			options.dynamoDB = client
		}
	}
}

// WithS3Client configures the S3 client used by Client.
func WithS3Client(client S3Client) Option {
	return func(options *options) {
		if client != nil {
			options.s3 = client
		}
	}
}

// WithSecretsManagerClient configures the Secrets Manager client used by Client.
func WithSecretsManagerClient(client SecretsManagerClient) Option {
	return func(options *options) {
		if client != nil {
			options.secrets = client
		}
	}
}

// WithSQSClient configures the SQS client used by Client.
func WithSQSClient(client SQSClient) Option {
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
	normalized.Region = strings.TrimSpace(config.Region)
	normalized.EndpointURL = strings.TrimRight(strings.TrimSpace(config.EndpointURL), "/")
	return normalized
}

func validateConfig(Config) error {
	return nil
}
