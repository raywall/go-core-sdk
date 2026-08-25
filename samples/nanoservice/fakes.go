// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/nanoservice implements local fakes for external integrations.
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
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
)

type fakeSecretsManagerClient struct {
	secretID string
	value    string
}

func (f fakeSecretsManagerClient) GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		Name:         aws.String(f.secretID),
		VersionId:    aws.String("version-1"),
		SecretString: aws.String(f.value),
	}, nil
}

func (fakeSecretsManagerClient) PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (fakeSecretsManagerClient) UpdateSecret(context.Context, *secretsmanager.UpdateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return &secretsmanager.UpdateSecretOutput{}, nil
}

type fakeAWSSQSClient struct {
	sent []sentMessage
}

type sentMessage struct {
	body       string
	attributes map[string]string
}

func (f *fakeAWSSQSClient) SendMessage(_ context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	message := sentMessage{
		body:       aws.ToString(input.MessageBody),
		attributes: make(map[string]string, len(input.MessageAttributes)),
	}
	for key, value := range input.MessageAttributes {
		message.attributes[key] = aws.ToString(value.StringValue)
	}
	f.sent = append(f.sent, message)
	return &sqs.SendMessageOutput{MessageId: aws.String(fmt.Sprintf("nanoservice-message-%d", len(f.sent)))}, nil
}

func (f *fakeAWSSQSClient) ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{}, nil
}

func (f *fakeAWSSQSClient) DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

type fakeSourceQueue struct {
	messages []consumeraws.SQSMessage
	deleted  []string
}

func (f *fakeSourceQueue) ReceiveSQS(context.Context, consumeraws.SQSReceiveInput) (consumeraws.SQSReceiveOutput, error) {
	return consumeraws.SQSReceiveOutput{Messages: f.messages}, nil
}

func (f *fakeSourceQueue) DeleteSQS(_ context.Context, input consumeraws.SQSDeleteInput) error {
	f.deleted = append(f.deleted, input.ReceiptHandle)
	return nil
}

type stdoutMetricsClient struct {
	out io.Writer
}

func (c stdoutMetricsClient) Count(name string, value int64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=count name=%s value=%d tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c stdoutMetricsClient) Incr(name string, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=increment name=%s tags=%v rate=%.1f\n", name, tags, rate)
	return err
}

func (c stdoutMetricsClient) Gauge(name string, value float64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=gauge name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c stdoutMetricsClient) Histogram(name string, value float64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=histogram name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c stdoutMetricsClient) Distribution(name string, value float64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=distribution name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c stdoutMetricsClient) Timing(name string, value time.Duration, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=timing name=%s value=%s tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func newSTS() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.PostForm.Get("client_id") != "nano-client" || r.PostForm.Get("client_secret") != "nano-secret" {
			http.Error(w, "invalid client credentials", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "nano-access-token",
			"token_type":   "Bearer",
			"expires_in":   300,
			"scope":        "financing.read payment.write",
			"active":       true,
		})
	}))
}

func newFinancingAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer nano-access-token" {
			http.Error(w, "missing or invalid token", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/financings/fin-123" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Financing{
			FinancingID: "fin-123",
			BorrowerID:  "worker-123",
			Status:      "ACTIVE",
			Product:     "student-financing",
			Installments: []Installment{
				{InstallmentID: "inst-003", DueDate: "2026-03-10", AmountDue: 60000, Status: "OPEN"},
				{InstallmentID: "inst-001", DueDate: "2026-01-10", AmountDue: 50000, Status: "OVERDUE"},
				{InstallmentID: "inst-004", DueDate: "2026-04-10", AmountDue: 30000, Status: "PAID"},
				{InstallmentID: "inst-002", DueDate: "2026-02-10", AmountDue: 40000, Status: "OPEN"},
			},
		})
	}))
}
