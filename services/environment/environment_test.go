// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package environment_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/raywall/go-core-sdk/services/environment"
	"github.com/raywall/go-core-sdk/services/environment/types"
)

func TestEnvironment_GetReturnsExistingValue(t *testing.T) {
	t.Parallel()

	service := newEnvironment(t, map[string]string{"APP_NAME": "orders"})

	value, err := service.Get(context.Background(), "APP_NAME")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value != "orders" {
		t.Fatalf("Get() = %q, want orders", value)
	}
}

func TestEnvironment_GetPreservesExistingEmptyValue(t *testing.T) {
	t.Parallel()

	service := newEnvironment(t, map[string]string{"APP_NAME": ""})

	value, err := service.GetDefault(context.Background(), "APP_NAME", "fallback")
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if value != "" {
		t.Fatalf("GetDefault() = %q, want empty string", value)
	}
}

func TestEnvironment_GetDefaultReturnsFallbackWhenAbsent(t *testing.T) {
	t.Parallel()

	service := newEnvironment(t, map[string]string{})

	value, err := service.GetDefault(context.Background(), "APP_NAME", "fallback")
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if value != "fallback" {
		t.Fatalf("GetDefault() = %q, want fallback", value)
	}
}

func TestEnvironment_GetReturnsNotFoundErrorWhenAbsent(t *testing.T) {
	t.Parallel()

	service := newEnvironment(t, map[string]string{})

	_, err := service.Get(context.Background(), "APP_NAME")
	if err == nil {
		t.Fatal("Get() error = nil, want error")
	}
	var notFound types.VariableNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Get() error type = %T, want VariableNotFoundError", err)
	}
	if notFound.Name != "APP_NAME" {
		t.Fatalf("VariableNotFoundError.Name = %q, want APP_NAME", notFound.Name)
	}
}

func TestEnvironment_RejectsEmptyName(t *testing.T) {
	t.Parallel()

	service := newEnvironment(t, map[string]string{})

	_, err := service.Get(context.Background(), " ")
	if err == nil {
		t.Fatal("Get() error = nil, want error")
	}
	var invalid types.InvalidInputError
	if !errors.As(err, &invalid) {
		t.Fatalf("Get() error type = %T, want InvalidInputError", err)
	}
}

func TestEnvironment_RespectsContextCancellation(t *testing.T) {
	t.Parallel()

	service := newEnvironment(t, map[string]string{"APP_NAME": "orders"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.Get(ctx, "APP_NAME")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() error = %v, want context.Canceled", err)
	}
}

func TestGetDefaultHelper(t *testing.T) {
	t.Parallel()

	value, err := environment.GetDefault(context.Background(), "APP_NAME", "orders", environment.WithLogger(discardLogger()), environment.WithLookupFunc(fakeLookup(map[string]string{})))
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if value != "orders" {
		t.Fatalf("GetDefault() = %q, want orders", value)
	}
}

func newEnvironment(t *testing.T, values map[string]string) *environment.Environment {
	t.Helper()
	service, err := environment.New(environment.WithLogger(discardLogger()), environment.WithLookupFunc(fakeLookup(values)))
	if err != nil {
		t.Fatalf("environment.New() error = %v", err)
	}
	return service
}

func fakeLookup(values map[string]string) environment.LookupFunc {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
