// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements the runtime-neutral event processor.
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
	"log/slog"

	"github.com/raywall/go-core-sdk/handlers"
)

type paymentProcessor struct {
	logger  *slog.Logger
	useCase PaymentProcessingUseCase
}

func (p paymentProcessor) Process(ctx context.Context, event handlers.Event) (handlers.Result, error) {
	result := handlers.Result{Metadata: map[string]string{"sample": "nanoservice"}}
	for _, record := range event.Records {
		var dto PaymentRequestDTO
		if err := json.Unmarshal(record.Body, &dto); err != nil {
			result.Failed++
			result.Failures = append(result.Failures, handlers.Failure{RecordID: record.ID, Error: err.Error()})
			continue
		}

		payment, err := p.useCase.Execute(ctx, dto)
		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, handlers.Failure{RecordID: record.ID, Error: err.Error()})
			continue
		}

		result.Processed++
		p.logger.InfoContext(ctx, "nanoservice_payment_processed",
			"request_id", payment.RequestID,
			"financing_id", payment.FinancingID,
			"selected_installments", len(payment.Installments),
			"total_applied", payment.TotalAppliedAmount,
			"allowed", payment.Allowed,
		)
	}
	return result, nil
}
