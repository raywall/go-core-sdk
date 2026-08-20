// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy/types defines public MCP proxy errors.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidConfigError is returned when MCP proxy configuration is invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid mcp proxy config"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid mcp proxy config: %s", e.Field)
	}
	return fmt.Sprintf("invalid mcp proxy config: %s: %s", e.Field, e.Reason)
}

// InvalidInputError is returned when an invocation request is invalid.
type InvalidInputError struct {
	// Field identifies the invalid input field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidInputError) Error() string {
	if e.Field == "" {
		return "invalid mcp proxy input"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid mcp proxy input: %s", e.Field)
	}
	return fmt.Sprintf("invalid mcp proxy input: %s: %s", e.Field, e.Reason)
}

// ToolNotFoundError is returned when an invocation references an unknown tool.
type ToolNotFoundError struct {
	// Name is the missing tool name.
	Name string
}

// Error implements the error interface.
func (e ToolNotFoundError) Error() string {
	if e.Name == "" {
		return "mcp proxy tool not found"
	}
	return fmt.Sprintf("mcp proxy tool not found: %s", e.Name)
}

// EncodeError wraps failures while encoding invocation arguments.
type EncodeError struct {
	// ToolName identifies the tool being invoked.
	ToolName string
	// Err is the wrapped encoding error.
	Err error
}

// Error implements the error interface.
func (e EncodeError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("mcp proxy encode failed for %s", e.ToolName)
	}
	return fmt.Sprintf("mcp proxy encode failed for %s: %v", e.ToolName, e.Err)
}

// Unwrap returns the wrapped encoding error.
func (e EncodeError) Unwrap() error {
	return e.Err
}

// RequestError wraps failures while building or executing a downstream request.
type RequestError struct {
	// ToolName identifies the tool being invoked.
	ToolName string
	// Operation identifies the failed request operation.
	Operation string
	// Err is the wrapped request error.
	Err error
}

// Error implements the error interface.
func (e RequestError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("mcp proxy request %s failed for %s", e.Operation, e.ToolName)
	}
	return fmt.Sprintf("mcp proxy request %s failed for %s: %v", e.Operation, e.ToolName, e.Err)
}

// Unwrap returns the wrapped request error.
func (e RequestError) Unwrap() error {
	return e.Err
}

// BackendError is returned when the downstream service returns a non-2xx status.
type BackendError struct {
	// ToolName identifies the invoked tool.
	ToolName string
	// StatusCode is the downstream HTTP status code.
	StatusCode int
	// Body contains a bounded copy of the downstream error response body.
	Body string
}

// Error implements the error interface.
func (e BackendError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("mcp proxy backend failed for %s: status %d", e.ToolName, e.StatusCode)
	}
	return fmt.Sprintf("mcp proxy backend failed for %s: status %d: %s", e.ToolName, e.StatusCode, e.Body)
}
