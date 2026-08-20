// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements decision evaluation results.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "time"

// EvaluationResult describes the outcome of one decision rule evaluation.
type EvaluationResult struct {
	// RuleName identifies the evaluated rule.
	RuleName string
	// Expression is the CEL expression that was evaluated.
	Expression string
	// Allowed is the boolean result returned by the CEL expression.
	Allowed bool
	// Duration is the total evaluation duration, including conversion, compile
	// cache lookup and CEL program execution.
	Duration time.Duration
	// EntityCount is the number of entities supplied to the expression.
	EntityCount int
	// CacheHit indicates whether the compiled CEL program came from cache.
	CacheHit bool
}
