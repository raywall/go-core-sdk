// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// environment/types defines public environment service errors.
//
// This file is part of the Environment bounded context within the Environment
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidInputError is returned when an environment variable request is invalid.
type InvalidInputError struct {
	// Field identifies the invalid input field.
	Field string
	// Reason explains why the input is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidInputError) Error() string {
	if e.Field == "" {
		return "invalid environment input"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid environment input: %s", e.Field)
	}
	return fmt.Sprintf("invalid environment input: %s: %s", e.Field, e.Reason)
}

// VariableNotFoundError is returned when a required variable is absent.
type VariableNotFoundError struct {
	// Name is the missing environment variable name.
	Name string
}

// Error implements the error interface.
func (e VariableNotFoundError) Error() string {
	if e.Name == "" {
		return "environment variable not found"
	}
	return fmt.Sprintf("environment variable not found: %s", e.Name)
}
