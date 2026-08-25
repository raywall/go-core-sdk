// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// handlers implements runtime-neutral event contracts.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package handlers

import "context"

// Runtime identifies the environment adapter used to execute a processor.
type Runtime string

const (
	// RuntimeLambda runs the processor through an AWS Lambda adapter.
	RuntimeLambda Runtime = "lambda"
	// RuntimeHTTP runs the processor through an HTTP server adapter for ECS, EC2 or EKS.
	RuntimeHTTP Runtime = "http"
	// RuntimeSQSWorker runs the processor through an SQS polling worker.
	RuntimeSQSWorker Runtime = "sqs-worker"
)

// Trigger identifies the inbound event family normalized by a runtime adapter.
type Trigger string

const (
	// TriggerJSON represents a generic JSON event envelope.
	TriggerJSON Trigger = "json"
	// TriggerHTTP represents an HTTP request carrying an event envelope.
	TriggerHTTP Trigger = "http"
	// TriggerS3 represents an S3 object notification.
	TriggerS3 Trigger = "s3"
	// TriggerSQS represents an SQS message batch.
	TriggerSQS Trigger = "sqs"
)

// Config defines runtime-neutral handler settings.
type Config struct {
	// Runtime selects the execution adapter.
	Runtime Runtime
	// Trigger selects the inbound event family expected by the adapter.
	Trigger Trigger
	// HTTPAddress is the listen address used by the HTTP adapter.
	HTTPAddress string
	// HTTPPath is the event endpoint path used by the HTTP adapter.
	HTTPPath string
	// SQSQueueURL is the source queue URL used by the SQS worker adapter.
	SQSQueueURL string
	// SQSMaxMessages limits messages fetched per polling cycle.
	SQSMaxMessages int32
	// SQSWaitTimeSeconds controls SQS long polling.
	SQSWaitTimeSeconds int32
	// SQSVisibilityTimeout optionally overrides message visibility during processing.
	SQSVisibilityTimeout int32
}

// Event is the normalized input delivered to application processors.
type Event struct {
	// Runtime identifies the runtime adapter that received the event.
	Runtime Runtime `json:"runtime,omitempty"`
	// Trigger identifies the normalized event family.
	Trigger Trigger `json:"trigger,omitempty"`
	// Records contains normalized event records.
	Records []Record `json:"records,omitempty"`
	// Metadata contains event-level metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Record is a normalized unit of work within an Event.
type Record struct {
	// ID identifies the record for acknowledgements or batch failures.
	ID string `json:"id,omitempty"`
	// Source identifies the platform source, such as aws:s3 or aws:sqs.
	Source string `json:"source,omitempty"`
	// Body contains the record payload when the source has a message body.
	Body []byte `json:"body,omitempty"`
	// Metadata contains record-level metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
	// S3 contains S3 object metadata for S3 notifications.
	S3 *S3Object `json:"s3,omitempty"`
	// SQS contains SQS message metadata for SQS events.
	SQS *SQSMessage `json:"sqs,omitempty"`
}

// S3Object identifies an S3 object referenced by an event record.
type S3Object struct {
	// Bucket is the S3 bucket name.
	Bucket string `json:"bucket"`
	// Key is the S3 object key.
	Key string `json:"key"`
	// ETag is the object entity tag when present in the notification.
	ETag string `json:"etag,omitempty"`
	// Size is the object size in bytes when present in the notification.
	Size int64 `json:"size,omitempty"`
}

// SQSMessage contains normalized SQS metadata.
type SQSMessage struct {
	// MessageID is the SQS message identifier.
	MessageID string `json:"messageId,omitempty"`
	// ReceiptHandle is required by polling workers to delete the message.
	ReceiptHandle string `json:"receiptHandle,omitempty"`
	// Attributes contains SQS system attributes.
	Attributes map[string]string `json:"attributes,omitempty"`
	// MessageAttributes contains custom SQS message attributes.
	MessageAttributes map[string]string `json:"messageAttributes,omitempty"`
}

// Result describes the processing outcome returned by a Processor.
type Result struct {
	// Processed is the number of records successfully processed.
	Processed int `json:"processed"`
	// Failed is the number of records that failed.
	Failed int `json:"failed"`
	// Failures identifies records that should be retried or reported as failed.
	Failures []Failure `json:"failures,omitempty"`
	// Metadata contains result-level metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Failure identifies a failed record.
type Failure struct {
	// RecordID identifies the failed record.
	RecordID string `json:"recordId"`
	// Error contains a safe error message for logs or responses.
	Error string `json:"error,omitempty"`
}

// Processor handles normalized runtime events.
type Processor interface {
	// Process executes application behavior for the normalized event.
	Process(context.Context, Event) (Result, error)
}

// ProcessorFunc adapts a function to Processor.
type ProcessorFunc func(context.Context, Event) (Result, error)

// Process executes f as a Processor.
func (f ProcessorFunc) Process(ctx context.Context, event Event) (Result, error) {
	return f(ctx, event)
}

// Runner starts a runtime adapter.
type Runner interface {
	// Start runs the adapter until completion or context cancellation.
	Start(context.Context) error
}
