// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public selector service errors.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidConfigError is returned when selector configuration is missing or
// invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the configuration value is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid selector configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid selector configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid selector configuration: %s: %s", e.Field, e.Reason)
}

// FieldNotFoundError is returned when a configured path cannot be resolved in
// an item.
type FieldNotFoundError struct {
	// Path is the configured field path.
	Path string
	// Segment is the path segment that could not be resolved.
	Segment string
}

// Error implements the error interface.
func (e FieldNotFoundError) Error() string {
	if e.Segment == "" {
		return fmt.Sprintf("selector field not found: %s", e.Path)
	}
	return fmt.Sprintf("selector field not found: %s at %s", e.Path, e.Segment)
}

// IncompatibleTypeError is returned when a field value cannot be used with the
// configured sort or selection kind.
type IncompatibleTypeError struct {
	// Path is the configured field path.
	Path string
	// Value is the incompatible value.
	Value any
	// Expected describes the expected value type.
	Expected string
}

// Error implements the error interface.
func (e IncompatibleTypeError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("selector incompatible type: expected %s, got %T", e.Expected, e.Value)
	}
	return fmt.Sprintf("selector incompatible type at %s: expected %s, got %T", e.Path, e.Expected, e.Value)
}

// WithPath returns a copy of the incompatible type error with Path populated
// when the current error does not already carry a path.
func (e IncompatibleTypeError) WithPath(path string) IncompatibleTypeError {
	if e.Path != "" {
		return e
	}
	e.Path = path
	return e
}

// InvalidAmountError is returned when an item amount or available amount is
// invalid for selection.
type InvalidAmountError struct {
	// Path is the configured amount path when the error came from an item.
	Path string
	// Value is the invalid amount value.
	Value any
	// Reason explains why the amount is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidAmountError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("invalid selector amount: %s", e.Reason)
	}
	return fmt.Sprintf("invalid selector amount at %s: %s", e.Path, e.Reason)
}
