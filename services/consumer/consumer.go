// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements the service facade and shared client wiring.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

// Consumer coordinates outbound REST, DynamoDB, S3 and SQS integrations.
//
// Consumer is safe for concurrent use. Lazy AWS client construction is guarded
// by an internal mutex; the lock is not held while performing outbound I/O.
type Consumer struct {
	config        Config
	logger        *slog.Logger
	httpClient    *http.Client
	tokenProvider types.TokenProvider

	mu       sync.Mutex
	awsCfg   *aws.Config
	dynamoDB types.DynamoDBClient
	s3       types.S3Client
	sqs      types.SQSClient
}

// New constructs a Consumer from Config.
//
// The returned Consumer is ready for REST calls immediately. AWS clients are
// created lazily on first use unless supplied by options.
func New(config Config, configurers ...Option) (*Consumer, error) {
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

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: normalized.HTTPTimeout}
	}

	return &Consumer{
		config:        normalized,
		logger:        options.logger,
		httpClient:    httpClient,
		tokenProvider: options.tokenProvider,
		awsCfg:        options.awsConfig,
		dynamoDB:      options.dynamoDB,
		s3:            options.s3,
		sqs:           options.sqs,
	}, nil
}

func (c *Consumer) awsConfig(ctx context.Context) (aws.Config, error) {
	c.mu.Lock()
	if c.awsCfg != nil {
		cfg := *c.awsCfg
		c.mu.Unlock()
		return cfg, nil
	}
	c.mu.Unlock()

	loadOptions := []func(*config.LoadOptions) error{}
	if c.config.AWSRegion != "" {
		loadOptions = append(loadOptions, config.WithRegion(c.config.AWSRegion))
	}
	cfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return aws.Config{}, err
	}

	c.mu.Lock()
	if c.awsCfg == nil {
		c.awsCfg = &cfg
	}
	cached := *c.awsCfg
	c.mu.Unlock()
	return cached, nil
}

func (c *Consumer) dynamoDBClient(ctx context.Context) (types.DynamoDBClient, error) {
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
	client := dynamodb.NewFromConfig(cfg)

	c.mu.Lock()
	if c.dynamoDB == nil {
		c.dynamoDB = client
	}
	cached := c.dynamoDB
	c.mu.Unlock()
	return cached, nil
}

func (c *Consumer) s3Client(ctx context.Context) (types.S3Client, error) {
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
	client := s3.NewFromConfig(cfg)

	c.mu.Lock()
	if c.s3 == nil {
		c.s3 = client
	}
	cached := c.s3
	c.mu.Unlock()
	return cached, nil
}

func (c *Consumer) sqsClient(ctx context.Context) (types.SQSClient, error) {
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
	client := sqs.NewFromConfig(cfg)

	c.mu.Lock()
	if c.sqs == nil {
		c.sqs = client
	}
	cached := c.sqs
	c.mu.Unlock()
	return cached, nil
}
