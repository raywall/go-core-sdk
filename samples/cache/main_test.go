package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/raywall/go-core-sdk/services/cache"
)

func TestRunStoresAndReadsCustomer(t *testing.T) {
	t.Parallel()

	store, err := cache.New[Customer](cache.Config{})
	if err != nil {
		t.Fatalf("cache.New() error = %v", err)
	}

	var out bytes.Buffer
	if err := run(context.Background(), CustomerUseCase{Store: store, Output: &out}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got := out.String(); !strings.Contains(got, "found=true") || !strings.Contains(got, "Ana") {
		t.Fatalf("run() output = %q", got)
	}
}
