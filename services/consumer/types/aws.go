// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public AWS client contracts for the consumer service.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// DynamoDBClient defines the DynamoDB operations used by Consumer.
type DynamoDBClient interface {
	// PutItem creates or replaces an item.
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	// UpdateItem updates an existing item and may return updated attributes.
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	// GetItem retrieves a single item by key.
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	// Query retrieves items matching a key condition expression.
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

// S3Client defines the S3 operations used by Consumer.
type S3Client interface {
	// PutObject stores an object in a bucket.
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	// GetObject retrieves an object from a bucket.
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// SecretsManagerClient defines the Secrets Manager operations used by Consumer.
type SecretsManagerClient interface {
	// GetSecretValue retrieves the current or requested value of a secret.
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	// PutSecretValue adds a new version value to an existing secret.
	PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	// UpdateSecret updates secret metadata and optionally creates a new secret value.
	UpdateSecret(context.Context, *secretsmanager.UpdateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
}

// SQSClient defines the SQS operations used by Consumer.
type SQSClient interface {
	// SendMessage sends a message to a queue.
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	// ReceiveMessage reads messages from a queue.
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	// DeleteMessage deletes a message from a queue by receipt handle.
	DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}
