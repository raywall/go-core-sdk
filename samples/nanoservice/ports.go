// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements application ports.
//
// This file is part of the Nanoservice sample bounded context within the
// Samples package.
//
// Author:  Raywall
// Created: 2026-08-25
// Updated: 2026-08-25

package main

import "context"

// Validator validates application commands and DTOs.
type Validator interface {
	// Validate validates a DTO or entity using application rules.
	Validate(context.Context, any) error
}

// FinancingReader loads financing data from an outbound dependency.
type FinancingReader interface {
	// GetFinancing loads financing data by id.
	GetFinancing(context.Context, string) (Financing, error)
}

// InstallmentSelector orders payable installments and applies an available amount.
type InstallmentSelector interface {
	// SelectInstallments orders payable installments and selects amounts.
	SelectInstallments(context.Context, []Installment, int64) (InstallmentSelection, error)
}

// EligibilityEvaluator evaluates whether the requested payment can proceed.
type EligibilityEvaluator interface {
	// EvaluateEligibility evaluates business rules for the selected payment.
	EvaluateEligibility(context.Context, PaymentInstruction, Financing, InstallmentSelection) (EligibilityResult, error)
}

// PaymentEventPublisher publishes the final payment event.
type PaymentEventPublisher interface {
	// PublishPaymentRequested publishes the payment event.
	PublishPaymentRequested(context.Context, PaymentEvent) error
}

// MetricsRecorder records custom business metrics.
type MetricsRecorder interface {
	// Increment records a counter increment with optional tags.
	Increment(context.Context, string, ...string) error
	// Gauge records a gauge value with optional tags.
	Gauge(context.Context, string, float64, ...string) error
}
