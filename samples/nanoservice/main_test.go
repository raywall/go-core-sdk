// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice tests the executable nano-service sample.
//
// This file is part of the Nanoservice sample bounded context within the
// Samples package.
//
// Author:  Raywall
// Created: 2026-08-25
// Updated: 2026-08-25

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunProcessesSQSMessageAndPublishesPaymentEvent(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	events, err := run(context.Background(), &out)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}

	event := events[0]
	if !event.Allowed {
		t.Fatalf("event.Allowed = false, want true")
	}
	if event.TotalAppliedAmount != 120000 {
		t.Fatalf("TotalAppliedAmount = %d, want 120000", event.TotalAppliedAmount)
	}
	if len(event.Installments) != 3 {
		t.Fatalf("len(Installments) = %d, want 3", len(event.Installments))
	}
	if event.Installments[0].InstallmentID != "inst-001" {
		t.Fatalf("first installment = %q, want inst-001", event.Installments[0].InstallmentID)
	}
	last := event.Installments[len(event.Installments)-1]
	if last.InstallmentID != "inst-003" || !last.Partial || last.AppliedAmount != 30000 {
		t.Fatalf("last installment = %+v, want partial payment of 30000 for inst-003", last)
	}

	got := out.String()
	for _, want := range []string{
		"processed=2",
		"deleted=2",
		"allowed=true",
		"reason=payment_allowed",
		"firstTotalApplied=120000",
		"firstInstallment=inst-001",
		"firstLastPartial=true",
		"metric=increment name=student_financing.payment.completed",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
