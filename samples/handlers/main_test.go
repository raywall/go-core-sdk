package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunDemonstratesAllHandlerAdapters(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := run(context.Background(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"lambdaS3Processed=1", "httpProcessed=1", "sqsWorkerDeleted=1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
