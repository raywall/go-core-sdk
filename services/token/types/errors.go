// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public token service errors.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import (
	"errors"
	"fmt"
)

var (
	// ErrManagerAlreadyStarted is returned when Start is called while the
	// token manager already has an active background refresh loop.
	ErrManagerAlreadyStarted = errors.New("token manager already started")

	// ErrManagerStopped is returned when an operation requires a running token
	// manager but the manager is stopped.
	ErrManagerStopped = errors.New("token manager stopped")

	// ErrTokenUnavailable is returned when no token has been fetched yet.
	ErrTokenUnavailable = errors.New("token unavailable")
)

// InvalidConfigError is returned when the token manager configuration is
// missing a required field or contains an invalid value.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the field value is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid token configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid token configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid token configuration: %s: %s", e.Field, e.Reason)
}

// STSRequestError wraps transport-level failures returned while calling the
// configured STS endpoint.
type STSRequestError struct {
	// Operation identifies the HTTP operation that failed.
	Operation string
	// Err is the underlying transport or request construction error.
	Err error
}

// Error implements the error interface.
func (e STSRequestError) Error() string {
	if e.Operation == "" {
		return fmt.Sprintf("sts request failed: %v", e.Err)
	}
	return fmt.Sprintf("sts request failed during %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying transport error.
func (e STSRequestError) Unwrap() error {
	return e.Err
}

// STSResponseError is returned when the STS endpoint responds with an invalid
// HTTP status code or malformed token payload.
type STSResponseError struct {
	// StatusCode is the HTTP status code returned by STS. A zero value means
	// the response body could not be decoded after a successful HTTP status.
	StatusCode int
	// Body contains a small diagnostic excerpt of the response body.
	Body string
	// Err is the underlying decode or validation error when available.
	Err error
}

// Error implements the error interface.
func (e STSResponseError) Error() string {
	switch {
	case e.StatusCode != 0 && e.Body != "":
		return fmt.Sprintf("sts response failed: status=%d body=%q", e.StatusCode, e.Body)
	case e.StatusCode != 0:
		return fmt.Sprintf("sts response failed: status=%d", e.StatusCode)
	case e.Err != nil:
		return fmt.Sprintf("sts response failed: %v", e.Err)
	default:
		return "sts response failed"
	}
}

// Unwrap returns the underlying response parsing error.
func (e STSResponseError) Unwrap() error {
	return e.Err
}
