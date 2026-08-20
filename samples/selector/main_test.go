package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunSortsAndSelectsInstallmentsWithPartialPayment(t *testing.T) {
	t.Parallel()

	installments := []Installment{
		{Number: 3, Status: "OPEN", DueDate: mustDate("2026-03-01"), AmountCents: 10_000},
		{Number: 1, Status: "OPEN", DueDate: mustDate("2026-01-01"), AmountCents: 10_000},
		{Number: 2, Status: "OPEN", DueDate: mustDate("2026-02-01"), AmountCents: 10_000},
	}

	var out bytes.Buffer
	if err := run(context.Background(), &out, installments, 25_000); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"firstDue=2026-01-01 payments=3 applied=25000 remaining=0",
		"installment=3 applied=5000 partial=true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
