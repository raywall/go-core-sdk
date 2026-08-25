// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements the payment processing use case.
//
// This file is part of the Nanoservice sample bounded context within the
// Samples package.
//
// Author:  Raywall
// Created: 2026-08-25
// Updated: 2026-08-25

package main

import "context"

// PaymentProcessingUseCase orchestrates one small event-driven business action.
type PaymentProcessingUseCase struct {
	Validator Validator
	Financing FinancingReader
	Selector  InstallmentSelector
	Decision  EligibilityEvaluator
	Publisher PaymentEventPublisher
	Metrics   MetricsRecorder
}

// Execute validates the inbound DTO, builds an entity, selects installments and publishes an event.
func (u PaymentProcessingUseCase) Execute(ctx context.Context, dto PaymentRequestDTO) (PaymentEvent, error) {
	if err := u.Validator.Validate(ctx, dto); err != nil {
		_ = u.increment(ctx, "payment.validation_failed")
		return PaymentEvent{}, err
	}

	instruction := dto.toInstruction()
	if err := u.Validator.Validate(ctx, instruction); err != nil {
		_ = u.increment(ctx, "payment.entity_validation_failed")
		return PaymentEvent{}, err
	}

	financing, err := u.Financing.GetFinancing(ctx, instruction.FinancingID)
	if err != nil {
		_ = u.increment(ctx, "payment.financing_failed")
		return PaymentEvent{}, err
	}
	if err := u.Validator.Validate(ctx, financing); err != nil {
		_ = u.increment(ctx, "payment.financing_validation_failed")
		return PaymentEvent{}, err
	}

	selection, err := u.Selector.SelectInstallments(ctx, financing.Installments, instruction.AvailableAmount)
	if err != nil {
		_ = u.increment(ctx, "payment.installment_selection_failed")
		return PaymentEvent{}, err
	}

	eligibility, err := u.Decision.EvaluateEligibility(ctx, instruction, financing, selection)
	if err != nil {
		_ = u.increment(ctx, "payment.decision_failed")
		return PaymentEvent{}, err
	}

	event := buildPaymentEvent(instruction, eligibility, selection)
	if err := u.Publisher.PublishPaymentRequested(ctx, event); err != nil {
		_ = u.increment(ctx, "payment.publish_failed")
		return PaymentEvent{}, err
	}

	_ = u.increment(ctx, "payment.completed", "allowed:"+boolTag(event.Allowed))
	if u.Metrics != nil {
		_ = u.Metrics.Gauge(ctx, "payment.available_amount", float64(instruction.AvailableAmount))
		_ = u.Metrics.Gauge(ctx, "payment.total_applied_amount", float64(selection.TotalAppliedAmount))
	}
	return event, nil
}

func (u PaymentProcessingUseCase) increment(ctx context.Context, name string, tags ...string) error {
	if u.Metrics == nil {
		return nil
	}
	return u.Metrics.Increment(ctx, name, tags...)
}

func boolTag(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func buildPaymentEvent(instruction PaymentInstruction, eligibility EligibilityResult, selection InstallmentSelection) PaymentEvent {
	payments := make([]PaymentInstallment, 0, len(selection.Payments))
	for _, payment := range selection.Payments {
		payments = append(payments, PaymentInstallment{
			InstallmentID:  payment.Installment.InstallmentID,
			DueDate:        payment.Installment.DueDate,
			Status:         payment.Installment.Status,
			RequiredAmount: payment.RequiredAmount,
			AppliedAmount:  payment.AppliedAmount,
			Partial:        payment.Partial,
		})
	}
	return PaymentEvent{
		EventType:          "StudentFinancingPaymentRequested",
		RequestID:          instruction.RequestID,
		FinancingID:        instruction.FinancingID,
		WorkerID:           instruction.WorkerID,
		Allowed:            eligibility.Allowed,
		Reason:             eligibility.Reason,
		TotalAppliedAmount: selection.TotalAppliedAmount,
		RemainingAmount:    selection.RemainingAmount,
		Installments:       payments,
	}
}
