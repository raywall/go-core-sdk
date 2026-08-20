// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// decision implements CEL-based decision rule evaluation.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package decision evaluates CEL expressions against multiple named entities.
//
// The package is designed for microservices that need fast, simple and
// observable business-rule decisions. Callers provide a CEL expression and a
// map of entities; each entity name becomes a variable in the expression.
//
// Usage:
//
//	engine, err := decision.New()
//	if err != nil {
//		return err
//	}
//
//	result, err := engine.Evaluate(ctx, types.EvaluationInput{
//		Rule: types.Rule{
//			Name:       "margin-approved",
//			Expression: "worker.active && proposal.amount <= worker.availableMargin",
//		},
//		Entities: map[string]any{
//			"worker":   worker,
//			"proposal": proposal,
//		},
//	})
//
// Thread safety: Decision is safe for concurrent use. Compiled CEL programs are
// cached by expression and entity names behind an internal lock.
package decision
