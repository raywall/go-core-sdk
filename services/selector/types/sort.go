// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements selector sort configuration.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

// Item is a generic selector record.
type Item map[string]any

// Direction defines the order used to sort item values.
type Direction string

const (
	// Ascending sorts values from the lowest value to the highest value.
	Ascending Direction = "asc"

	// Descending sorts values from the highest value to the lowest value.
	Descending Direction = "desc"
)

// ValueKind defines how selector values are normalized before comparison.
type ValueKind string

const (
	// KindString compares values as strings.
	KindString ValueKind = "string"

	// KindNumber compares values as numbers.
	KindNumber ValueKind = "number"

	// KindTime compares values as time instants.
	KindTime ValueKind = "time"
)

// SortConfig configures item ordering by a field path.
type SortConfig struct {
	// Path is the field path used for ordering, such as dueDate,
	// customer.name or installments[0].amount.
	Path string
	// Kind controls how field values are compared. A zero value uses string
	// comparison.
	Kind ValueKind
	// Direction controls whether values are sorted ascending or descending. A
	// zero value uses ascending order.
	Direction Direction
	// TimeLayout is used to parse string values when Kind is KindTime. A zero
	// value uses time.RFC3339.
	TimeLayout string
}
