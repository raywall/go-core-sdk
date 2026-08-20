// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// selector implements item sorting and amount selection.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package selector

import (
	"context"
	"errors"
	"log/slog"
	"sort"

	selectorinternal "github.com/raywall/go-core-sdk/services/selector/internal"
	"github.com/raywall/go-core-sdk/services/selector/types"
)

// Selector orders records and selects items using an available amount.
//
// Selector is safe for concurrent use. It does not mutate input slices; sorting
// returns a new slice with the original item values in ordered positions.
type Selector struct {
	logger *slog.Logger
}

// New constructs a Selector service.
//
// The default selector writes structured JSON logs to stdout. Options can
// replace the logger for tests or application observability pipelines.
func New(configurers ...Option) (*Selector, error) {
	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}
	return &Selector{logger: options.logger}, nil
}

// Sort orders typed items using the configured field path.
//
// Sort constructs a default Selector for the call. Use New and the Item methods
// when a long-lived configured service instance is preferred.
func Sort[T any](ctx context.Context, items []T, config types.SortConfig, options ...Option) ([]T, error) {
	selector, err := New(options...)
	if err != nil {
		return nil, err
	}
	return sortTyped(ctx, selector, items, config)
}

// Select applies an available amount over typed ordered items.
//
// Select walks items in order. Integral mode stops at the first item that cannot
// be fully covered; partial mode includes that item with the remaining amount
// and then stops.
func Select[T any](ctx context.Context, items []T, config types.SelectionConfig, options ...Option) (types.SelectionResult[T], error) {
	selector, err := New(options...)
	if err != nil {
		return types.SelectionResult[T]{}, err
	}
	return selectTyped(ctx, selector, items, config)
}

// SortAndSelect orders typed items and applies the selection configuration in a
// single call.
func SortAndSelect[T any](ctx context.Context, items []T, sortConfig types.SortConfig, selectionConfig types.SelectionConfig, options ...Option) ([]T, types.SelectionResult[T], error) {
	selector, err := New(options...)
	if err != nil {
		return nil, types.SelectionResult[T]{}, err
	}

	ordered, err := sortTyped(ctx, selector, items, sortConfig)
	if err != nil {
		return nil, types.SelectionResult[T]{}, err
	}
	result, err := selectTyped(ctx, selector, ordered, selectionConfig)
	if err != nil {
		return nil, types.SelectionResult[T]{}, err
	}
	return ordered, result, nil
}

// SortItems orders generic map items using the configured field path.
func (s *Selector) SortItems(ctx context.Context, items []types.Item, config types.SortConfig) ([]types.Item, error) {
	return sortTyped(ctx, s, items, config)
}

// SelectItems applies an available amount over generic map items.
func (s *Selector) SelectItems(ctx context.Context, items []types.Item, config types.SelectionConfig) (types.SelectionResult[types.Item], error) {
	result, err := selectTyped(ctx, s, items, config)
	if err != nil {
		return types.SelectionResult[types.Item]{}, err
	}
	for index := range result.Payments {
		result.Payments[index].Item = CloneItem(result.Payments[index].Item)
	}
	return result, nil
}

// SortAndSelectItems orders generic map items and applies the selection
// configuration in one call.
func (s *Selector) SortAndSelectItems(ctx context.Context, items []types.Item, sortConfig types.SortConfig, selectionConfig types.SelectionConfig) ([]types.Item, types.SelectionResult[types.Item], error) {
	ordered, err := s.SortItems(ctx, items, sortConfig)
	if err != nil {
		return nil, types.SelectionResult[types.Item]{}, err
	}
	result, err := s.SelectItems(ctx, ordered, selectionConfig)
	if err != nil {
		return nil, types.SelectionResult[types.Item]{}, err
	}
	return ordered, result, nil
}

// ValueAt reads a configured field path from an item.
func ValueAt(item any, path string) (any, error) {
	return selectorinternal.ValueAt(item, path)
}

// CloneItem returns a shallow copy of a generic selector item.
func CloneItem(item types.Item) types.Item {
	copied := make(types.Item, len(item))
	for key, value := range item {
		copied[key] = value
	}
	return copied
}

func sortTyped[T any](ctx context.Context, selector *Selector, items []T, config types.SortConfig) ([]T, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config = normalizeSortConfig(config)
	if err := validateSortConfig(config); err != nil {
		return nil, err
	}

	entries := make([]sortEntry[T], len(items))
	for index, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, err := selectorinternal.ValueAt(item, config.Path)
		if err != nil {
			return nil, err
		}
		if _, err := selectorinternal.CompareValues(value, value, config.Kind, config.TimeLayout); err != nil {
			var incompatibleType types.IncompatibleTypeError
			if errors.As(err, &incompatibleType) {
				return nil, incompatibleType.WithPath(config.Path)
			}
			return nil, err
		}
		entries[index] = sortEntry[T]{item: item, value: value}
	}

	var compareErr error
	sort.SliceStable(entries, func(left int, right int) bool {
		if compareErr != nil {
			return false
		}
		comparison, err := selectorinternal.CompareValues(entries[left].value, entries[right].value, config.Kind, config.TimeLayout)
		if err != nil {
			compareErr = err
			return false
		}
		if config.Direction == types.Descending {
			return comparison > 0
		}
		return comparison < 0
	})
	if compareErr != nil {
		var incompatibleType types.IncompatibleTypeError
		if errors.As(compareErr, &incompatibleType) {
			return nil, incompatibleType.WithPath(config.Path)
		}
		return nil, compareErr
	}

	ordered := make([]T, len(entries))
	for index, entry := range entries {
		ordered[index] = entry.item
	}

	selector.logger.InfoContext(ctx, "selector_sort_completed", "items", len(items), "path", config.Path, "kind", config.Kind, "direction", config.Direction)
	return ordered, nil
}

func selectTyped[T any](ctx context.Context, selector *Selector, items []T, config types.SelectionConfig) (types.SelectionResult[T], error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config = normalizeSelectionConfig(config)
	if err := validateSelectionConfig(config); err != nil {
		return types.SelectionResult[T]{}, err
	}

	result := types.SelectionResult[T]{RemainingAmount: config.AvailableAmount}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return types.SelectionResult[T]{}, err
		}
		if result.RemainingAmount <= 0 {
			break
		}

		value, err := selectorinternal.ValueAt(item, config.AmountPath)
		if err != nil {
			return types.SelectionResult[T]{}, err
		}
		required, err := selectorinternal.AmountValue(value, config.DecimalScale, config.AmountPath)
		if err != nil {
			return types.SelectionResult[T]{}, err
		}

		if required <= result.RemainingAmount {
			result.Payments = append(result.Payments, types.Payment[T]{
				Item:           item,
				RequiredAmount: required,
				AppliedAmount:  required,
			})
			result.RemainingAmount -= required
			result.TotalAppliedAmount += required
			continue
		}

		if config.Mode == types.ModePartial {
			applied := result.RemainingAmount
			result.Payments = append(result.Payments, types.Payment[T]{
				Item:           item,
				RequiredAmount: required,
				AppliedAmount:  applied,
				Partial:        true,
			})
			result.TotalAppliedAmount += applied
			result.RemainingAmount = 0
		}
		break
	}

	selector.logger.InfoContext(ctx, "selector_selection_completed", "items", len(items), "selected", len(result.Payments), "amount_path", config.AmountPath, "mode", config.Mode, "total_applied", result.TotalAppliedAmount, "remaining", result.RemainingAmount)
	return result, nil
}

type sortEntry[T any] struct {
	item  T
	value any
}

func normalizeSortConfig(config types.SortConfig) types.SortConfig {
	if config.Kind == "" {
		config.Kind = types.KindString
	}
	if config.Direction == "" {
		config.Direction = types.Ascending
	}
	return config
}

func validateSortConfig(config types.SortConfig) error {
	if config.Path == "" {
		return types.InvalidConfigError{Field: "Path", Reason: "is required"}
	}
	switch config.Kind {
	case types.KindString, types.KindNumber, types.KindTime:
	default:
		return types.InvalidConfigError{Field: "Kind", Reason: "unsupported value kind"}
	}
	switch config.Direction {
	case types.Ascending, types.Descending:
	default:
		return types.InvalidConfigError{Field: "Direction", Reason: "unsupported direction"}
	}
	return nil
}

func normalizeSelectionConfig(config types.SelectionConfig) types.SelectionConfig {
	if config.Mode == "" {
		config.Mode = types.ModeIntegral
	}
	if config.DecimalScale <= 0 {
		config.DecimalScale = 1
	}
	return config
}

func validateSelectionConfig(config types.SelectionConfig) error {
	if config.AmountPath == "" {
		return types.InvalidConfigError{Field: "AmountPath", Reason: "is required"}
	}
	if config.AvailableAmount < 0 {
		return types.InvalidAmountError{Value: config.AvailableAmount, Reason: "available amount must not be negative"}
	}
	if config.DecimalScale <= 0 {
		return types.InvalidConfigError{Field: "DecimalScale", Reason: "must be greater than zero"}
	}
	switch config.Mode {
	case types.ModeIntegral, types.ModePartial:
	default:
		return types.InvalidConfigError{Field: "Mode", Reason: "unsupported selection mode"}
	}
	return nil
}
