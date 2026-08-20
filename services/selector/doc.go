// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// selector implements financial item ordering and value-based selection.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package selector orders financial records and selects payable items using an
// available amount.
//
// The package is useful for flows such as ordering overdue installments from
// oldest to newest and then selecting which installments can be fully or
// partially paid. Financial amounts are represented as int64 in the caller's
// minimum monetary unit, such as cents, to avoid floating-point drift.
//
// Usage:
//
//	ordered, result, err := selector.SortAndSelect(ctx, installments,
//		types.SortConfig{
//			Path:      "dueDate",
//			Kind:      types.KindTime,
//			Direction: types.Ascending,
//		},
//		types.SelectionConfig{
//			AmountPath:      "amountCents",
//			AvailableAmount: 25000,
//			Mode:            types.ModePartial,
//		},
//	)
//
// Thread safety: Selector is safe for concurrent use after construction.
package selector
