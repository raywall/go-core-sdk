// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// worker implements the SQS polling runner.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package worker

import (
	"context"
	"time"

	"github.com/raywall/go-core-sdk/handlers"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

const defaultPollInterval = 250 * time.Millisecond

// SQSClient defines the SQS operations required by Runner.
type SQSClient interface {
	// ReceiveSQS receives messages from an SQS queue.
	ReceiveSQS(context.Context, consumeraws.SQSReceiveInput) (consumeraws.SQSReceiveOutput, error)
	// DeleteSQS deletes a successfully processed message.
	DeleteSQS(context.Context, consumeraws.SQSDeleteInput) error
}

// Option customizes a worker Runner during construction.
type Option func(*Runner)

// WithPollInterval configures the delay between empty or completed polling cycles.
func WithPollInterval(interval time.Duration) Option {
	return func(runner *Runner) {
		if interval > 0 {
			runner.pollInterval = interval
		}
	}
}

// Runner executes a Processor through SQS polling.
type Runner struct {
	config       handlers.Config
	processor    handlers.Processor
	client       SQSClient
	pollInterval time.Duration
}

// NewSQSRunner constructs an SQS polling runner.
func NewSQSRunner(config handlers.Config, processor handlers.Processor, client SQSClient, options ...Option) (*Runner, error) {
	normalized := handlers.NormalizeConfig(config)
	if normalized.SQSQueueURL == "" {
		return nil, handlers.InvalidConfigError{Field: "SQSQueueURL", Reason: "is required"}
	}
	if processor == nil {
		return nil, handlers.InvalidConfigError{Field: "Processor", Reason: "is required"}
	}
	if client == nil {
		return nil, handlers.InvalidConfigError{Field: "SQSClient", Reason: "is required"}
	}
	runner := &Runner{
		config:       normalized,
		processor:    processor,
		client:       client,
		pollInterval: defaultPollInterval,
	}
	for _, option := range options {
		if option != nil {
			option(runner)
		}
	}
	return runner, nil
}

// Start polls SQS until ctx is cancelled.
func (r *Runner) Start(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.RunOnce(ctx); err != nil {
			return err
		}
		timer := time.NewTimer(r.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// RunOnce performs one SQS receive/process/delete cycle.
func (r *Runner) RunOnce(ctx context.Context) error {
	output, err := r.client.ReceiveSQS(ctx, consumeraws.SQSReceiveInput{
		QueueURL:              r.config.SQSQueueURL,
		MaxNumberOfMessages:   r.config.SQSMaxMessages,
		WaitTimeSeconds:       r.config.SQSWaitTimeSeconds,
		VisibilityTimeout:     r.config.SQSVisibilityTimeout,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		return err
	}
	if len(output.Messages) == 0 {
		return nil
	}

	event := normalizeSQSMessages(output.Messages)
	result, err := r.processor.Process(ctx, event)
	if err != nil {
		return err
	}

	failed := make(map[string]struct{}, len(result.Failures))
	for _, failure := range result.Failures {
		failed[failure.RecordID] = struct{}{}
	}
	for _, message := range output.Messages {
		if _, ok := failed[message.MessageID]; ok {
			continue
		}
		if err := r.client.DeleteSQS(ctx, consumeraws.SQSDeleteInput{
			QueueURL:      r.config.SQSQueueURL,
			ReceiptHandle: message.ReceiptHandle,
		}); err != nil {
			return err
		}
	}
	return nil
}

func normalizeSQSMessages(messages []consumeraws.SQSMessage) handlers.Event {
	records := make([]handlers.Record, 0, len(messages))
	for _, message := range messages {
		records = append(records, handlers.Record{
			ID:     message.MessageID,
			Source: "aws:sqs",
			Body:   []byte(message.Body),
			SQS: &handlers.SQSMessage{
				MessageID:         message.MessageID,
				ReceiptHandle:     message.ReceiptHandle,
				Attributes:        message.Attributes,
				MessageAttributes: message.MessageAttributes,
			},
		})
	}
	return handlers.Event{
		Runtime: handlers.RuntimeSQSWorker,
		Trigger: handlers.TriggerSQS,
		Records: records,
	}
}
