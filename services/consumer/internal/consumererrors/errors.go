// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// consumererrors implements shared consumer error types.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

// Package consumererrors contains shared errors used by consumer subpackages.
package consumererrors

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
