// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// aws implements AWS consumer error types.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package aws

import (
	"fmt"

	"github.com/raywall/go-core-sdk/services/consumer/internal/consumererrors"
)

// InvalidConfigError is returned when AWS consumer configuration is invalid.
type InvalidConfigError = consumererrors.InvalidConfigError

// DecodeError is returned when an AWS response cannot be decoded.
type DecodeError = consumererrors.DecodeError

// DynamoDBError is returned when a DynamoDB operation fails.
type DynamoDBError struct {
	// Operation identifies the DynamoDB operation.
	Operation string
	// Err is the wrapped DynamoDB, AWS config or marshal error.
	Err error
}

// Error implements the error interface.
func (e DynamoDBError) Error() string {
	if e.Operation == "" {
		return "dynamodb operation failed"
	}
	if e.Err == nil {
		return "dynamodb operation failed: " + e.Operation
	}
	return fmt.Sprintf("dynamodb operation failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped DynamoDB error.
func (e DynamoDBError) Unwrap() error {
	return e.Err
}

// S3Error is returned when an S3 operation fails.
type S3Error struct {
	// Operation identifies the S3 operation.
	Operation string
	// Err is the wrapped S3, AWS config or body error.
	Err error
}

// Error implements the error interface.
func (e S3Error) Error() string {
	if e.Operation == "" {
		return "s3 operation failed"
	}
	if e.Err == nil {
		return "s3 operation failed: " + e.Operation
	}
	return fmt.Sprintf("s3 operation failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped S3 error.
func (e S3Error) Unwrap() error {
	return e.Err
}

// SecretsManagerError is returned when a Secrets Manager operation fails.
type SecretsManagerError struct {
	// Operation identifies the Secrets Manager operation.
	Operation string
	// Err is the wrapped Secrets Manager, AWS config or decode error.
	Err error
}

// Error implements the error interface.
func (e SecretsManagerError) Error() string {
	if e.Operation == "" {
		return "secrets manager operation failed"
	}
	if e.Err == nil {
		return "secrets manager operation failed: " + e.Operation
	}
	return fmt.Sprintf("secrets manager operation failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped Secrets Manager error.
func (e SecretsManagerError) Unwrap() error {
	return e.Err
}

// SQSError is returned when an SQS operation fails.
type SQSError struct {
	// Operation identifies the SQS operation.
	Operation string
	// Err is the wrapped SQS or AWS config error.
	Err error
}

// Error implements the error interface.
func (e SQSError) Error() string {
	if e.Operation == "" {
		return "sqs operation failed"
	}
	if e.Err == nil {
		return "sqs operation failed: " + e.Operation
	}
	return fmt.Sprintf("sqs operation failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped SQS error.
func (e SQSError) Unwrap() error {
	return e.Err
}
