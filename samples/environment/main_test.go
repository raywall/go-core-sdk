package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunLoadsRequiredAndDefaultEnvironmentValues(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	lookup := func(name string) (string, bool) {
		values := map[string]string{"APP_SERVICE_NAME": "orders-worker"}
		value, ok := values[name]
		return value, ok
	}

	if err := run(context.Background(), &out, lookup); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got, want := out.String(), "service=orders-worker environment=local\n"; got != want {
		t.Fatalf("run() output = %q, want %q", got, want)
	}
}
