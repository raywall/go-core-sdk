// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// lambda implements AWS Lambda runner wiring.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package lambda

import (
	"context"

	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/raywall/go-core-sdk/handlers"
)

// Runner executes a Processor through AWS Lambda.
type Runner struct {
	config    handlers.Config
	processor handlers.Processor
}

// NewRunner constructs a Lambda runner.
func NewRunner(config handlers.Config, processor handlers.Processor) (*Runner, error) {
	normalized := handlers.NormalizeConfig(config)
	if normalized.Trigger == "" {
		return nil, handlers.InvalidConfigError{Field: "Trigger", Reason: "is required"}
	}
	if processor == nil {
		return nil, handlers.InvalidConfigError{Field: "Processor", Reason: "is required"}
	}
	return &Runner{config: normalized, processor: processor}, nil
}

// Start registers the configured Lambda handler and blocks until the Lambda runtime exits.
func (r *Runner) Start(ctx context.Context) error {
	switch r.config.Trigger {
	case handlers.TriggerS3:
		awslambda.StartWithContext(ctx, r.HandleS3)
	case handlers.TriggerSQS:
		awslambda.StartWithContext(ctx, r.HandleSQS)
	default:
		return handlers.InvalidConfigError{Field: "Trigger", Reason: "must be s3 or sqs for lambda runtime"}
	}
	return nil
}
