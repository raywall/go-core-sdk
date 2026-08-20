// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements field-level validation metadata.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

// FieldError describes one invalid field found during struct validation.
type FieldError struct {
	// Namespace is the full field path using configured external field names,
	// such as Request.items[0].sku.
	Namespace string
	// StructNamespace is the full field path using Go struct field names.
	StructNamespace string
	// Field is the leaf field name using the configured external field name.
	Field string
	// StructField is the Go struct field name for the leaf field.
	StructField string
	// Tag is the validation tag that failed, such as required or min.
	Tag string
	// ActualTag is the underlying validator tag that failed after aliases are
	// resolved.
	ActualTag string
	// Param is the parameter supplied to the failed validation tag.
	Param string
	// Kind is the reflection kind of the invalid field value.
	Kind string
	// Type is the reflection type of the invalid field value.
	Type string
	// Value is the invalid field value observed by the validator.
	Value any
	// Message is a human-readable validation failure summary.
	Message string
}
