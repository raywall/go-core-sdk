// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public validation service errors.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import (
	"errors"
	"fmt"
)

// ErrValidationFailed identifies errors produced by validation rule failures.
var ErrValidationFailed = errors.New("validation failed")

// InvalidInputError is returned when Validate receives a value that cannot be
// validated as a struct graph.
type InvalidInputError struct {
	// Reason explains why the input value cannot be validated.
	Reason string
}

// Error implements the error interface.
func (e InvalidInputError) Error() string {
	if e.Reason == "" {
		return "invalid validation input"
	}
	return "invalid validation input: " + e.Reason
}

// ValidationError is returned when one or more struct fields fail validation.
//
// The Fields slice contains every field that must be adjusted by the caller.
// ValidationError matches ErrValidationFailed through errors.Is.
type ValidationError struct {
	// Fields contains every invalid field found during validation.
	Fields []FieldError
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e == nil || len(e.Fields) == 0 {
		return ErrValidationFailed.Error()
	}
	if len(e.Fields) == 1 {
		return fmt.Sprintf("%s: 1 field requires adjustment", ErrValidationFailed)
	}
	return fmt.Sprintf("%s: %d fields require adjustment", ErrValidationFailed, len(e.Fields))
}

// Is reports whether target is ErrValidationFailed.
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidationFailed
}

// FieldErrors returns a copy of the invalid fields carried by the error.
func (e *ValidationError) FieldErrors() []FieldError {
	if e == nil {
		return nil
	}
	return append([]FieldError(nil), e.Fields...)
}
