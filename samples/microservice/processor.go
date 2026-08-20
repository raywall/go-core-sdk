// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/microservice implements the sample payment processing workflow.
//
// This file is part of the Microservice sample bounded context within the
// Samples service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/raywall/go-core-sdk/core"
	consumertypes "github.com/raywall/go-core-sdk/services/consumer/types"
	decisiontypes "github.com/raywall/go-core-sdk/services/decision/types"
	"github.com/raywall/go-core-sdk/services/parser"
	selectortypes "github.com/raywall/go-core-sdk/services/selector/types"
)

const dateLayout = "2006-01-02"

type paymentProcessor struct {
	runtime        *core.Core
	financingAPI   string
	paymentQueue   string
	tokenManagerID string
}

func (p *paymentProcessor) processS3Notification(ctx context.Context, event s3Notification) error {
	startedAt := time.Now()
	if err := p.runtime.Validator().Validate(ctx, event); err != nil {
		_ = p.runtime.Observability().Increment(ctx, "s3_notification.validation_failed")
		return err
	}

	p.runtime.Logger().InfoContext(ctx, "microservice_s3_notification_received", "records", len(event.Records))
	_ = p.runtime.Observability().Count(ctx, "s3_notification.received", int64(len(event.Records)))

	for _, record := range event.Records {
		if err := p.processRecord(ctx, record); err != nil {
			_ = p.runtime.Observability().Increment(ctx, "payment_processing.failed", "stage:record")
			return err
		}
	}

	duration := time.Since(startedAt)
	_ = p.runtime.Observability().Timing(ctx, "payment_processing.duration", duration)
	p.runtime.Logger().InfoContext(ctx, "microservice_s3_notification_processed", "records", len(event.Records), "duration", duration.String())
	return nil
}

func (p *paymentProcessor) processRecord(ctx context.Context, record s3NotificationRecord) error {
	p.runtime.Logger().InfoContext(ctx, "microservice_record_processing_started", "bucket", record.S3.Bucket.Name, "key", record.S3.Object.Key)

	instruction, err := p.loadInstruction(ctx, record)
	if err != nil {
		return err
	}

	financing, err := p.loadFinancing(ctx, instruction)
	if err != nil {
		return err
	}

	_, selection, err := p.selectInstallments(ctx, instruction, financing)
	if err != nil {
		return err
	}
	_ = p.runtime.Observability().Gauge(ctx, "installments.selected", float64(len(selection.Payments)))
	_ = p.runtime.Observability().Gauge(ctx, "payment.applied_amount", float64(selection.TotalAppliedAmount))

	decisionResult, err := p.evaluateBusinessRules(ctx, instruction, financing, selection)
	if err != nil {
		return err
	}
	if !decisionResult.Allowed {
		_ = p.runtime.Observability().Increment(ctx, "business_rules.denied")
		return fmt.Errorf("business rules denied payment event")
	}
	_ = p.runtime.Observability().Increment(ctx, "business_rules.allowed")

	event := buildPaymentEvent(instruction, selection)
	if err := p.publishPaymentEvent(ctx, event); err != nil {
		return err
	}

	p.runtime.Logger().InfoContext(ctx, "microservice_record_processing_completed",
		"instruction_id", instruction.InstructionID,
		"financing_id", instruction.FinancingID,
		"payments", len(event.Installments),
		"total_applied", event.TotalAppliedAmount,
		"remaining", event.RemainingAmount,
	)
	return nil
}

func (p *paymentProcessor) loadInstruction(ctx context.Context, record s3NotificationRecord) (paymentInstruction, error) {
	object, err := p.runtime.Consumer().GetS3(ctx, consumertypes.S3GetInput{
		Bucket: record.S3.Bucket.Name,
		Key:    record.S3.Object.Key,
	})
	if err != nil {
		_ = p.runtime.Observability().Increment(ctx, "s3_object.load_failed")
		return paymentInstruction{}, err
	}
	_ = p.runtime.Observability().Gauge(ctx, "s3_object.bytes", float64(len(object.Body)))

	var instructionDTO paymentInstructionDTO
	if err := json.Unmarshal(object.Body, &instructionDTO); err != nil {
		_ = p.runtime.Observability().Increment(ctx, "s3_object.decode_failed")
		return paymentInstruction{}, err
	}
	if err := p.runtime.Validator().Validate(ctx, instructionDTO); err != nil {
		_ = p.runtime.Observability().Increment(ctx, "payment_instruction_dto.validation_failed")
		return paymentInstruction{}, err
	}

	instruction, err := parser.ParseAs[paymentInstruction](ctx, instructionDTO, parser.WithLogger(p.runtime.Logger()))
	if err != nil {
		_ = p.runtime.Observability().Increment(ctx, "payment_instruction.parse_failed")
		return paymentInstruction{}, err
	}
	if err := p.runtime.Validator().Validate(ctx, instruction); err != nil {
		_ = p.runtime.Observability().Increment(ctx, "payment_instruction.validation_failed")
		return paymentInstruction{}, err
	}

	p.runtime.Logger().InfoContext(ctx, "microservice_payment_instruction_loaded",
		"instruction_id", instruction.InstructionID,
		"worker_id", instruction.WorkerID,
		"financing_id", instruction.FinancingID,
		"available_amount", instruction.AvailableAmount,
	)
	_ = p.runtime.Observability().Increment(ctx, "payment_instruction.loaded")
	return instruction, nil
}

func (p *paymentProcessor) loadFinancing(ctx context.Context, instruction paymentInstruction) (financingResponse, error) {
	manager, ok := p.runtime.TokenManager(p.tokenManagerID)
	if !ok {
		return financingResponse{}, fmt.Errorf("token manager %q not found", p.tokenManagerID)
	}

	url := strings.TrimRight(p.financingAPI, "/") + "/financings/" + instruction.FinancingID
	response, err := p.runtime.Consumer().REST(http.MethodGet, url).
		WithHeader("Authorization", manager.Token().ToString()).
		WithHeader("Accept", "application/json").
		Do(ctx)
	if err != nil {
		_ = p.runtime.Observability().Increment(ctx, "student_financing_api.failed")
		return financingResponse{}, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = p.runtime.Observability().Increment(ctx, "student_financing_api.failed", fmt.Sprintf("status:%d", response.StatusCode))
		return financingResponse{}, fmt.Errorf("student financing API returned %s", response.Status)
	}

	var financing financingResponse
	if err := response.DecodeJSON(&financing); err != nil {
		_ = p.runtime.Observability().Increment(ctx, "student_financing_api.decode_failed")
		return financingResponse{}, err
	}
	if err := p.runtime.Validator().Validate(ctx, financing); err != nil {
		_ = p.runtime.Observability().Increment(ctx, "student_financing_api.validation_failed")
		return financingResponse{}, err
	}

	p.runtime.Logger().InfoContext(ctx, "microservice_financing_loaded",
		"financing_id", financing.FinancingID,
		"borrower_id", financing.BorrowerID,
		"installments", len(financing.Installments),
	)
	_ = p.runtime.Observability().Increment(ctx, "student_financing_api.loaded")
	return financing, nil
}

func (p *paymentProcessor) selectInstallments(ctx context.Context, instruction paymentInstruction, financing financingResponse) ([]selectortypes.Item, selectortypes.SelectionResult[selectortypes.Item], error) {
	items := payableInstallmentItems(financing.Installments)
	ordered, selection, err := p.runtime.Selector().SortAndSelectItems(ctx, items,
		selectortypes.SortConfig{
			Path:       "dueDate",
			Kind:       selectortypes.KindTime,
			Direction:  selectortypes.Ascending,
			TimeLayout: dateLayout,
		},
		selectortypes.SelectionConfig{
			AmountPath:      "amountDue",
			AvailableAmount: instruction.AvailableAmount,
			Mode:            selectortypes.ModePartial,
		},
	)
	if err != nil {
		_ = p.runtime.Observability().Increment(ctx, "installments.selection_failed")
		return nil, selectortypes.SelectionResult[selectortypes.Item]{}, err
	}

	p.runtime.Logger().InfoContext(ctx, "microservice_installments_selected",
		"eligible_installments", len(items),
		"selected_installments", len(selection.Payments),
		"total_applied", selection.TotalAppliedAmount,
		"remaining", selection.RemainingAmount,
	)
	_ = p.runtime.Observability().Increment(ctx, "installments.selection_completed")
	return ordered, selection, nil
}

func (p *paymentProcessor) evaluateBusinessRules(ctx context.Context, instruction paymentInstruction, financing financingResponse, selection selectortypes.SelectionResult[selectortypes.Item]) (decisiontypes.EvaluationResult, error) {
	result, err := p.runtime.Decision().Evaluate(ctx, decisiontypes.EvaluationInput{
		Rule: decisiontypes.Rule{
			Name:        "student-financing-payment-allowed",
			Description: "Payment must belong to the borrower and apply a positive amount to a controlled number of installments.",
			Expression:  "instruction.availableAmount > 0 && financing.borrowerId == instruction.workerId && selection.totalAppliedAmount > 0 && selection.paymentCount > 0 && selection.paymentCount <= 5",
		},
		Entities: map[string]any{
			"instruction": map[string]any{
				"availableAmount": instruction.AvailableAmount,
				"workerId":        instruction.WorkerID,
			},
			"financing": map[string]any{
				"borrowerId": financing.BorrowerID,
			},
			"selection": map[string]any{
				"paymentCount":       len(selection.Payments),
				"totalAppliedAmount": selection.TotalAppliedAmount,
			},
		},
	})
	if err != nil {
		_ = p.runtime.Observability().Increment(ctx, "business_rules.evaluation_failed")
		return decisiontypes.EvaluationResult{}, err
	}
	return result, nil
}

func (p *paymentProcessor) publishPaymentEvent(ctx context.Context, event paymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	output, err := p.runtime.Consumer().SendSQS(ctx, consumertypes.SQSSendInput{
		QueueURL: p.paymentQueue,
		Body:     string(body),
		MessageAttributes: map[string]string{
			"eventType":   event.EventType,
			"financingId": event.FinancingID,
		},
	})
	if err != nil {
		_ = p.runtime.Observability().Increment(ctx, "payment_event.publish_failed")
		return err
	}

	p.runtime.Logger().InfoContext(ctx, "microservice_payment_event_published",
		"message_id", output.MessageID,
		"event_type", event.EventType,
		"financing_id", event.FinancingID,
	)
	_ = p.runtime.Observability().Increment(ctx, "payment_event.published")
	return nil
}

func payableInstallmentItems(installments []installment) []selectortypes.Item {
	items := make([]selectortypes.Item, 0, len(installments))
	for _, item := range installments {
		status := strings.ToUpper(strings.TrimSpace(item.Status))
		if status != "OPEN" && status != "OVERDUE" {
			continue
		}
		items = append(items, selectortypes.Item{
			"installmentId": item.InstallmentID,
			"dueDate":       item.DueDate,
			"amountDue":     item.AmountDue,
			"status":        status,
		})
	}
	return items
}

func buildPaymentEvent(instruction paymentInstruction, selection selectortypes.SelectionResult[selectortypes.Item]) paymentEvent {
	payments := make([]selectedInstallment, 0, len(selection.Payments))
	for _, payment := range selection.Payments {
		payments = append(payments, selectedInstallment{
			InstallmentID:  itemString(payment.Item, "installmentId"),
			DueDate:        itemString(payment.Item, "dueDate"),
			Status:         itemString(payment.Item, "status"),
			RequiredAmount: payment.RequiredAmount,
			AppliedAmount:  payment.AppliedAmount,
			Partial:        payment.Partial,
		})
	}
	return paymentEvent{
		EventType:          "StudentFinancingPaymentRequested",
		InstructionID:      instruction.InstructionID,
		WorkerID:           instruction.WorkerID,
		FinancingID:        instruction.FinancingID,
		TotalAppliedAmount: selection.TotalAppliedAmount,
		RemainingAmount:    selection.RemainingAmount,
		Installments:       payments,
	}
}

func itemString(item selectortypes.Item, key string) string {
	value, _ := item[key].(string)
	return value
}
