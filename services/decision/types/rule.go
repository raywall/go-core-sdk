// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements decision rule inputs.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

// Rule describes one CEL decision rule.
type Rule struct {
	// Name identifies the rule in logs and results.
	Name string
	// Expression is the CEL expression evaluated by the decision service.
	Expression string
	// Description documents the business purpose of the rule.
	Description string
}

// EvaluationInput contains the rule and entities used for one decision.
type EvaluationInput struct {
	// Rule is the CEL rule to compile and evaluate.
	Rule Rule
	// Entities maps variable names to entity values. Each key becomes directly
	// available inside the CEL expression.
	Entities map[string]any
}
