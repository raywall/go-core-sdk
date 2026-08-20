package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunEmitsLogsAndCustomMetrics(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := run(context.Background(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"order_processing_started",
		"metric=increment name=orders.events.received",
		"metric=gauge name=orders.queue.depth",
		"order_processing_finished",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
