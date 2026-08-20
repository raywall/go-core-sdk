package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRunParsesProposalDTOIntoDomainEntity(t *testing.T) {
	t.Parallel()

	dto := ProposalDTO{
		ID:             "proposal-123",
		Customer:       CustomerDTO{Document: "12345678901", Name: "Ana"},
		RequestedCents: 75000,
	}

	var out bytes.Buffer
	if err := run(context.Background(), &out, dto); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got, want := out.String(), "proposal=proposal-123 customer=12345678901 amount=75000\n"; got != want {
		t.Fatalf("run() output = %q, want %q", got, want)
	}
}
