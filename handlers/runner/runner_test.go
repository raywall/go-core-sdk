// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// runner tests handler runtime selection.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package runner_test

import (
	"context"
	"testing"

	"github.com/raywall/go-core-sdk/handlers"
	"github.com/raywall/go-core-sdk/handlers/runner"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

func TestNewSelectsHTTPRunner(t *testing.T) {
	t.Parallel()

	processor := handlers.ProcessorFunc(func(context.Context, handlers.Event) (handlers.Result, error) {
		return handlers.Result{}, nil
	})
	selected, err := runner.New(handlers.Config{Runtime: handlers.RuntimeHTTP}, processor)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if selected == nil {
		t.Fatal("selected runner is nil")
	}
}

func TestNewSelectsSQSWorkerWithInjectedClient(t *testing.T) {
	t.Parallel()

	processor := handlers.ProcessorFunc(func(context.Context, handlers.Event) (handlers.Result, error) {
		return handlers.Result{}, nil
	})
	selected, err := runner.New(
		handlers.Config{Runtime: handlers.RuntimeSQSWorker, SQSQueueURL: "queue-url"},
		processor,
		runner.WithSQSClient(fakeSQSClient{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if selected == nil {
		t.Fatal("selected runner is nil")
	}
}

type fakeSQSClient struct{}

func (fakeSQSClient) ReceiveSQS(context.Context, consumeraws.SQSReceiveInput) (consumeraws.SQSReceiveOutput, error) {
	return consumeraws.SQSReceiveOutput{}, nil
}

func (fakeSQSClient) DeleteSQS(context.Context, consumeraws.SQSDeleteInput) error {
	return nil
}
