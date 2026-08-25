// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice tests the payment processing use case.
//
// This file is part of the Nanoservice sample bounded context within the
// Samples package.
//
// Author:  Raywall
// Created: 2026-08-25
// Updated: 2026-08-25

package main

import (
	"context"
	"testing"
)

func TestPaymentProcessingUseCase_ExecutePublishesPartialPayment(t *testing.T) {
	t.Parallel()

	publisher := &fakePaymentPublisher{}
	useCase := PaymentProcessingUseCase{
		Validator: fakeValidator{},
		Financing: fakeFinancingReader{financing: Financing{
			FinancingID: "fin-123",
			BorrowerID:  "worker-123",
			Status:      "ACTIVE",
			Product:     "student-financing",
			Installments: []Installment{
				{InstallmentID: "inst-001", DueDate: "2026-01-10", AmountDue: 50000, Status: "OVERDUE"},
				{InstallmentID: "inst-002", DueDate: "2026-02-10", AmountDue: 40000, Status: "OPEN"},
				{InstallmentID: "inst-003", DueDate: "2026-03-10", AmountDue: 60000, Status: "OPEN"},
			},
		}},
		Selector: fakeInstallmentSelector{selection: InstallmentSelection{
			Payments: []InstallmentPayment{
				{Installment: Installment{InstallmentID: "inst-001", DueDate: "2026-01-10", Status: "OVERDUE"}, RequiredAmount: 50000, AppliedAmount: 50000},
				{Installment: Installment{InstallmentID: "inst-002", DueDate: "2026-02-10", Status: "OPEN"}, RequiredAmount: 40000, AppliedAmount: 40000},
				{Installment: Installment{InstallmentID: "inst-003", DueDate: "2026-03-10", Status: "OPEN"}, RequiredAmount: 60000, AppliedAmount: 30000, Partial: true},
			},
			TotalAppliedAmount: 120000,
		}},
		Decision:  fakeEligibilityEvaluator{result: EligibilityResult{Allowed: true, Reason: "payment_allowed"}},
		Publisher: publisher,
		Metrics:   fakeMetrics{},
	}

	event, err := useCase.Execute(context.Background(), PaymentRequestDTO{
		RequestID:       "req-001",
		FinancingID:     "fin-123",
		WorkerID:        "worker-123",
		AvailableAmount: 120000,
		RequestedBy:     "test",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if event.TotalAppliedAmount != 120000 {
		t.Fatalf("TotalAppliedAmount = %d, want 120000", event.TotalAppliedAmount)
	}
	if len(event.Installments) != 3 || !event.Installments[2].Partial {
		t.Fatalf("Installments = %+v, want third installment partial", event.Installments)
	}
	if len(publisher.events) != 1 || publisher.events[0].RequestID != "req-001" {
		t.Fatalf("published events = %+v, want one event for req-001", publisher.events)
	}
}

type fakeValidator struct{}

func (fakeValidator) Validate(context.Context, any) error {
	return nil
}

type fakeFinancingReader struct {
	financing Financing
}

func (f fakeFinancingReader) GetFinancing(context.Context, string) (Financing, error) {
	return f.financing, nil
}

type fakeInstallmentSelector struct {
	selection InstallmentSelection
}

func (f fakeInstallmentSelector) SelectInstallments(context.Context, []Installment, int64) (InstallmentSelection, error) {
	return f.selection, nil
}

type fakeEligibilityEvaluator struct {
	result EligibilityResult
}

func (f fakeEligibilityEvaluator) EvaluateEligibility(context.Context, PaymentInstruction, Financing, InstallmentSelection) (EligibilityResult, error) {
	return f.result, nil
}

type fakePaymentPublisher struct {
	events []PaymentEvent
}

func (p *fakePaymentPublisher) PublishPaymentRequested(_ context.Context, event PaymentEvent) error {
	p.events = append(p.events, event)
	return nil
}

type fakeMetrics struct{}

func (fakeMetrics) Increment(context.Context, string, ...string) error {
	return nil
}

func (fakeMetrics) Gauge(context.Context, string, float64, ...string) error {
	return nil
}
