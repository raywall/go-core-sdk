// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/microservice implements local fakes for external integrations.
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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
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

type fakeS3Client struct {
	objects map[string]string
}

func newFakeS3Client(bucket string, key string, body string) *fakeS3Client {
	return &fakeS3Client{
		objects: map[string]string{
			objectID(bucket, key): body,
		},
	}
}

func (f *fakeS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	f.objects[objectID(aws.ToString(input.Bucket), aws.ToString(input.Key))] = string(body)
	return &s3.PutObjectOutput{ETag: aws.String("sample-etag")}, nil
}

func (f *fakeS3Client) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	body, ok := f.objects[objectID(aws.ToString(input.Bucket), aws.ToString(input.Key))]
	if !ok {
		return nil, fmt.Errorf("object not found: %s/%s", aws.ToString(input.Bucket), aws.ToString(input.Key))
	}
	return &s3.GetObjectOutput{
		Body:        io.NopCloser(strings.NewReader(body)),
		ContentType: aws.String("application/json"),
	}, nil
}

type fakeSQSClient struct {
	messages []sentMessage
}

type sentMessage struct {
	body       string
	attributes map[string]string
}

func (f *fakeSQSClient) SendMessage(_ context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	message := sentMessage{
		body:       aws.ToString(input.MessageBody),
		attributes: make(map[string]string, len(input.MessageAttributes)),
	}
	for key, value := range input.MessageAttributes {
		message.attributes[key] = aws.ToString(value.StringValue)
	}
	f.messages = append(f.messages, message)
	return &sqs.SendMessageOutput{MessageId: aws.String(fmt.Sprintf("sample-message-%d", len(f.messages)))}, nil
}

func (f *fakeSQSClient) ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	messages := make([]sqstypes.Message, 0, len(f.messages))
	for index, message := range f.messages {
		messages = append(messages, sqstypes.Message{
			MessageId:     aws.String(fmt.Sprintf("sample-message-%d", index+1)),
			ReceiptHandle: aws.String(fmt.Sprintf("sample-receipt-%d", index+1)),
			Body:          aws.String(message.body),
		})
	}
	return &sqs.ReceiveMessageOutput{Messages: messages}, nil
}

func (fakeSQSClient) DeleteMessage(context.Context, *sqs.DeleteMessageInput, ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
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
		if r.PostForm.Get("client_id") != "student-financing-client" || r.PostForm.Get("client_secret") != "student-financing-secret" {
			http.Error(w, "invalid client credentials", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "sample-access-token",
			"token_type":    "Bearer",
			"expires_in":    300,
			"refresh_token": "sample-refresh-token",
			"scope":         "student-financing.read payments.write",
			"active":        true,
		})
	}))
}

func newStudentFinancingAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sample-access-token" {
			http.Error(w, "missing or invalid token", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/financings/fin-123" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(financingResponse{
			FinancingID: "fin-123",
			BorrowerID:  "worker-123",
			Product:     "student-financing",
			Installments: []installment{
				{InstallmentID: "inst-005", DueDate: "2026-05-10", AmountDue: 45000, Status: "OPEN"},
				{InstallmentID: "inst-001", DueDate: "2026-01-10", AmountDue: 50000, Status: "OVERDUE"},
				{InstallmentID: "inst-003", DueDate: "2026-03-10", AmountDue: 45000, Status: "PAID"},
				{InstallmentID: "inst-004", DueDate: "2026-04-10", AmountDue: 45000, Status: "OPEN"},
				{InstallmentID: "inst-002", DueDate: "2026-02-10", AmountDue: 50000, Status: "OVERDUE"},
			},
		})
	}))
}

func objectID(bucket string, key string) string {
	return strings.TrimSpace(bucket) + "/" + strings.TrimSpace(key)
}
