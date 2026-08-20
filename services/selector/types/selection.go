// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements selector payment selection configuration and results.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

// SelectionMode defines whether selector may include a partially covered item.
type SelectionMode string

const (
	// ModeIntegral selects only items that can be fully covered by the
	// available amount.
	ModeIntegral SelectionMode = "integral"

	// ModePartial selects fully covered items first and then includes the next
	// item with a partial applied amount when any balance remains.
	ModePartial SelectionMode = "partial"
)

// SelectionConfig configures how an available amount is applied to ordered
// items.
type SelectionConfig struct {
	// AmountPath is the item field path that contains the required amount.
	AmountPath string
	// AvailableAmount is the amount available to apply, represented in the
	// caller's minimum monetary unit, such as cents.
	AvailableAmount int64
	// Mode controls whether partially covered items may be returned. A zero
	// value uses ModeIntegral.
	Mode SelectionMode
	// DecimalScale converts decimal string or float values into the minimum
	// monetary unit. Use 100 to convert 10.50 into 1050 cents. A zero value uses
	// 1, meaning incoming values are already in the minimum unit.
	DecimalScale int64
}

// Payment represents the amount applied to one selected item.
type Payment[T any] struct {
	// Item is the selected item.
	Item T
	// RequiredAmount is the full amount required by the item.
	RequiredAmount int64
	// AppliedAmount is the amount actually applied to the item.
	AppliedAmount int64
	// Partial indicates whether AppliedAmount is lower than RequiredAmount.
	Partial bool
}

// SelectionResult contains selected payments and balance totals.
type SelectionResult[T any] struct {
	// Payments contains the selected item payments in input order.
	Payments []Payment[T]
	// RemainingAmount is the amount left after applying payments.
	RemainingAmount int64
	// TotalAppliedAmount is the sum of all applied payment amounts.
	TotalAppliedAmount int64
}
