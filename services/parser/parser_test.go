// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package parser_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/raywall/go-core-sdk/services/parser"
	"github.com/raywall/go-core-sdk/services/parser/types"
)

func TestParser_ParseDTOToEntity(t *testing.T) {
	t.Parallel()

	service := newParser(t)
	source := orderDTO{
		ID:       "order-123",
		Customer: customerDTO{Document: "12345678901"},
		Items: []itemDTO{
			{SKU: "sku-1", Quantity: 2},
			{SKU: "sku-2", Quantity: 1},
		},
		Extra: "ignored",
	}
	var target orderEntity

	if err := service.Parse(context.Background(), source, &target); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := orderEntity{
		ID:       "order-123",
		Customer: customerEntity{Document: "12345678901"},
		Items: []itemEntity{
			{SKU: "sku-1", Quantity: 2},
			{SKU: "sku-2", Quantity: 1},
		},
	}
	if !reflect.DeepEqual(target, want) {
		t.Fatalf("target = %+v, want %+v", target, want)
	}
}

func TestParser_ParseMapToStruct(t *testing.T) {
	t.Parallel()

	service := newParser(t)
	source := map[string]any{
		"id": "customer-123",
		"profile": map[string]any{
			"name": "Ana",
		},
	}
	var target customerProfile

	if err := service.Parse(context.Background(), source, &target); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if target.ID != "customer-123" || target.Profile.Name != "Ana" {
		t.Fatalf("target = %+v", target)
	}
}

func TestParseAsConvertsSlice(t *testing.T) {
	t.Parallel()

	source := []itemDTO{
		{SKU: "sku-1", Quantity: 2},
		{SKU: "sku-2", Quantity: 1},
	}

	target, err := parser.ParseAs[[]itemEntity](context.Background(), source, parser.WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("ParseAs() error = %v", err)
	}

	want := []itemEntity{
		{SKU: "sku-1", Quantity: 2},
		{SKU: "sku-2", Quantity: 1},
	}
	if !reflect.DeepEqual(target, want) {
		t.Fatalf("target = %+v, want %+v", target, want)
	}
}

func TestParser_ParseRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	service := newParser(t)
	var target orderEntity
	tests := []struct {
		name   string
		source any
		target any
		field  string
	}{
		{name: "nil source", source: nil, target: &target, field: "Source"},
		{name: "nil target", source: orderDTO{}, target: nil, field: "Target"},
		{name: "non pointer target", source: orderDTO{}, target: target, field: "Target"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := service.Parse(context.Background(), test.source, test.target)
			if err == nil {
				t.Fatal("Parse() error = nil, want error")
			}
			var invalid types.InvalidInputError
			if !errors.As(err, &invalid) {
				t.Fatalf("Parse() error type = %T, want InvalidInputError", err)
			}
			if invalid.Field != test.field {
				t.Fatalf("InvalidInputError.Field = %q, want %q", invalid.Field, test.field)
			}
		})
	}
}

func TestParser_ParseWrapsEncodeError(t *testing.T) {
	t.Parallel()

	service := newParser(t)
	var target map[string]any

	err := service.Parse(context.Background(), map[string]any{"fn": func() {}}, &target)
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	var encodeErr types.EncodeError
	if !errors.As(err, &encodeErr) {
		t.Fatalf("Parse() error type = %T, want EncodeError", err)
	}
}

func TestParser_ParseWrapsDecodeError(t *testing.T) {
	t.Parallel()

	service := newParser(t)
	source := map[string]any{"quantity": "not-a-number"}
	var target itemEntity

	err := service.Parse(context.Background(), source, &target)
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
	var decodeErr types.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("Parse() error type = %T, want DecodeError", err)
	}
}

func TestParser_ParseRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	service := newParser(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var target itemEntity

	err := service.Parse(ctx, itemDTO{SKU: "sku-1", Quantity: 1}, &target)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Parse() error = %v, want context.Canceled", err)
	}
}

func newParser(t *testing.T) *parser.Parser {
	t.Helper()
	service, err := parser.New(parser.WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("parser.New() error = %v", err)
	}
	return service
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type orderDTO struct {
	ID       string      `json:"id"`
	Customer customerDTO `json:"customer"`
	Items    []itemDTO   `json:"items"`
	Extra    string      `json:"extra"`
}

type customerDTO struct {
	Document string `json:"document"`
}

type itemDTO struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type orderEntity struct {
	ID       string         `json:"id"`
	Customer customerEntity `json:"customer"`
	Items    []itemEntity   `json:"items"`
}

type customerEntity struct {
	Document string `json:"document"`
}

type itemEntity struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type customerProfile struct {
	ID      string `json:"id"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
}
