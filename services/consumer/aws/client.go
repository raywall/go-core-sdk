// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// aws implements the AWS consumer client.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package aws

import (
	"context"
	"log/slog"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Client coordinates DynamoDB, S3, Secrets Manager and SQS integrations.
//
// Client is safe for concurrent use. Lazy AWS client construction is guarded
// by an internal mutex; the lock is not held while performing outbound I/O.
type Client struct {
	config Config
	logger *slog.Logger

	mu       sync.Mutex
	awsCfg   *awssdk.Config
	dynamoDB DynamoDBClient
	s3       S3Client
	secrets  SecretsManagerClient
	sqs      SQSClient
}

// New constructs an AWS Client from Config.
//
// AWS clients are created lazily on first use unless supplied by options.
func New(config Config, configurers ...Option) (*Client, error) {
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

	return &Client{
		config:   normalized,
		logger:   options.logger,
		awsCfg:   options.awsConfig,
		dynamoDB: options.dynamoDB,
		s3:       options.s3,
		secrets:  options.secrets,
		sqs:      options.sqs,
	}, nil
}

// Config returns a copy of the normalized AWS configuration.
func (c *Client) Config() Config {
	if c == nil {
		return Config{}
	}
	return c.config
}

func (c *Client) awsConfig(ctx context.Context) (awssdk.Config, error) {
	c.mu.Lock()
	if c.awsCfg != nil {
		cfg := *c.awsCfg
		c.mu.Unlock()
		return cfg, nil
	}
	c.mu.Unlock()

	loadOptions := []func(*awssdkconfig.LoadOptions) error{}
	if c.config.Region != "" {
		loadOptions = append(loadOptions, awssdkconfig.WithRegion(c.config.Region))
	}
	cfg, err := awssdkconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return awssdk.Config{}, err
	}

	c.mu.Lock()
	if c.awsCfg == nil {
		c.awsCfg = &cfg
	}
	cached := *c.awsCfg
	c.mu.Unlock()
	return cached, nil
}

func (c *Client) dynamoDBClient(ctx context.Context) (DynamoDBClient, error) {
	c.mu.Lock()
	if c.dynamoDB != nil {
		client := c.dynamoDB
		c.mu.Unlock()
		return client, nil
	}
	c.mu.Unlock()

	cfg, err := c.awsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg, func(options *dynamodb.Options) {
		if c.config.EndpointURL != "" {
			options.BaseEndpoint = awssdk.String(c.config.EndpointURL)
		}
	})

	c.mu.Lock()
	if c.dynamoDB == nil {
		c.dynamoDB = client
	}
	cached := c.dynamoDB
	c.mu.Unlock()
	return cached, nil
}

func (c *Client) s3Client(ctx context.Context) (S3Client, error) {
	c.mu.Lock()
	if c.s3 != nil {
		client := c.s3
		c.mu.Unlock()
		return client, nil
	}
	c.mu.Unlock()

	cfg, err := c.awsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		if c.config.EndpointURL != "" {
			options.BaseEndpoint = awssdk.String(c.config.EndpointURL)
		}
		options.UsePathStyle = c.config.S3UsePathStyle
	})

	c.mu.Lock()
	if c.s3 == nil {
		c.s3 = client
	}
	cached := c.s3
	c.mu.Unlock()
	return cached, nil
}

func (c *Client) secretsManagerClient(ctx context.Context) (SecretsManagerClient, error) {
	c.mu.Lock()
	if c.secrets != nil {
		client := c.secrets
		c.mu.Unlock()
		return client, nil
	}
	c.mu.Unlock()

	cfg, err := c.awsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := secretsmanager.NewFromConfig(cfg, func(options *secretsmanager.Options) {
		if c.config.EndpointURL != "" {
			options.BaseEndpoint = awssdk.String(c.config.EndpointURL)
		}
	})

	c.mu.Lock()
	if c.secrets == nil {
		c.secrets = client
	}
	cached := c.secrets
	c.mu.Unlock()
	return cached, nil
}

func (c *Client) sqsClient(ctx context.Context) (SQSClient, error) {
	c.mu.Lock()
	if c.sqs != nil {
		client := c.sqs
		c.mu.Unlock()
		return client, nil
	}
	c.mu.Unlock()

	cfg, err := c.awsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := sqs.NewFromConfig(cfg, func(options *sqs.Options) {
		if c.config.EndpointURL != "" {
			options.BaseEndpoint = awssdk.String(c.config.EndpointURL)
		}
	})

	c.mu.Lock()
	if c.sqs == nil {
		c.sqs = client
	}
	cached := c.sqs
	c.mu.Unlock()
	return cached, nil
}
