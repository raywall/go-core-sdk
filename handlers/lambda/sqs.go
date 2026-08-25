// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// lambda implements SQS event normalization.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package lambda

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/raywall/go-core-sdk/handlers"
)

// HandleSQS normalizes an AWS Lambda SQS event and returns batch item failures.
func (r *Runner) HandleSQS(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	result, err := r.processor.Process(ctx, NormalizeSQSEvent(event))
	response := events.SQSEventResponse{}
	for _, failure := range result.Failures {
		response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: failure.RecordID})
	}
	return response, err
}

// NormalizeSQSEvent converts an AWS Lambda SQS event to handlers.Event.
func NormalizeSQSEvent(event events.SQSEvent) handlers.Event {
	records := make([]handlers.Record, 0, len(event.Records))
	for _, record := range event.Records {
		records = append(records, handlers.Record{
			ID:     record.MessageId,
			Source: "aws:sqs",
			Body:   []byte(record.Body),
			SQS: &handlers.SQSMessage{
				MessageID:         record.MessageId,
				ReceiptHandle:     record.ReceiptHandle,
				Attributes:        record.Attributes,
				MessageAttributes: sqsMessageAttributes(record.MessageAttributes),
			},
		})
	}
	return handlers.Event{
		Runtime: handlers.RuntimeLambda,
		Trigger: handlers.TriggerSQS,
		Records: records,
	}
}

func sqsMessageAttributes(attributes map[string]events.SQSMessageAttribute) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	converted := make(map[string]string, len(attributes))
	for key, value := range attributes {
		if value.StringValue != nil {
			converted[key] = *value.StringValue
		}
	}
	return converted
}
