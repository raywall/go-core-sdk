// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements sample data contracts.
//
// This file is part of the Nanoservice sample bounded context within the
// Samples package.
//
// Author:  Raywall
// Created: 2026-08-25
// Updated: 2026-08-25

package main

// PaymentRequestDTO is the command received from the inbound SQS message.
type PaymentRequestDTO struct {
	// RequestID identifies the payment request.
	RequestID string `json:"requestId" validate:"required"`
	// FinancingID identifies the student financing to query.
	FinancingID string `json:"financingId" validate:"required"`
	// WorkerID identifies the financing borrower.
	WorkerID string `json:"workerId" validate:"required"`
	// AvailableAmount is the amount available to apply in cents.
	AvailableAmount int64 `json:"availableAmount" validate:"required,gt=0"`
	// RequestedBy identifies the upstream requester.
	RequestedBy string `json:"requestedBy" validate:"required"`
	// SourceSystem contains optional transport metadata.
	SourceSystem string `json:"sourceSystem"`
}

// PaymentInstruction is the internal entity used by the use case.
type PaymentInstruction struct {
	// RequestID identifies the payment request.
	RequestID string `json:"requestId" validate:"required"`
	// FinancingID identifies the student financing to query.
	FinancingID string `json:"financingId" validate:"required"`
	// WorkerID identifies the financing borrower.
	WorkerID string `json:"workerId" validate:"required"`
	// AvailableAmount is the amount available to apply in cents.
	AvailableAmount int64 `json:"availableAmount" validate:"required,gt=0"`
	// RequestedBy identifies the upstream requester.
	RequestedBy string `json:"requestedBy" validate:"required"`
}

// Financing contains the external student financing data used by the use case.
type Financing struct {
	// FinancingID identifies the financing contract.
	FinancingID string `json:"financingId" validate:"required"`
	// BorrowerID identifies the borrower that owns the financing.
	BorrowerID string `json:"borrowerId" validate:"required"`
	// Status identifies whether the financing can receive payments.
	Status string `json:"status" validate:"required"`
	// Product identifies the financing product.
	Product string `json:"product" validate:"required"`
	// Installments contains the financing installment schedule.
	Installments []Installment `json:"installments" validate:"required,min=1,dive"`
}

// Installment represents one financing installment.
type Installment struct {
	// InstallmentID identifies the installment.
	InstallmentID string `json:"installmentId" validate:"required"`
	// DueDate is the installment due date in YYYY-MM-DD format.
	DueDate string `json:"dueDate" validate:"required"`
	// AmountDue is the open amount in cents.
	AmountDue int64 `json:"amountDue" validate:"required,gt=0"`
	// Status is the installment lifecycle status.
	Status string `json:"status" validate:"required"`
}

// InstallmentSelection is the application-level result of ordering and applying a payment amount.
type InstallmentSelection struct {
	// Ordered contains payable installments sorted from oldest to newest.
	Ordered []Installment
	// Payments contains selected installments and applied amounts.
	Payments []InstallmentPayment
	// RemainingAmount is the available amount left after selection.
	RemainingAmount int64
	// TotalAppliedAmount is the total amount selected for payment.
	TotalAppliedAmount int64
	// EligibleInstallment is the number of open or overdue installments considered.
	EligibleInstallment int
}

// InstallmentPayment describes one selected installment in the payment allocation.
type InstallmentPayment struct {
	// Installment is the selected installment.
	Installment Installment
	// RequiredAmount is the full amount required by the installment.
	RequiredAmount int64
	// AppliedAmount is the amount applied in cents.
	AppliedAmount int64
	// Partial indicates whether AppliedAmount is lower than RequiredAmount.
	Partial bool
}

// PaymentInstallment is the outbound event representation of a selected installment.
type PaymentInstallment struct {
	// InstallmentID identifies the selected installment.
	InstallmentID string `json:"installmentId"`
	// DueDate is the selected installment due date.
	DueDate string `json:"dueDate"`
	// Status is the source installment status.
	Status string `json:"status"`
	// RequiredAmount is the full amount required in cents.
	RequiredAmount int64 `json:"requiredAmount"`
	// AppliedAmount is the selected amount in cents.
	AppliedAmount int64 `json:"appliedAmount"`
	// Partial indicates whether the installment received a partial payment.
	Partial bool `json:"partial"`
}

// PaymentEvent is the outbound event published after business rule validation.
type PaymentEvent struct {
	// EventType identifies the outbound domain event.
	EventType string `json:"eventType"`
	// RequestID identifies the source payment request.
	RequestID string `json:"requestId"`
	// FinancingID identifies the financing contract.
	FinancingID string `json:"financingId"`
	// WorkerID identifies the financing borrower.
	WorkerID string `json:"workerId"`
	// Allowed indicates whether business rules approved the event.
	Allowed bool `json:"allowed"`
	// Reason explains the business decision.
	Reason string `json:"reason"`
	// TotalAppliedAmount is the total amount selected for payment.
	TotalAppliedAmount int64 `json:"totalAppliedAmount"`
	// RemainingAmount is the available amount left after selection.
	RemainingAmount int64 `json:"remainingAmount"`
	// Installments contains the selected installment payments.
	Installments []PaymentInstallment `json:"installments"`
}

// EligibilityResult contains the business decision result.
type EligibilityResult struct {
	// Allowed indicates whether the payment can proceed.
	Allowed bool
	// Reason explains the business decision.
	Reason string
}

func (dto PaymentRequestDTO) toInstruction() PaymentInstruction {
	return PaymentInstruction{
		RequestID:       dto.RequestID,
		FinancingID:     dto.FinancingID,
		WorkerID:        dto.WorkerID,
		AvailableAmount: dto.AvailableAmount,
		RequestedBy:     dto.RequestedBy,
	}
}
