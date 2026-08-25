// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// worker tests SQS polling handlers.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package worker_test

import (
	"context"
	"testing"

	"github.com/raywall/go-core-sdk/handlers"
	"github.com/raywall/go-core-sdk/handlers/worker"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

func TestRunOnceDeletesOnlySuccessfulMessages(t *testing.T) {
	t.Parallel()

	client := &fakeSQSClient{
		messages: []consumeraws.SQSMessage{
			{MessageID: "msg-1", ReceiptHandle: "receipt-1", Body: "ok"},
			{MessageID: "msg-2", ReceiptHandle: "receipt-2", Body: "fail"},
		},
	}
	processor := handlers.ProcessorFunc(func(_ context.Context, event handlers.Event) (handlers.Result, error) {
		return handlers.Result{
			Processed: 1,
			Failed:    1,
			Failures: []handlers.Failure{
				{RecordID: event.Records[1].ID, Error: "boom"},
			},
		}, nil
	})
	runner, err := worker.NewSQSRunner(handlers.Config{SQSQueueURL: "queue-url"}, processor, client)
	if err != nil {
		t.Fatalf("NewSQSRunner() error = %v", err)
	}

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if len(client.deleted) != 1 || client.deleted[0] != "receipt-1" {
		t.Fatalf("deleted = %#v, want receipt-1 only", client.deleted)
	}
}

type fakeSQSClient struct {
	messages []consumeraws.SQSMessage
	deleted  []string
}

func (f *fakeSQSClient) ReceiveSQS(context.Context, consumeraws.SQSReceiveInput) (consumeraws.SQSReceiveOutput, error) {
	return consumeraws.SQSReceiveOutput{Messages: f.messages}, nil
}

func (f *fakeSQSClient) DeleteSQS(_ context.Context, input consumeraws.SQSDeleteInput) error {
	f.deleted = append(f.deleted, input.ReceiptHandle)
	return nil
}
