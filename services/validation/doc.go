// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// validation implements struct validation orchestration.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package validation simplifies validating structs and their nested structs
// using github.com/go-playground/validator/v10.
//
// The service returns a typed validation error that carries every invalid field
// found during validation. It also walks nested structs inside pointers, slices,
// arrays and maps so callers do not have to remember to add dive tags only to
// validate child struct fields.
//
// Usage:
//
//	validator, err := validation.New()
//	if err != nil {
//		return err
//	}
//
//	if err := validator.Validate(ctx, request); err != nil {
//		var validationErr *types.ValidationError
//		if errors.As(err, &validationErr) {
//			for _, field := range validationErr.Fields {
//				fmt.Println(field.Namespace, field.Tag)
//			}
//		}
//	}
//
// Thread safety: Validator is safe for concurrent use after construction. All
// custom validations must be registered through options passed to New before
// the validator is used.
package validation
