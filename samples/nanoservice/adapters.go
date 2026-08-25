// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements outbound adapters.
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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/raywall/go-core-sdk/core"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	decisiontypes "github.com/raywall/go-core-sdk/services/decision/types"
	selectortypes "github.com/raywall/go-core-sdk/services/selector/types"
)

const dateLayout = "2006-01-02"

type financingAPIAdapter struct {
	runtime        *core.Core
	baseURL        string
	tokenManagerID string
}

func (a financingAPIAdapter) GetFinancing(ctx context.Context, financingID string) (Financing, error) {
	manager, ok := a.runtime.TokenManager(a.tokenManagerID)
	if !ok {
		return Financing{}, fmt.Errorf("token manager %q not found", a.tokenManagerID)
	}
	url := strings.TrimRight(a.baseURL, "/") + "/financings/" + strings.TrimSpace(financingID)
	response, err := a.runtime.REST().REST(http.MethodGet, url).
		WithHeader("Authorization", manager.Token().ToString()).
		WithHeader("Accept", "application/json").
		Do(ctx)
	if err != nil {
		return Financing{}, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Financing{}, fmt.Errorf("financing API returned %s", response.Status)
	}

	var financing Financing
	if err := response.DecodeJSON(&financing); err != nil {
		return Financing{}, err
	}
	return financing, nil
}

type selectorAdapter struct {
	runtime *core.Core
}

func (a selectorAdapter) SelectInstallments(ctx context.Context, installments []Installment, availableAmount int64) (InstallmentSelection, error) {
	items := payableInstallmentItems(installments)
	ordered, result, err := a.runtime.Selector().SortAndSelectItems(ctx, items,
		selectortypes.SortConfig{
			Path:       "dueDate",
			Kind:       selectortypes.KindTime,
			Direction:  selectortypes.Ascending,
			TimeLayout: dateLayout,
		},
		selectortypes.SelectionConfig{
			AmountPath:      "amountDue",
			AvailableAmount: availableAmount,
			Mode:            selectortypes.ModePartial,
		},
	)
	if err != nil {
		return InstallmentSelection{}, err
	}

	payments := make([]InstallmentPayment, 0, len(result.Payments))
	for _, payment := range result.Payments {
		payments = append(payments, InstallmentPayment{
			Installment:    installmentFromItem(payment.Item),
			RequiredAmount: payment.RequiredAmount,
			AppliedAmount:  payment.AppliedAmount,
			Partial:        payment.Partial,
		})
	}
	return InstallmentSelection{
		Ordered:             installmentsFromItems(ordered),
		Payments:            payments,
		RemainingAmount:     result.RemainingAmount,
		TotalAppliedAmount:  result.TotalAppliedAmount,
		EligibleInstallment: len(items),
	}, nil
}

type decisionAdapter struct {
	runtime *core.Core
}

func (a decisionAdapter) EvaluateEligibility(ctx context.Context, instruction PaymentInstruction, financing Financing, selection InstallmentSelection) (EligibilityResult, error) {
	result, err := a.runtime.Decision().Evaluate(ctx, decisiontypes.EvaluationInput{
		Rule: decisiontypes.Rule{
			Name:        "student-financing-payment-allowed",
			Description: "Payment must belong to the borrower, target active financing and select between one and five payable installments.",
			Expression:  "instruction.workerId == financing.borrowerId && financing.status == 'ACTIVE' && selection.totalAppliedAmount > 0 && selection.paymentCount > 0 && selection.paymentCount <= 5",
		},
		Entities: map[string]any{
			"instruction": map[string]any{
				"workerId": instruction.WorkerID,
			},
			"financing": map[string]any{
				"borrowerId": financing.BorrowerID,
				"status":     financing.Status,
			},
			"selection": map[string]any{
				"paymentCount":       len(selection.Payments),
				"totalAppliedAmount": selection.TotalAppliedAmount,
			},
		},
	})
	if err != nil {
		return EligibilityResult{}, err
	}
	reason := "payment_denied"
	if result.Allowed {
		reason = "payment_allowed"
	}
	return EligibilityResult{Allowed: result.Allowed, Reason: reason}, nil
}

type sqsPaymentPublisher struct {
	runtime  *core.Core
	queueURL string
}

func (p sqsPaymentPublisher) PublishPaymentRequested(ctx context.Context, event PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = p.runtime.AWS().SendSQS(ctx, consumeraws.SQSSendInput{
		QueueURL: p.queueURL,
		Body:     string(body),
		MessageAttributes: map[string]string{
			"eventType":   event.EventType,
			"financingId": event.FinancingID,
			"allowed":     boolTag(event.Allowed),
		},
	})
	return err
}

func payableInstallmentItems(installments []Installment) []selectortypes.Item {
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

func installmentsFromItems(items []selectortypes.Item) []Installment {
	installments := make([]Installment, 0, len(items))
	for _, item := range items {
		installments = append(installments, installmentFromItem(item))
	}
	return installments
}

func installmentFromItem(item selectortypes.Item) Installment {
	return Installment{
		InstallmentID: itemString(item, "installmentId"),
		DueDate:       itemString(item, "dueDate"),
		AmountDue:     itemInt64(item, "amountDue"),
		Status:        itemString(item, "status"),
	}
}

func itemString(item selectortypes.Item, key string) string {
	value, _ := item[key].(string)
	return value
}

func itemInt64(item selectortypes.Item, key string) int64 {
	switch value := item[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}
