// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumer implements SQS convenience operations.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package consumer

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/raywall/go-core-sdk/services/consumer/types"
)

// SendSQS sends a message to an SQS queue.
func (c *Consumer) SendSQS(ctx context.Context, input types.SQSSendInput) (types.SQSSendOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.sqsClient(ctx)
	if err != nil {
		return types.SQSSendOutput{}, types.SQSError{Operation: "load_config", Err: err}
	}
	request := &sqs.SendMessageInput{
		QueueUrl:          aws.String(strings.TrimSpace(input.QueueURL)),
		MessageBody:       aws.String(input.Body),
		DelaySeconds:      input.DelaySeconds,
		MessageAttributes: toSQSMessageAttributes(input.MessageAttributes),
	}
	c.logger.InfoContext(ctx, "consumer_sqs_send_started", "queue_url", sanitizeQueueURL(input.QueueURL))
	response, err := client.SendMessage(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_sqs_send_failed", "queue_url", sanitizeQueueURL(input.QueueURL), "error", err)
		return types.SQSSendOutput{}, types.SQSError{Operation: "send_message", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_sqs_send_completed", "queue_url", sanitizeQueueURL(input.QueueURL), "message_id", aws.ToString(response.MessageId))
	return types.SQSSendOutput{MessageID: aws.ToString(response.MessageId)}, nil
}

// ReceiveSQS receives messages from an SQS queue.
func (c *Consumer) ReceiveSQS(ctx context.Context, input types.SQSReceiveInput) (types.SQSReceiveOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.sqsClient(ctx)
	if err != nil {
		return types.SQSReceiveOutput{}, types.SQSError{Operation: "load_config", Err: err}
	}
	request := &sqs.ReceiveMessageInput{
		QueueUrl:              aws.String(strings.TrimSpace(input.QueueURL)),
		MaxNumberOfMessages:   input.MaxNumberOfMessages,
		WaitTimeSeconds:       input.WaitTimeSeconds,
		VisibilityTimeout:     input.VisibilityTimeout,
		AttributeNames:        toQueueAttributeNames(input.AttributeNames),
		MessageAttributeNames: input.MessageAttributeNames,
	}
	c.logger.InfoContext(ctx, "consumer_sqs_receive_started", "queue_url", sanitizeQueueURL(input.QueueURL), "wait_time_seconds", input.WaitTimeSeconds)
	response, err := client.ReceiveMessage(ctx, request)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_sqs_receive_failed", "queue_url", sanitizeQueueURL(input.QueueURL), "error", err)
		return types.SQSReceiveOutput{}, types.SQSError{Operation: "receive_message", Err: err}
	}
	messages := make([]types.SQSMessage, 0, len(response.Messages))
	for _, message := range response.Messages {
		messages = append(messages, types.SQSMessage{
			MessageID:         aws.ToString(message.MessageId),
			ReceiptHandle:     aws.ToString(message.ReceiptHandle),
			Body:              aws.ToString(message.Body),
			Attributes:        message.Attributes,
			MessageAttributes: fromSQSMessageAttributes(message.MessageAttributes),
		})
	}
	c.logger.InfoContext(ctx, "consumer_sqs_receive_completed", "queue_url", sanitizeQueueURL(input.QueueURL), "count", len(messages))
	return types.SQSReceiveOutput{Messages: messages}, nil
}

// DeleteSQS deletes a message from an SQS queue by receipt handle.
func (c *Consumer) DeleteSQS(ctx context.Context, input types.SQSDeleteInput) error {
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := c.sqsClient(ctx)
	if err != nil {
		return types.SQSError{Operation: "load_config", Err: err}
	}
	request := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(strings.TrimSpace(input.QueueURL)),
		ReceiptHandle: aws.String(input.ReceiptHandle),
	}
	c.logger.InfoContext(ctx, "consumer_sqs_delete_started", "queue_url", sanitizeQueueURL(input.QueueURL))
	if _, err := client.DeleteMessage(ctx, request); err != nil {
		c.logger.ErrorContext(ctx, "consumer_sqs_delete_failed", "queue_url", sanitizeQueueURL(input.QueueURL), "error", err)
		return types.SQSError{Operation: "delete_message", Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_sqs_delete_completed", "queue_url", sanitizeQueueURL(input.QueueURL))
	return nil
}

func toSQSMessageAttributes(attributes map[string]string) map[string]sqstypes.MessageAttributeValue {
	if len(attributes) == 0 {
		return nil
	}
	converted := make(map[string]sqstypes.MessageAttributeValue, len(attributes))
	for key, value := range attributes {
		converted[key] = sqstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(value),
		}
	}
	return converted
}

func fromSQSMessageAttributes(attributes map[string]sqstypes.MessageAttributeValue) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	converted := make(map[string]string, len(attributes))
	for key, value := range attributes {
		converted[key] = aws.ToString(value.StringValue)
	}
	return converted
}

func toQueueAttributeNames(values []string) []sqstypes.QueueAttributeName {
	if len(values) == 0 {
		return nil
	}
	converted := make([]sqstypes.QueueAttributeName, 0, len(values))
	for _, value := range values {
		converted = append(converted, sqstypes.QueueAttributeName(value))
	}
	return converted
}

func sanitizeQueueURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 && idx < len(trimmed)-1 {
		return trimmed[:idx+1] + "***"
	}
	return trimmed
}
