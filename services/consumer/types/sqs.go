// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public SQS request and response contracts.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

// SQSSendInput describes a message send operation.
type SQSSendInput struct {
	// QueueURL is the target SQS queue URL.
	QueueURL string
	// Body is the message body.
	Body string
	// DelaySeconds optionally delays message visibility.
	DelaySeconds int32
	// MessageAttributes contains string message attributes.
	MessageAttributes map[string]string
}

// SQSSendOutput contains metadata returned after sending a message.
type SQSSendOutput struct {
	// MessageID is the SQS message identifier.
	MessageID string
}

// SQSReceiveInput describes a message receive operation.
type SQSReceiveInput struct {
	// QueueURL is the source SQS queue URL.
	QueueURL string
	// MaxNumberOfMessages limits how many messages are returned.
	MaxNumberOfMessages int32
	// WaitTimeSeconds enables long polling when greater than zero.
	WaitTimeSeconds int32
	// VisibilityTimeout optionally overrides message visibility timeout.
	VisibilityTimeout int32
	// AttributeNames lists system attributes to retrieve.
	AttributeNames []string
	// MessageAttributeNames lists custom message attributes to retrieve.
	MessageAttributeNames []string
}

// SQSReceiveOutput contains messages returned by SQS.
type SQSReceiveOutput struct {
	// Messages contains the received messages.
	Messages []SQSMessage
}

// SQSMessage contains the essential values of an SQS message.
type SQSMessage struct {
	// MessageID is the SQS message identifier.
	MessageID string
	// ReceiptHandle is required to delete the message after processing.
	ReceiptHandle string
	// Body is the message body.
	Body string
	// Attributes contains system attributes returned by SQS.
	Attributes map[string]string
	// MessageAttributes contains string custom message attributes returned by SQS.
	MessageAttributes map[string]string
}

// SQSDeleteInput describes a message delete operation.
type SQSDeleteInput struct {
	// QueueURL is the source SQS queue URL.
	QueueURL string
	// ReceiptHandle identifies the message to delete.
	ReceiptHandle string
}
