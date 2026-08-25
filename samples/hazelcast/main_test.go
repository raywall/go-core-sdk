// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/hazelcast tests the distributed configuration consumer sample.
//
// This file is part of the Hazelcast sample bounded context within the
// Samples service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunReadsRuntimeParameters(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if err := run(context.Background(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"enabled=true", "maxInstallments=5", "strategy=oldest-first", "allowPartial=true", "minimumAmount=1000"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
