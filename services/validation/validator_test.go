// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// validation tests recursive struct validation behavior.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package validation_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	playground "github.com/go-playground/validator/v10"

	"github.com/raywall/go-core-sdk/services/validation"
	"github.com/raywall/go-core-sdk/services/validation/types"
)

type validationRequest struct {
	Name     string               `json:"name" validate:"required"`
	Email    string               `json:"email" validate:"required,email"`
	Customer validationCustomer   `json:"customer"`
	Items    []validationItem     `json:"items" validate:"required,min=1"`
	Approver *validationApprover  `json:"approver" validate:"required"`
	Metadata map[string]auditInfo `json:"metadata"`
}

type validationCustomer struct {
	Document string `json:"document" validate:"required,len=11"`
}

type validationItem struct {
	SKU      string `json:"sku" validate:"required"`
	Quantity int    `json:"quantity" validate:"min=1"`
}

type validationApprover struct {
	ID string `json:"id" validate:"required"`
}

type auditInfo struct {
	Owner string `json:"owner" validate:"required"`
}

// TestValidator_ValidateAcceptsValidStruct verifies a complete struct graph can
// pass validation.
func TestValidator_ValidateAcceptsValidStruct(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	err := validator.Validate(context.Background(), validRequest())

	if err != nil {
		t.Fatalf("Validator.Validate() error = %v, want nil", err)
	}
}

// TestValidator_ValidateReturnsAllRootFieldErrors verifies multiple root-level
// validation failures are returned in one error.
func TestValidator_ValidateReturnsAllRootFieldErrors(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	request := validRequest()
	request.Name = ""
	request.Email = "not-an-email"

	err := validator.Validate(context.Background(), request)
	validationErr := requireValidationError(t, err)

	assertField(t, validationErr, "validationRequest.name", "required")
	assertField(t, validationErr, "validationRequest.email", "email")
}

// TestValidator_ValidateReturnsNestedStructErrors verifies nested struct fields
// are reported with the full parent path.
func TestValidator_ValidateReturnsNestedStructErrors(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	request := validRequest()
	request.Customer.Document = "123"

	err := validator.Validate(context.Background(), request)
	validationErr := requireValidationError(t, err)

	assertField(t, validationErr, "validationRequest.customer.document", "len")
}

// TestValidator_ValidateWalksSliceElementsWithoutDive verifies child structs in
// slices are validated even when the slice tag does not use dive.
func TestValidator_ValidateWalksSliceElementsWithoutDive(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	request := validRequest()
	request.Items = []validationItem{{SKU: "", Quantity: 0}}

	err := validator.Validate(context.Background(), request)
	validationErr := requireValidationError(t, err)

	assertField(t, validationErr, "validationRequest.items[0].sku", "required")
	assertField(t, validationErr, "validationRequest.items[0].quantity", "min")
}

// TestValidator_ValidateWalksMapValues verifies child structs in map values are
// validated and reported with the map key in the namespace.
func TestValidator_ValidateWalksMapValues(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	request := validRequest()
	request.Metadata = map[string]auditInfo{"primary": {Owner: ""}}

	err := validator.Validate(context.Background(), request)
	validationErr := requireValidationError(t, err)

	assertField(t, validationErr, "validationRequest.metadata[primary].owner", "required")
}

// TestValidator_ValidateReportsRequiredPointer verifies required pointer fields
// are reported without attempting to traverse nil values.
func TestValidator_ValidateReportsRequiredPointer(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	request := validRequest()
	request.Approver = nil

	err := validator.Validate(context.Background(), request)
	validationErr := requireValidationError(t, err)

	assertField(t, validationErr, "validationRequest.approver", "required")
}

// TestValidator_ValidateRejectsInvalidInput verifies nil values are rejected as
// invalid input instead of being treated as validation failures.
func TestValidator_ValidateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)

	tests := []struct {
		name  string
		value any
	}{
		{name: "nil interface", value: nil},
		{name: "nil pointer", value: (*validationRequest)(nil)},
		{name: "scalar", value: "not-a-struct"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := validator.Validate(context.Background(), test.value)
			var invalidInput types.InvalidInputError
			if !errors.As(err, &invalidInput) {
				t.Fatalf("Validator.Validate() error = %T, want types.InvalidInputError", err)
			}
		})
	}
}

// TestValidator_WithValidationRegistersCustomRule verifies custom validation
// functions can be registered at construction time.
func TestValidator_WithValidationRegistersCustomRule(t *testing.T) {
	t.Parallel()

	type request struct {
		Code string `json:"code" validate:"startswithx"`
	}

	validator, err := validation.New(
		validation.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))),
		validation.WithValidation("startswithx", func(level playground.FieldLevel) bool {
			return len(level.Field().String()) > 0 && level.Field().String()[0] == 'x'
		}),
	)
	if err != nil {
		t.Fatalf("validation.New() error = %v", err)
	}

	err = validator.Validate(context.Background(), request{Code: "abc"})
	validationErr := requireValidationError(t, err)

	assertField(t, validationErr, "request.code", "startswithx")
}

func newTestValidator(t *testing.T) *validation.Validator {
	t.Helper()

	validator, err := validation.New(validation.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))))
	if err != nil {
		t.Fatalf("validation.New() error = %v", err)
	}
	return validator
}

func validRequest() validationRequest {
	return validationRequest{
		Name:  "Jane",
		Email: "jane@example.com",
		Customer: validationCustomer{
			Document: "12345678901",
		},
		Items: []validationItem{
			{SKU: "sku-1", Quantity: 1},
		},
		Approver: &validationApprover{ID: "approver-1"},
		Metadata: map[string]auditInfo{
			"primary": {Owner: "ops"},
		},
	}
}

func requireValidationError(t *testing.T, err error) *types.ValidationError {
	t.Helper()

	if err == nil {
		t.Fatal("Validator.Validate() error = nil, want validation error")
	}
	var validationErr *types.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validator.Validate() error = %T, want *types.ValidationError", err)
	}
	return validationErr
}

func assertField(t *testing.T, validationErr *types.ValidationError, namespace string, tag string) {
	t.Helper()

	for _, field := range validationErr.Fields {
		if reflect.DeepEqual([]string{field.Namespace, field.Tag}, []string{namespace, tag}) {
			return
		}
	}
	t.Fatalf("field (%q, %q) not found in %#v", namespace, tag, validationErr.Fields)
}
