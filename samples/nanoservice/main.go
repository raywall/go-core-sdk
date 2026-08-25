// Copyright (c) 2026 Raywall Malheiros de Souza. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements an executable event-driven SDK composition sample.
//
// This file is part of the Nanoservice sample bounded context within the
// Samples service.
//
// Author:  Raywall Malheiros
// Created: 2026-08-25
// Updated: 2026-08-25

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/core"
	"github.com/raywall/go-core-sdk/handlers"
	"github.com/raywall/go-core-sdk/handlers/worker"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	"github.com/raywall/go-core-sdk/services/observability"
)

const (
	defaultTokenManagerID = "financing-api"
	defaultInputQueueURL  = "https://sqs.us-east-1.amazonaws.com/123456789012/payment-input"
	defaultOutputQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/payment-output"
	defaultSecretID       = "nanoservice/financing-api"
)

// Funcionamento:
// 1. Sera provisionado em um ECS
// 2. Ira utilizar um token management para gerenciamento de token STS
// 3. Vai receber eventos SQS em lote de uma fila
// 4. Precisara deserializar o DTO e valida os dados usando o validation
// 5. Com os dados validos, precisa passar os dados para a entidade
// 6. Ira ordenar a lista e selecionar itens com base no valor usando o selector (aceita parcial)
// 7. Enviara eventos SQS para outros sistemas
// 8. Deve registrar metricas customizadas no datadog
// 9. Deve usar log estruturado
func main() {
	if _, err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, out io.Writer) ([]PaymentEvent, error) {
	sts := newSTS()
	defer sts.Close()

	financingAPI := newFinancingAPI()
	defer financingAPI.Close()

	outboundSQS := &fakeAWSSQSClient{}
	sourceQueue := &fakeSourceQueue{
		messages: []consumeraws.SQSMessage{
			{
				MessageID:     "payment-message-1",
				ReceiptHandle: "payment-receipt-1",
				Body:          sampleCommandJSON("req-001", 120000),
			},
			{
				MessageID:     "payment-message-2",
				ReceiptHandle: "payment-receipt-2",
				Body:          sampleCommandJSON("req-002", 70000),
			},
		},
	}

	cfg, err := config.Load(ctx,
		config.WithServiceName("student-financing-payment-nano"),
		config.WithEnvironment("local"),
		config.WithVersion("1.0.0"),
		config.WithAWSRegion("us-east-1"),
		config.WithObservability(config.ObservabilityConfig{
			MetricPrefix:   "student_financing",
			DatadogAddress: "127.0.0.1:8125",
			DefaultTags:    []string{"runtime:ecs", "sample:nanoservice"},
		}),
		config.WithToken(defaultTokenManagerID, config.TokenConfig{
			BaseURL:        sts.URL,
			Endpoint:       "/oauth/token",
			ValidateSSL:    true,
			SecretID:       defaultSecretID,
			RequestTimeout: 3 * time.Second,
			RefreshBefore:  30 * time.Second,
		}),
	)
	if err != nil {
		return nil, err
	}

	runtime, err := core.New(ctx, cfg,
		core.WithAWSOptions(
			consumeraws.WithSecretsManagerClient(fakeSecretsManagerClient{
				secretID: defaultSecretID,
				value:    `{"client_id":"nano-client","client_secret":"nano-secret"}`,
			}),
			consumeraws.WithSQSClient(outboundSQS),
		),
		core.WithObservabilityOptions(
			observability.WithMetricsClient(stdoutMetricsClient{out: out}),
			observability.WithWriter(io.Discard),
		),
		core.WithTokenAutoStart(true),
	)
	if err != nil {
		return nil, err
	}
	defer runtime.Stop()

	useCase := PaymentProcessingUseCase{
		Validator: runtime.Validator(),
		Financing: financingAPIAdapter{
			runtime:        runtime,
			baseURL:        financingAPI.URL,
			tokenManagerID: defaultTokenManagerID,
		},
		Selector:  selectorAdapter{runtime: runtime},
		Decision:  decisionAdapter{runtime: runtime},
		Publisher: sqsPaymentPublisher{runtime: runtime, queueURL: defaultOutputQueueURL},
		Metrics:   runtime.Observability(),
	}
	processor := paymentProcessor{
		logger:  runtime.Logger(),
		useCase: useCase,
	}
	runner, err := worker.NewSQSRunner(handlers.Config{
		SQSQueueURL:        defaultInputQueueURL,
		SQSMaxMessages:     10,
		SQSWaitTimeSeconds: 1,
	}, processor, sourceQueue)
	if err != nil {
		return nil, err
	}
	if err := runner.RunOnce(ctx); err != nil {
		return nil, err
	}
	if len(outboundSQS.sent) == 0 {
		return nil, fmt.Errorf("no outbound payment event published")
	}

	events := make([]PaymentEvent, 0, len(outboundSQS.sent))
	for _, message := range outboundSQS.sent {
		var event PaymentEvent
		if err := json.Unmarshal([]byte(message.body), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	first := events[0]
	_, err = fmt.Fprintf(out, "processed=%d deleted=%d allowed=%t reason=%s firstTotalApplied=%d firstInstallment=%s firstLastPartial=%t\n",
		len(outboundSQS.sent),
		len(sourceQueue.deleted),
		first.Allowed,
		first.Reason,
		first.TotalAppliedAmount,
		first.Installments[0].InstallmentID,
		first.Installments[len(first.Installments)-1].Partial,
	)
	return events, err
}

func sampleCommandJSON(requestID string, availableAmount int64) string {
	body, _ := json.Marshal(PaymentRequestDTO{
		RequestID:       requestID,
		FinancingID:     "fin-123",
		WorkerID:        "worker-123",
		AvailableAmount: availableAmount,
		RequestedBy:     "sample-nanoservice",
		SourceSystem:    "local",
	})
	return string(body)
}
