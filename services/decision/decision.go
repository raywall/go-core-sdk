// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// decision implements CEL expression compilation and evaluation.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package decision

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"cel.dev/cel-go/cel"

	decisioninternal "github.com/raywall/go-core-sdk/services/decision/internal"
	"github.com/raywall/go-core-sdk/services/decision/types"
)

// Decision evaluates CEL rules against named entities.
//
// Decision is safe for concurrent use. It caches compiled programs by
// expression and entity names to avoid recompiling hot decision rules.
type Decision struct {
	logger *slog.Logger
	mu     sync.RWMutex
	cache  map[string]compiledRule
}

type compiledRule struct {
	program cel.Program
}

// New constructs a Decision service.
//
// The default service writes structured JSON logs to stdout and starts with an
// empty compiled-program cache.
func New(configurers ...Option) (*Decision, error) {
	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}
	return &Decision{
		logger: options.logger,
		cache:  make(map[string]compiledRule),
	}, nil
}

// Evaluate compiles or reuses the CEL rule and evaluates it against entities.
//
// Each key in EvaluationInput.Entities becomes a CEL variable. Entity values may
// be maps, structs, slices or scalar values. Struct field names use json tags
// when present.
func (d *Decision) Evaluate(ctx context.Context, input types.EvaluationInput) (types.EvaluationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	startedAt := time.Now()

	if err := validateInput(input); err != nil {
		return types.EvaluationResult{}, err
	}

	activation, err := decisioninternal.Activation(input.Entities)
	if err != nil {
		return types.EvaluationResult{}, types.EntityConversionError{Err: err}
	}

	entityNames := sortedEntityNames(activation)
	compiled, cacheHit, err := d.compiled(input.Rule.Expression, entityNames)
	if err != nil {
		return types.EvaluationResult{}, err
	}

	output, _, err := compiled.program.Eval(activation)
	if err != nil {
		return types.EvaluationResult{}, types.EvaluationError{
			RuleName:   input.Rule.Name,
			Expression: input.Rule.Expression,
			Err:        err,
		}
	}

	value := output.Value()
	allowed, ok := value.(bool)
	if !ok {
		return types.EvaluationResult{}, types.NonBooleanResultError{
			RuleName:   input.Rule.Name,
			Expression: input.Rule.Expression,
			Value:      value,
		}
	}

	result := types.EvaluationResult{
		RuleName:    input.Rule.Name,
		Expression:  input.Rule.Expression,
		Allowed:     allowed,
		Duration:    time.Since(startedAt),
		EntityCount: len(input.Entities),
		CacheHit:    cacheHit,
	}

	d.logger.InfoContext(ctx, "decision_evaluation_completed", "rule", input.Rule.Name, "allowed", result.Allowed, "duration", result.Duration.String(), "entities", result.EntityCount, "cache_hit", result.CacheHit)
	return result, nil
}

func (d *Decision) compiled(expression string, entityNames []string) (compiledRule, bool, error) {
	key := cacheKey(expression, entityNames)

	d.mu.RLock()
	cached, ok := d.cache[key]
	d.mu.RUnlock()
	if ok {
		return cached, true, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if cached, ok := d.cache[key]; ok {
		return cached, true, nil
	}

	envOptions := make([]cel.EnvOption, 0, len(entityNames))
	for _, name := range entityNames {
		envOptions = append(envOptions, cel.Variable(name, cel.DynType))
	}
	env, err := cel.NewEnv(envOptions...)
	if err != nil {
		return compiledRule{}, false, fmt.Errorf("new CEL environment: %w", err)
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return compiledRule{}, false, types.CompilationError{Expression: expression, Err: issues.Err()}
	}

	program, err := env.Program(ast)
	if err != nil {
		return compiledRule{}, false, types.CompilationError{Expression: expression, Err: err}
	}

	compiled := compiledRule{program: program}
	d.cache[key] = compiled
	return compiled, false, nil
}

func validateInput(input types.EvaluationInput) error {
	if strings.TrimSpace(input.Rule.Expression) == "" {
		return types.InvalidConfigError{Field: "Rule.Expression", Reason: "is required"}
	}
	if input.Entities == nil {
		return types.InvalidConfigError{Field: "Entities", Reason: "is required"}
	}
	for name := range input.Entities {
		if strings.TrimSpace(name) == "" {
			return types.InvalidConfigError{Field: "Entities", Reason: "contains an empty entity name"}
		}
	}
	return nil
}

func sortedEntityNames(entities map[string]any) []string {
	names := make([]string, 0, len(entities))
	for name := range entities {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func cacheKey(expression string, entityNames []string) string {
	return expression + "\x00" + strings.Join(entityNames, "\x00")
}
