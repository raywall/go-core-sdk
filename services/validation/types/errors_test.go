// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types tests validation error behavior.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types_test

import (
	"errors"
	"testing"

	"github.com/raywall/go-core-sdk/services/validation/types"
)

// TestValidationError_Is verifies that ValidationError can be matched with
// ErrValidationFailed using errors.Is.
func TestValidationError_Is(t *testing.T) {
	t.Parallel()

	err := &types.ValidationError{Fields: []types.FieldError{{Namespace: "Request.name"}}}
	if !errors.Is(err, types.ErrValidationFailed) {
		t.Fatal("errors.Is(err, ErrValidationFailed) = false, want true")
	}
}

// TestValidationError_FieldErrorsCopiesSlice verifies callers cannot mutate the
// stored field list through FieldErrors.
func TestValidationError_FieldErrorsCopiesSlice(t *testing.T) {
	t.Parallel()

	err := &types.ValidationError{Fields: []types.FieldError{{Namespace: "Request.name"}}}
	fields := err.FieldErrors()
	fields[0].Namespace = "mutated"

	if got, want := err.Fields[0].Namespace, "Request.name"; got != want {
		t.Fatalf("ValidationError.Fields[0].Namespace = %q, want %q", got, want)
	}
}
