// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// parser/types defines public parser errors.
//
// This file is part of the Parser bounded context within the Parser service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidInputError is returned when parser input cannot be converted.
type InvalidInputError struct {
	// Field identifies the invalid input field.
	Field string
	// Reason explains why the input is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidInputError) Error() string {
	if e.Field == "" {
		return "invalid parser input"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid parser input: %s", e.Field)
	}
	return fmt.Sprintf("invalid parser input: %s: %s", e.Field, e.Reason)
}

// EncodeError wraps JSON encoding failures for the source value.
type EncodeError struct {
	// Err is the wrapped JSON error.
	Err error
}

// Error implements the error interface.
func (e EncodeError) Error() string {
	if e.Err == nil {
		return "parser encode failed"
	}
	return fmt.Sprintf("parser encode failed: %v", e.Err)
}

// Unwrap returns the wrapped JSON error.
func (e EncodeError) Unwrap() error {
	return e.Err
}

// DecodeError wraps JSON decoding failures for the target value.
type DecodeError struct {
	// Err is the wrapped JSON error.
	Err error
}

// Error implements the error interface.
func (e DecodeError) Error() string {
	if e.Err == nil {
		return "parser decode failed"
	}
	return fmt.Sprintf("parser decode failed: %v", e.Err)
}

// Unwrap returns the wrapped JSON error.
func (e DecodeError) Unwrap() error {
	return e.Err
}
