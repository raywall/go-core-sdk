// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public decision service errors.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidConfigError is returned when decision evaluation input is incomplete
// or invalid.
type InvalidConfigError struct {
	// Field identifies the invalid input field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid decision configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid decision configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid decision configuration: %s: %s", e.Field, e.Reason)
}

// CompilationError is returned when a CEL expression cannot be parsed or
// checked.
type CompilationError struct {
	// Expression is the CEL expression that failed compilation.
	Expression string
	// Err is the underlying CEL issue.
	Err error
}

// Error implements the error interface.
func (e CompilationError) Error() string {
	return fmt.Sprintf("decision expression compilation failed: %v", e.Err)
}

// Unwrap returns the underlying CEL compilation error.
func (e CompilationError) Unwrap() error {
	return e.Err
}

// EvaluationError is returned when a compiled CEL expression fails at runtime.
type EvaluationError struct {
	// RuleName identifies the rule being evaluated.
	RuleName string
	// Expression is the CEL expression that failed evaluation.
	Expression string
	// Err is the underlying CEL runtime error.
	Err error
}

// Error implements the error interface.
func (e EvaluationError) Error() string {
	if e.RuleName == "" {
		return fmt.Sprintf("decision expression evaluation failed: %v", e.Err)
	}
	return fmt.Sprintf("decision expression evaluation failed for %s: %v", e.RuleName, e.Err)
}

// Unwrap returns the underlying CEL evaluation error.
func (e EvaluationError) Unwrap() error {
	return e.Err
}

// NonBooleanResultError is returned when a CEL expression evaluates
// successfully but does not return a boolean value.
type NonBooleanResultError struct {
	// RuleName identifies the evaluated rule.
	RuleName string
	// Expression is the evaluated CEL expression.
	Expression string
	// Value is the non-boolean result.
	Value any
}

// Error implements the error interface.
func (e NonBooleanResultError) Error() string {
	if e.RuleName == "" {
		return fmt.Sprintf("decision expression returned non-boolean value %T", e.Value)
	}
	return fmt.Sprintf("decision expression %s returned non-boolean value %T", e.RuleName, e.Value)
}

// EntityConversionError is returned when an entity cannot be converted into a
// CEL-friendly activation value.
type EntityConversionError struct {
	// Name is the entity name supplied by the caller.
	Name string
	// Err is the underlying conversion error.
	Err error
}

// Error implements the error interface.
func (e EntityConversionError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("decision entity conversion failed: %v", e.Err)
	}
	return fmt.Sprintf("decision entity conversion failed for %s: %v", e.Name, e.Err)
}

// Unwrap returns the underlying entity conversion error.
func (e EntityConversionError) Unwrap() error {
	return e.Err
}
