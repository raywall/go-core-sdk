// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/handlers demonstrates runtime-neutral handler adapters.
//
// This file is part of the Handler sample bounded context within the Samples service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/raywall/go-core-sdk/handlers"
	handlerhttp "github.com/raywall/go-core-sdk/handlers/http"
	handlerlambda "github.com/raywall/go-core-sdk/handlers/lambda"
	"github.com/raywall/go-core-sdk/handlers/worker"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type sampleProcessor struct{}

func (sampleProcessor) Process(_ context.Context, event handlers.Event) (handlers.Result, error) {
	result := handlers.Result{Processed: len(event.Records)}
	for _, record := range event.Records {
		if record.ID == "msg-fail" {
			result.Processed--
			result.Failed++
			result.Failures = append(result.Failures, handlers.Failure{RecordID: record.ID, Error: "sample failure"})
		}
	}
	return result, nil
}

func run(ctx context.Context, out io.Writer) error {
	processor := sampleProcessor{}
	lambdaProcessed, err := runLambdaS3(ctx, processor)
	if err != nil {
		return err
	}
	httpProcessed, err := runHTTP(ctx, processor)
	if err != nil {
		return err
	}
	deleted, err := runSQSWorker(ctx, processor)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "lambdaS3Processed=%d httpProcessed=%d sqsWorkerDeleted=%d\n", lambdaProcessed, httpProcessed, deleted)
	return err
}

func runLambdaS3(ctx context.Context, processor handlers.Processor) (int, error) {
	runner, err := handlerlambda.NewRunner(handlers.Config{Trigger: handlers.TriggerS3}, processor)
	if err != nil {
		return 0, err
	}
	result, err := runner.HandleS3(ctx, events.S3Event{
		Records: []events.S3EventRecord{
			{
				EventName: "ObjectCreated:Put",
				S3: events.S3Entity{
					Bucket: events.S3Bucket{Name: "student-financing-inbox"},
					Object: events.S3Object{Key: "payments/instruction-001.json"},
				},
			},
		},
	})
	return result.Processed, err
}

func runHTTP(ctx context.Context, processor handlers.Processor) (int, error) {
	runner, err := handlerhttp.NewRunner(handlers.Config{Trigger: handlers.TriggerS3}, processor)
	if err != nil {
		return 0, err
	}
	server := httptest.NewServer(runner.Handler())
	defer server.Close()

	body, err := json.Marshal(handlers.Event{
		Records: []handlers.Record{
			{ID: "http-record", Source: "sample:http"},
		},
	})
	if err != nil {
		return 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/events", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	var result handlers.Result
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Processed, nil
}

func runSQSWorker(ctx context.Context, processor handlers.Processor) (int, error) {
	client := &fakeSQSClient{
		messages: []consumeraws.SQSMessage{
			{MessageID: "msg-ok", ReceiptHandle: "receipt-ok", Body: `{"ok":true}`},
			{MessageID: "msg-fail", ReceiptHandle: "receipt-fail", Body: `{"ok":false}`},
		},
	}
	runner, err := worker.NewSQSRunner(handlers.Config{SQSQueueURL: "https://sqs.us-east-1.amazonaws.com/123/sample"}, processor, client)
	if err != nil {
		return 0, err
	}
	if err := runner.RunOnce(ctx); err != nil {
		return 0, err
	}
	return len(client.deleted), nil
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
