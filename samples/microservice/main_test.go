package main

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
)

func TestRunProcessesStudentFinancingPaymentEvent(t *testing.T) {
	t.Parallel()

	var out safeBuffer
	event, err := run(context.Background(), &out)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if event.EventType != "StudentFinancingPaymentRequested" {
		t.Fatalf("event.EventType = %q", event.EventType)
	}
	if event.TotalAppliedAmount != 120000 {
		t.Fatalf("event.TotalAppliedAmount = %d", event.TotalAppliedAmount)
	}
	if got := len(event.Installments); got != 3 {
		t.Fatalf("len(event.Installments) = %d, want 3", got)
	}
	last := event.Installments[len(event.Installments)-1]
	if last.InstallmentID != "inst-004" || !last.Partial || last.AppliedAmount != 20000 {
		t.Fatalf("last installment = %+v", last)
	}

	got := out.String()
	for _, want := range []string{"microservice_payment_event_published", "metric=increment name=student_financing.payment_event.published"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
