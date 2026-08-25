// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// rest implements REST consumer error types.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package rest

import (
	"fmt"

	"github.com/raywall/go-core-sdk/services/consumer/internal/consumererrors"
)

// InvalidConfigError is returned when REST consumer configuration is invalid.
type InvalidConfigError = consumererrors.InvalidConfigError

// DecodeError is returned when a REST response cannot be decoded.
type DecodeError = consumererrors.DecodeError

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
