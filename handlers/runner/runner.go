// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// runner implements runtime selection.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package runner

import (
	"log/slog"
	"os"
	"time"

	"github.com/raywall/go-core-sdk/handlers"
	handlerhttp "github.com/raywall/go-core-sdk/handlers/http"
	handlerlambda "github.com/raywall/go-core-sdk/handlers/lambda"
	"github.com/raywall/go-core-sdk/handlers/worker"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

// Option customizes runtime selection.
type Option func(*options)

type options struct {
	logger       *slog.Logger
	sqsClient    worker.SQSClient
	pollInterval time.Duration
}

// WithLogger configures the logger used by default clients created by New.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithSQSClient configures the SQS client used by the sqs-worker runtime.
func WithSQSClient(client worker.SQSClient) Option {
	return func(options *options) {
		if client != nil {
			options.sqsClient = client
		}
	}
}

// WithPollInterval configures the delay between SQS polling cycles.
func WithPollInterval(interval time.Duration) Option {
	return func(options *options) {
		if interval > 0 {
			options.pollInterval = interval
		}
	}
}

// New selects a concrete handler runner from config.
func New(config handlers.Config, processor handlers.Processor, configurers ...Option) (handlers.Runner, error) {
	normalized := handlers.NormalizeConfig(config)
	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}

	switch normalized.Runtime {
	case handlers.RuntimeLambda:
		return handlerlambda.NewRunner(normalized, processor)
	case handlers.RuntimeHTTP:
		return handlerhttp.NewRunner(normalized, processor)
	case handlers.RuntimeSQSWorker:
		return newSQSWorker(normalized, processor, options)
	default:
		return nil, handlers.InvalidConfigError{Field: "Runtime", Reason: "must be lambda, http or sqs-worker"}
	}
}

// NewFromEnv loads handler configuration from environment variables and selects a runner.
func NewFromEnv(prefix string, processor handlers.Processor, configurers ...Option) (handlers.Runner, error) {
	return New(handlers.LoadConfigFromEnv(prefix), processor, configurers...)
}

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func newSQSWorker(config handlers.Config, processor handlers.Processor, options options) (handlers.Runner, error) {
	client := options.sqsClient
	if client == nil {
		awsClient, err := consumeraws.New(consumeraws.Config{}, consumeraws.WithLogger(options.logger))
		if err != nil {
			return nil, err
		}
		client = awsClient
	}
	workerOptions := []worker.Option{}
	if options.pollInterval > 0 {
		workerOptions = append(workerOptions, worker.WithPollInterval(options.pollInterval))
	}
	return worker.NewSQSRunner(config, processor, client, workerOptions...)
}
