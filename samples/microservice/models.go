// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/microservice implements sample data contracts.
//
// This file is part of the Microservice sample bounded context within the
// Samples service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package main

// s3Notification is the subset of an S3 notification event used by the sample.
type s3Notification struct {
	// Records contains object-created records.
	Records []s3NotificationRecord `json:"Records" validate:"required,min=1,dive"`
}

// s3NotificationRecord contains one S3 object reference.
type s3NotificationRecord struct {
	// EventName identifies the S3 event type.
	EventName string `json:"eventName" validate:"required"`
	// S3 contains bucket and object metadata.
	S3 s3Entity `json:"s3" validate:"required"`
}

// s3Entity contains the S3 bucket and object references.
type s3Entity struct {
	// Bucket identifies the source bucket.
	Bucket s3Bucket `json:"bucket" validate:"required"`
	// Object identifies the source object.
	Object s3Object `json:"object" validate:"required"`
}

// s3Bucket contains bucket metadata from an S3 notification.
type s3Bucket struct {
	// Name is the source bucket name.
	Name string `json:"name" validate:"required"`
}

// s3Object contains object metadata from an S3 notification.
type s3Object struct {
	// Key is the source object key.
	Key string `json:"key" validate:"required"`
}

// paymentInstructionDTO is the JSON document stored in S3.
type paymentInstructionDTO struct {
	// InstructionID is the business idempotency key for this request.
	InstructionID string `json:"instructionId" validate:"required"`
	// WorkerID identifies the borrower that owns the student financing.
	WorkerID string `json:"workerId" validate:"required"`
	// FinancingID identifies the student financing to query in the partner API.
	FinancingID string `json:"financingId" validate:"required"`
	// AvailableAmount is the amount available for payment in cents.
	AvailableAmount int64 `json:"availableAmount" validate:"required,gt=0"`
	// RequestedBy identifies the upstream system that requested processing.
	RequestedBy string `json:"requestedBy" validate:"required"`
	// SourceSystem is transport metadata ignored by the internal entity.
	SourceSystem string `json:"sourceSystem"`
}

// paymentInstruction is the internal instruction entity used by the sample.
type paymentInstruction struct {
	// InstructionID is the business idempotency key for this request.
	InstructionID string `json:"instructionId" validate:"required"`
	// WorkerID identifies the borrower that owns the student financing.
	WorkerID string `json:"workerId" validate:"required"`
	// FinancingID identifies the student financing to query in the partner API.
	FinancingID string `json:"financingId" validate:"required"`
	// AvailableAmount is the amount available for payment in cents.
	AvailableAmount int64 `json:"availableAmount" validate:"required,gt=0"`
	// RequestedBy identifies the upstream system that requested processing.
	RequestedBy string `json:"requestedBy" validate:"required"`
}

// financingResponse is returned by the protected student financing API.
type financingResponse struct {
	// FinancingID identifies the financing.
	FinancingID string `json:"financingId" validate:"required"`
	// BorrowerID identifies the borrower that owns the financing.
	BorrowerID string `json:"borrowerId" validate:"required"`
	// Product identifies the financing product.
	Product string `json:"product" validate:"required"`
	// Installments contains the financing installment schedule.
	Installments []installment `json:"installments" validate:"required,min=1,dive"`
}

// installment represents one student financing installment.
type installment struct {
	// InstallmentID identifies the installment.
	InstallmentID string `json:"installmentId" validate:"required"`
	// DueDate is the installment due date in YYYY-MM-DD format.
	DueDate string `json:"dueDate" validate:"required"`
	// AmountDue is the open amount in cents.
	AmountDue int64 `json:"amountDue" validate:"required,gt=0"`
	// Status is the installment lifecycle status.
	Status string `json:"status" validate:"required"`
}

// selectedInstallment describes one installment included in the payment event.
type selectedInstallment struct {
	// InstallmentID identifies the selected installment.
	InstallmentID string `json:"installmentId"`
	// DueDate is the selected installment due date.
	DueDate string `json:"dueDate"`
	// Status is the source installment status.
	Status string `json:"status"`
	// RequiredAmount is the full amount required in cents.
	RequiredAmount int64 `json:"requiredAmount"`
	// AppliedAmount is the amount applied in cents.
	AppliedAmount int64 `json:"appliedAmount"`
	// Partial indicates whether the installment received a partial payment.
	Partial bool `json:"partial"`
}

// paymentEvent is published to SQS after all business checks pass.
type paymentEvent struct {
	// EventType identifies the domain event.
	EventType string `json:"eventType"`
	// InstructionID is the source instruction id.
	InstructionID string `json:"instructionId"`
	// WorkerID identifies the borrower.
	WorkerID string `json:"workerId"`
	// FinancingID identifies the financing.
	FinancingID string `json:"financingId"`
	// TotalAppliedAmount is the total payment amount in cents.
	TotalAppliedAmount int64 `json:"totalAppliedAmount"`
	// RemainingAmount is the balance left after selecting installments.
	RemainingAmount int64 `json:"remainingAmount"`
	// Installments contains the selected installment payments.
	Installments []selectedInstallment `json:"installments"`
}
