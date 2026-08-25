// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// lambda tests AWS Lambda handler adapters.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package lambda_test

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/raywall/go-core-sdk/handlers"
	handlerlambda "github.com/raywall/go-core-sdk/handlers/lambda"
)

func TestNormalizeS3Event(t *testing.T) {
	t.Parallel()

	event := handlerlambda.NormalizeS3Event(events.S3Event{
		Records: []events.S3EventRecord{
			{
				EventName: "ObjectCreated:Put",
				AWSRegion: "us-east-1",
				S3: events.S3Entity{
					Bucket: events.S3Bucket{Name: "docs"},
					Object: events.S3Object{Key: "a.txt", Size: 10, ETag: "etag"},
				},
			},
		},
	})

	if event.Runtime != handlers.RuntimeLambda || event.Trigger != handlers.TriggerS3 {
		t.Fatalf("event = %#v", event)
	}
	if got := event.Records[0].S3.Key; got != "a.txt" {
		t.Fatalf("S3 key = %q, want a.txt", got)
	}
}

func TestHandleSQSReturnsBatchItemFailures(t *testing.T) {
	t.Parallel()

	processor := handlers.ProcessorFunc(func(_ context.Context, event handlers.Event) (handlers.Result, error) {
		return handlers.Result{
			Processed: 1,
			Failed:    1,
			Failures: []handlers.Failure{
				{RecordID: event.Records[1].ID, Error: "boom"},
			},
		}, nil
	})
	runner, err := handlerlambda.NewRunner(handlers.Config{Trigger: handlers.TriggerSQS}, processor)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	response, err := runner.HandleSQS(context.Background(), events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "msg-1", Body: "ok"},
			{MessageId: "msg-2", Body: "fail"},
		},
	})
	if err != nil {
		t.Fatalf("HandleSQS() error = %v", err)
	}
	if len(response.BatchItemFailures) != 1 || response.BatchItemFailures[0].ItemIdentifier != "msg-2" {
		t.Fatalf("response = %#v", response)
	}
}
