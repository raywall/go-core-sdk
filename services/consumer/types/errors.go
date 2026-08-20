// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public consumer service errors.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidConfigError is returned when consumer configuration is missing or invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the configuration value is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid consumer configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid consumer configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid consumer configuration: %s: %s", e.Field, e.Reason)
}

// RESTError is returned when a REST request cannot be prepared or executed.
type RESTError struct {
	// Operation identifies the failed REST operation.
	Operation string
	// Err is the wrapped error returned by the HTTP stack or encoder.
	Err error
}

// Error implements the error interface.
func (e RESTError) Error() string {
	if e.Operation == "" {
		return "rest request failed"
	}
	if e.Err == nil {
		return "rest request failed: " + e.Operation
	}
	return fmt.Sprintf("rest request failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped REST error.
func (e RESTError) Unwrap() error {
	return e.Err
}

// TokenRequiredError is returned when a REST call requests token injection but
// no usable token provider or token value is available.
type TokenRequiredError struct {
	// Reason explains why the token could not be injected.
	Reason string
}

// Error implements the error interface.
func (e TokenRequiredError) Error() string {
	if e.Reason == "" {
		return "authorization token required"
	}
	return "authorization token required: " + e.Reason
}

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

// DecodeError is returned when a response body or AWS item cannot be decoded
// into the caller-provided target.
type DecodeError struct {
	// Operation identifies the decode operation.
	Operation string
	// Err is the wrapped decoder error.
	Err error
}

// Error implements the error interface.
func (e DecodeError) Error() string {
	if e.Operation == "" {
		return "decode failed"
	}
	if e.Err == nil {
		return "decode failed: " + e.Operation
	}
	return fmt.Sprintf("decode failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped decode error.
func (e DecodeError) Unwrap() error {
	return e.Err
}
