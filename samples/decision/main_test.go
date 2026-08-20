package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunEvaluatesMarginDecision(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := run(context.Background(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "rule=margin-approved") || !strings.Contains(got, "allowed=true") {
		t.Fatalf("run() output = %q", got)
	}
}
