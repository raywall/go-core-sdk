// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types tests decision error wrapping behavior.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types_test

import (
	"errors"
	"testing"

	"github.com/raywall/go-core-sdk/services/decision/types"
)

// TestEvaluationError_Unwrap verifies wrapped CEL evaluation errors can be
// inspected with errors.Is.
func TestEvaluationError_Unwrap(t *testing.T) {
	t.Parallel()

	base := errors.New("runtime")
	err := types.EvaluationError{RuleName: "rule", Err: base}

	if !errors.Is(err, base) {
		t.Fatal("errors.Is(EvaluationError, base) = false, want true")
	}
}

// TestCompilationError_Unwrap verifies wrapped CEL compilation errors can be
// inspected with errors.Is.
func TestCompilationError_Unwrap(t *testing.T) {
	t.Parallel()

	base := errors.New("compile")
	err := types.CompilationError{Err: base}

	if !errors.Is(err, base) {
		t.Fatal("errors.Is(CompilationError, base) = false, want true")
	}
}
