// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// selector tests financial item ordering and selection behavior.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package selector_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/services/selector"
	"github.com/raywall/go-core-sdk/services/selector/types"
)

type installment struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	DueDate   time.Time `json:"dueDate"`
	Amount    int64     `json:"amount"`
	Borrower  borrower  `json:"borrower"`
	Reference string    `json:"reference"`
}

type borrower struct {
	Name string `json:"name"`
}

// TestSortOrdersTypedItemsByNumberStringAndTime verifies typed structs can be
// sorted by several value kinds and directions.
func TestSortOrdersTypedItemsByNumberStringAndTime(t *testing.T) {
	t.Parallel()

	items := []installment{
		{ID: "b", DueDate: mustDate(t, "2026-02-01"), Amount: 200, Borrower: borrower{Name: "Maria"}},
		{ID: "a", DueDate: mustDate(t, "2026-01-01"), Amount: 100, Borrower: borrower{Name: "Ana"}},
		{ID: "c", DueDate: mustDate(t, "2026-03-01"), Amount: 100, Borrower: borrower{Name: "Zoe"}},
	}

	byAmount, err := selector.Sort(context.Background(), items, types.SortConfig{
		Path: "amount",
		Kind: types.KindNumber,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Sort by amount error = %v", err)
	}
	assertIDs(t, byAmount, []string{"a", "c", "b"})

	byName, err := selector.Sort(context.Background(), items, types.SortConfig{
		Path:      "borrower.name",
		Kind:      types.KindString,
		Direction: types.Descending,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Sort by name error = %v", err)
	}
	assertIDs(t, byName, []string{"c", "b", "a"})

	byDate, err := selector.Sort(context.Background(), items, types.SortConfig{
		Path:      "dueDate",
		Kind:      types.KindTime,
		Direction: types.Descending,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Sort by dueDate error = %v", err)
	}
	assertIDs(t, byDate, []string{"c", "b", "a"})
}

// TestSortPreservesStableOrder verifies equal sort keys retain their original
// relative order.
func TestSortPreservesStableOrder(t *testing.T) {
	t.Parallel()

	items := []installment{
		{ID: "first", Amount: 100},
		{ID: "second", Amount: 100},
		{ID: "third", Amount: 50},
	}

	ordered, err := selector.Sort(context.Background(), items, types.SortConfig{
		Path: "amount",
		Kind: types.KindNumber,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Sort error = %v", err)
	}

	assertIDs(t, ordered, []string{"third", "first", "second"})
}

// TestSelectIntegralStopsAtFirstUncoveredItem verifies integral mode only
// returns fully covered items and keeps the remaining balance.
func TestSelectIntegralStopsAtFirstUncoveredItem(t *testing.T) {
	t.Parallel()

	result, err := selector.Select(context.Background(), []installment{
		{ID: "1", Amount: 100},
		{ID: "2", Amount: 100},
	}, types.SelectionConfig{
		AmountPath:      "amount",
		AvailableAmount: 150,
		Mode:            types.ModeIntegral,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Select error = %v", err)
	}

	if got, want := len(result.Payments), 1; got != want {
		t.Fatalf("payments length = %d, want %d", got, want)
	}
	if got, want := result.RemainingAmount, int64(50); got != want {
		t.Fatalf("RemainingAmount = %d, want %d", got, want)
	}
	if result.Payments[0].Partial {
		t.Fatal("first payment Partial = true, want false")
	}
}

// TestSelectPartialIncludesNextItem verifies partial mode applies the remaining
// balance to the next item and then stops.
func TestSelectPartialIncludesNextItem(t *testing.T) {
	t.Parallel()

	result, err := selector.Select(context.Background(), []installment{
		{ID: "1", Amount: 100},
		{ID: "2", Amount: 100},
		{ID: "3", Amount: 100},
	}, types.SelectionConfig{
		AmountPath:      "amount",
		AvailableAmount: 250,
		Mode:            types.ModePartial,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Select error = %v", err)
	}

	if got, want := len(result.Payments), 3; got != want {
		t.Fatalf("payments length = %d, want %d", got, want)
	}
	if got, want := result.Payments[2].AppliedAmount, int64(50); got != want {
		t.Fatalf("partial AppliedAmount = %d, want %d", got, want)
	}
	if !result.Payments[2].Partial {
		t.Fatal("partial payment Partial = false, want true")
	}
	if got, want := result.TotalAppliedAmount, int64(250); got != want {
		t.Fatalf("TotalAppliedAmount = %d, want %d", got, want)
	}
}

// TestSelectConvertsDecimalValuesWithScale verifies decimal string values can
// be converted into the configured minimum monetary unit.
func TestSelectConvertsDecimalValuesWithScale(t *testing.T) {
	t.Parallel()

	result, err := selector.Select(context.Background(), []types.Item{
		{"id": "1", "amount": "10.50"},
		{"id": "2", "amount": 2.25},
	}, types.SelectionConfig{
		AmountPath:      "amount",
		AvailableAmount: 1275,
		Mode:            types.ModeIntegral,
		DecimalScale:    100,
	}, discardLogs())
	if err != nil {
		t.Fatalf("selector.Select error = %v", err)
	}

	if got, want := result.TotalAppliedAmount, int64(1275); got != want {
		t.Fatalf("TotalAppliedAmount = %d, want %d", got, want)
	}
}

// TestSortAndSelectItemsClonesSelectedMaps verifies generic map results do not
// share selected item maps with the caller.
func TestSortAndSelectItemsClonesSelectedMaps(t *testing.T) {
	t.Parallel()

	service := newTestSelector(t)
	items := []types.Item{
		{"id": "b", "due": "2026-02-01", "amount": 100},
		{"id": "a", "due": "2026-01-01", "amount": 100},
	}
	_, result, err := service.SortAndSelectItems(context.Background(), items, types.SortConfig{
		Path:       "due",
		Kind:       types.KindTime,
		TimeLayout: time.DateOnly,
	}, types.SelectionConfig{
		AmountPath:      "amount",
		AvailableAmount: 100,
	})
	if err != nil {
		t.Fatalf("SortAndSelectItems error = %v", err)
	}

	result.Payments[0].Item["id"] = "changed"
	if got := result.Payments[0].Item["id"]; got != "changed" {
		t.Fatalf("mutated clone id = %v, want changed", got)
	}
	if got := items[1]["id"]; got != "a" {
		t.Fatalf("caller item id = %v, want a", got)
	}
}

// TestValueAtReadsNestedPaths verifies field paths can traverse maps, structs
// and slice indexes.
func TestValueAtReadsNestedPaths(t *testing.T) {
	t.Parallel()

	value, err := selector.ValueAt(map[string]any{
		"customer": map[string]any{
			"contracts": []any{
				map[string]any{"amount": 100},
			},
		},
	}, "customer.contracts[0].amount")
	if err != nil {
		t.Fatalf("selector.ValueAt error = %v", err)
	}
	if got, want := value, 100; got != want {
		t.Fatalf("ValueAt = %v, want %v", got, want)
	}
}

// TestSelectorReturnsHelpfulErrors verifies invalid configuration, missing
// fields and incompatible values return typed errors.
func TestSelectorReturnsHelpfulErrors(t *testing.T) {
	t.Parallel()

	if _, err := selector.Sort(context.Background(), []installment{{ID: "1"}}, types.SortConfig{}, discardLogs()); !isInvalidConfig(err) {
		t.Fatalf("Sort invalid config error = %T %[1]v, want InvalidConfigError", err)
	}
	if _, err := selector.Sort(context.Background(), []installment{{ID: "1"}}, types.SortConfig{Path: "missing"}, discardLogs()); !isFieldNotFound(err) {
		t.Fatalf("Sort missing field error = %T %[1]v, want FieldNotFoundError", err)
	}
	if _, err := selector.Sort(context.Background(), []installment{{ID: "abc"}}, types.SortConfig{Path: "id", Kind: types.KindNumber}, discardLogs()); !isIncompatibleType(err) {
		t.Fatalf("Sort incompatible type error = %T %[1]v, want IncompatibleTypeError", err)
	}
	if _, err := selector.Select(context.Background(), []installment{{Amount: -1}}, types.SelectionConfig{AmountPath: "amount", AvailableAmount: 1}, discardLogs()); !isInvalidAmount(err) {
		t.Fatalf("Select invalid amount error = %T %[1]v, want InvalidAmountError", err)
	}
}

func newTestSelector(t *testing.T) *selector.Selector {
	t.Helper()

	service, err := selector.New(discardLogs())
	if err != nil {
		t.Fatalf("selector.New error = %v", err)
	}
	return service
}

func discardLogs() selector.Option {
	return selector.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func assertIDs(t *testing.T, items []installment, want []string) {
	t.Helper()

	if len(items) != len(want) {
		t.Fatalf("items length = %d, want %d", len(items), len(want))
	}
	for index := range items {
		if items[index].ID != want[index] {
			t.Fatalf("items[%d].ID = %q, want %q; items=%+v", index, items[index].ID, want[index], items)
		}
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		t.Fatalf("time.Parse error = %v", err)
	}
	return parsed
}

func isInvalidConfig(err error) bool {
	var target types.InvalidConfigError
	return errors.As(err, &target)
}

func isFieldNotFound(err error) bool {
	var target types.FieldNotFoundError
	return errors.As(err, &target)
}

func isIncompatibleType(err error) bool {
	var target types.IncompatibleTypeError
	return errors.As(err, &target)
}

func isInvalidAmount(err error) bool {
	var target types.InvalidAmountError
	return errors.As(err, &target)
}
