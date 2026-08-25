// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent implements agent error types.
//
// This file is part of the Agent bounded context within the AI package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package agent

import "fmt"

// InvalidConfigError is returned when agent configuration is invalid.
type InvalidConfigError struct {
	// Field identifies the invalid field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid agent configuration"
	}
	if e.Reason == "" {
		return "invalid agent configuration: " + e.Field
	}
	return fmt.Sprintf("invalid agent configuration: %s %s", e.Field, e.Reason)
}

// ModelError is returned when a model API request fails.
type ModelError struct {
	// Operation identifies the failed model operation.
	Operation string
	// StatusCode is the HTTP status code when available.
	StatusCode int
	// Body contains a bounded response body when available.
	Body string
	// Err is the wrapped error.
	Err error
}

// Error implements the error interface.
func (e ModelError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("model request failed: %s: status %d: %s", e.Operation, e.StatusCode, e.Body)
	}
	if e.Err == nil {
		return "model request failed: " + e.Operation
	}
	return fmt.Sprintf("model request failed: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the wrapped model error.
func (e ModelError) Unwrap() error {
	return e.Err
}

// ToolError is returned when a tool cannot be executed.
type ToolError struct {
	// Name identifies the tool.
	Name string
	// Err is the wrapped tool error.
	Err error
}

// Error implements the error interface.
func (e ToolError) Error() string {
	if e.Err == nil {
		return "tool failed: " + e.Name
	}
	return fmt.Sprintf("tool failed: %s: %v", e.Name, e.Err)
}

// Unwrap returns the wrapped tool error.
func (e ToolError) Unwrap() error {
	return e.Err
}
