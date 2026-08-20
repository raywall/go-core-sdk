// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// decision tests CEL-based decision evaluation.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package decision_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/raywall/go-core-sdk/services/decision"
	"github.com/raywall/go-core-sdk/services/decision/types"
)

type worker struct {
	Active          bool  `json:"active"`
	Age             int64 `json:"age"`
	AvailableMargin int64 `json:"availableMargin"`
}

type proposal struct {
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

// TestDecision_EvaluateReturnsAllowed verifies a true CEL expression evaluates
// to an allowed decision.
func TestDecision_EvaluateReturnsAllowed(t *testing.T) {
	t.Parallel()

	engine := newTestDecision(t)
	result, err := engine.Evaluate(context.Background(), types.EvaluationInput{
		Rule: types.Rule{
			Name:       "approved",
			Expression: "worker.active && worker.age >= 18 && proposal.amount <= worker.availableMargin",
		},
		Entities: map[string]any{
			"worker":   worker{Active: true, Age: 35, AvailableMargin: 1000},
			"proposal": proposal{Amount: 700, Status: "OPEN"},
		},
	})
	if err != nil {
		t.Fatalf("Decision.Evaluate() error = %v", err)
	}

	if !result.Allowed {
		t.Fatal("EvaluationResult.Allowed = false, want true")
	}
	if result.CacheHit {
		t.Fatal("EvaluationResult.CacheHit = true on first evaluation, want false")
	}
}

// TestDecision_EvaluateReturnsDenied verifies a false CEL expression evaluates
// to a denied decision without returning an error.
func TestDecision_EvaluateReturnsDenied(t *testing.T) {
	t.Parallel()

	engine := newTestDecision(t)
	result, err := engine.Evaluate(context.Background(), types.EvaluationInput{
		Rule: types.Rule{
			Name:       "denied",
			Expression: "proposal.amount <= worker.availableMargin",
		},
		Entities: map[string]any{
			"worker":   worker{AvailableMargin: 100},
			"proposal": proposal{Amount: 700},
		},
	})
	if err != nil {
		t.Fatalf("Decision.Evaluate() error = %v", err)
	}

	if result.Allowed {
		t.Fatal("EvaluationResult.Allowed = true, want false")
	}
}

// TestDecision_EvaluateSupportsNestedMapsAndLists verifies expressions can read
// nested maps and list values.
func TestDecision_EvaluateSupportsNestedMapsAndLists(t *testing.T) {
	t.Parallel()

	engine := newTestDecision(t)
	result, err := engine.Evaluate(context.Background(), types.EvaluationInput{
		Rule: types.Rule{
			Name:       "nested",
			Expression: "contract.installments[0].status == 'OPEN' && contract.borrower.document == '123'",
		},
		Entities: map[string]any{
			"contract": map[string]any{
				"borrower": map[string]any{"document": "123"},
				"installments": []map[string]any{
					{"status": "OPEN"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Decision.Evaluate() error = %v", err)
	}

	if !result.Allowed {
		t.Fatal("EvaluationResult.Allowed = false, want true")
	}
}

// TestDecision_EvaluateUsesCompiledCache verifies repeated evaluations reuse a
// compiled CEL program.
func TestDecision_EvaluateUsesCompiledCache(t *testing.T) {
	t.Parallel()

	engine := newTestDecision(t)
	input := types.EvaluationInput{
		Rule: types.Rule{
			Name:       "cache",
			Expression: "worker.active",
		},
		Entities: map[string]any{"worker": worker{Active: true}},
	}

	if _, err := engine.Evaluate(context.Background(), input); err != nil {
		t.Fatalf("first Evaluate() error = %v", err)
	}
	result, err := engine.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("second Evaluate() error = %v", err)
	}

	if !result.CacheHit {
		t.Fatal("EvaluationResult.CacheHit = false on second evaluation, want true")
	}
}

// TestDecision_EvaluateIsRaceSafe verifies concurrent evaluations can share the
// same compiled-program cache.
func TestDecision_EvaluateIsRaceSafe(t *testing.T) {
	t.Parallel()

	engine := newTestDecision(t)
	input := types.EvaluationInput{
		Rule: types.Rule{
			Name:       "concurrent",
			Expression: "worker.active && proposal.amount <= worker.availableMargin",
		},
		Entities: map[string]any{
			"worker":   worker{Active: true, AvailableMargin: 1000},
			"proposal": proposal{Amount: 500},
		},
	}

	var wg sync.WaitGroup
	for range 25 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := engine.Evaluate(context.Background(), input)
			if err != nil {
				t.Errorf("Decision.Evaluate() error = %v", err)
				return
			}
			if !result.Allowed {
				t.Error("EvaluationResult.Allowed = false, want true")
			}
		}()
	}
	wg.Wait()
}

// TestDecision_EvaluateReturnsTypedErrors verifies invalid expressions, runtime
// errors, non-boolean results and invalid inputs are reported with typed errors.
func TestDecision_EvaluateReturnsTypedErrors(t *testing.T) {
	t.Parallel()

	engine := newTestDecision(t)

	tests := []struct {
		name  string
		input types.EvaluationInput
		check func(error) bool
	}{
		{
			name: "missing expression",
			input: types.EvaluationInput{
				Rule:     types.Rule{Name: "missing"},
				Entities: map[string]any{},
			},
			check: isInvalidConfig,
		},
		{
			name: "compilation error",
			input: types.EvaluationInput{
				Rule:     types.Rule{Name: "compile", Expression: "worker."},
				Entities: map[string]any{"worker": worker{}},
			},
			check: isCompilationError,
		},
		{
			name: "evaluation error",
			input: types.EvaluationInput{
				Rule:     types.Rule{Name: "eval", Expression: "missing.active"},
				Entities: map[string]any{"worker": worker{}},
			},
			check: isCompilationError,
		},
		{
			name: "non boolean",
			input: types.EvaluationInput{
				Rule:     types.Rule{Name: "non-bool", Expression: "worker.age"},
				Entities: map[string]any{"worker": worker{Age: 10}},
			},
			check: isNonBooleanError,
		},
		{
			name: "entity conversion",
			input: types.EvaluationInput{
				Rule:     types.Rule{Name: "entity", Expression: "worker.active"},
				Entities: map[string]any{"worker": map[int]string{1: "bad"}},
			},
			check: isEntityConversionError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := engine.Evaluate(context.Background(), test.input)
			if err == nil {
				t.Fatal("Decision.Evaluate() error = nil, want error")
			}
			if !test.check(err) {
				t.Fatalf("Decision.Evaluate() error = %T %[1]v, want expected typed error", err)
			}
		})
	}
}

func newTestDecision(t *testing.T) *decision.Decision {
	t.Helper()

	engine, err := decision.New(decision.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))))
	if err != nil {
		t.Fatalf("decision.New() error = %v", err)
	}
	return engine
}

func isInvalidConfig(err error) bool {
	var target types.InvalidConfigError
	return errors.As(err, &target)
}

func isCompilationError(err error) bool {
	var target types.CompilationError
	return errors.As(err, &target)
}

func isNonBooleanError(err error) bool {
	var target types.NonBooleanResultError
	return errors.As(err, &target)
}

func isEntityConversionError(err error) bool {
	var target types.EntityConversionError
	return errors.As(err, &target)
}
