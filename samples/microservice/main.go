// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/microservice implements an executable end-to-end SDK composition sample.
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
	"io"
	"log"
	"os"
	"time"

	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/core"
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/environment"
	"github.com/raywall/go-core-sdk/services/observability"
)

const (
	defaultTokenManagerID  = "student-financing-api"
	defaultSourceBucket    = "student-financing-inbox"
	defaultSourceObjectKey = "payments/instruction-001.json"
	defaultPaymentQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/student-financing-payments"
)

func main() {
	if _, err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, out io.Writer) (paymentEvent, error) {
	sts := newSTS()
	defer sts.Close()

	financingAPI := newStudentFinancingAPI()
	defer financingAPI.Close()

	env, err := environment.New(environment.WithLookupFunc(sampleEnvironment()))
	if err != nil {
		return paymentEvent{}, err
	}
	settings, err := loadRuntimeSettings(ctx, env)
	if err != nil {
		return paymentEvent{}, err
	}

	s3Client := newFakeS3Client(settings.SourceBucket, settings.SourceObjectKey, sampleInstructionJSON())
	sqsClient := &fakeSQSClient{}

	cfg, err := config.Load(ctx,
		config.WithServiceName(settings.ServiceName),
		config.WithEnvironment(settings.Environment),
		config.WithVersion(settings.Version),
		config.WithAWSRegion(settings.AWSRegion),
		config.WithObservability(config.ObservabilityConfig{
			MetricPrefix: settings.MetricPrefix,
			DefaultTags:  []string{"team:education-credit", "runtime:sample"},
		}),
		config.WithToken(settings.TokenManagerID, config.TokenConfig{
			BaseURL:        sts.URL,
			Endpoint:       "/oauth/token",
			ValidateSSL:    true,
			SecretID:       settings.SecretID,
			RequestTimeout: 3 * time.Second,
			RefreshBefore:  30 * time.Second,
			MaxRetries:     1,
			RetryBackoff:   50 * time.Millisecond,
		}),
	)
	if err != nil {
		return paymentEvent{}, err
	}

	runtime, err := core.New(ctx, cfg,
		core.WithConsumerOptions(
			consumer.WithSecretsManagerClient(fakeSecretsManagerClient{
				secretID: settings.SecretID,
				value:    `{"client_id":"student-financing-client","client_secret":"student-financing-secret"}`,
			}),
			consumer.WithS3Client(s3Client),
			consumer.WithSQSClient(sqsClient),
		),
		core.WithObservabilityOptions(
			observability.WithMetricsClient(stdoutMetricsClient{out: out}),
			observability.WithWriter(out),
		),
		core.WithTokenAutoStart(true),
	)
	if err != nil {
		return paymentEvent{}, err
	}
	defer runtime.Stop()

	processor := paymentProcessor{
		runtime:        runtime,
		financingAPI:   financingAPI.URL,
		paymentQueue:   settings.PaymentQueueURL,
		tokenManagerID: settings.TokenManagerID,
	}

	if err := processor.processS3Notification(ctx, sampleS3Notification(settings)); err != nil {
		return paymentEvent{}, err
	}

	if len(sqsClient.messages) == 0 {
		return paymentEvent{}, nil
	}

	var event paymentEvent
	if err := json.Unmarshal([]byte(sqsClient.messages[0].body), &event); err != nil {
		return paymentEvent{}, err
	}
	runtime.Logger().InfoContext(ctx, "sample_sqs_payment_event_body", "body", sqsClient.messages[0].body)
	return event, nil
}

type runtimeSettings struct {
	ServiceName     string
	Environment     string
	Version         string
	AWSRegion       string
	MetricPrefix    string
	TokenManagerID  string
	SecretID        string
	SourceBucket    string
	SourceObjectKey string
	PaymentQueueURL string
}

func loadRuntimeSettings(ctx context.Context, env *environment.Environment) (runtimeSettings, error) {
	secretID, err := env.Get(ctx, "APP_STUDENT_FINANCING_SECRET_ID")
	if err != nil {
		return runtimeSettings{}, err
	}
	serviceName, err := env.GetDefault(ctx, "APP_SERVICE_NAME", "student-financing-payment-worker")
	if err != nil {
		return runtimeSettings{}, err
	}
	environmentName, err := env.GetDefault(ctx, "APP_ENVIRONMENT", "local")
	if err != nil {
		return runtimeSettings{}, err
	}
	version, err := env.GetDefault(ctx, "APP_VERSION", "1.0.0")
	if err != nil {
		return runtimeSettings{}, err
	}
	awsRegion, err := env.GetDefault(ctx, "APP_AWS_REGION", "us-east-1")
	if err != nil {
		return runtimeSettings{}, err
	}
	metricPrefix, err := env.GetDefault(ctx, "APP_OBSERVABILITY_PREFIX", "student_financing")
	if err != nil {
		return runtimeSettings{}, err
	}
	tokenManagerID, err := env.GetDefault(ctx, "APP_TOKEN_MANAGER_ID", defaultTokenManagerID)
	if err != nil {
		return runtimeSettings{}, err
	}
	sourceBucket, err := env.GetDefault(ctx, "APP_SOURCE_BUCKET", defaultSourceBucket)
	if err != nil {
		return runtimeSettings{}, err
	}
	sourceObjectKey, err := env.GetDefault(ctx, "APP_SOURCE_OBJECT_KEY", defaultSourceObjectKey)
	if err != nil {
		return runtimeSettings{}, err
	}
	paymentQueueURL, err := env.GetDefault(ctx, "APP_PAYMENT_QUEUE_URL", defaultPaymentQueueURL)
	if err != nil {
		return runtimeSettings{}, err
	}

	return runtimeSettings{
		ServiceName:     serviceName,
		Environment:     environmentName,
		Version:         version,
		AWSRegion:       awsRegion,
		MetricPrefix:    metricPrefix,
		TokenManagerID:  tokenManagerID,
		SecretID:        secretID,
		SourceBucket:    sourceBucket,
		SourceObjectKey: sourceObjectKey,
		PaymentQueueURL: paymentQueueURL,
	}, nil
}

func sampleEnvironment() environment.LookupFunc {
	values := map[string]string{
		"APP_SERVICE_NAME":                "student-financing-payment-worker",
		"APP_STUDENT_FINANCING_SECRET_ID": "student-financing/api-credentials",
	}
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func sampleS3Notification(settings runtimeSettings) s3Notification {
	return s3Notification{
		Records: []s3NotificationRecord{
			{
				EventName: "ObjectCreated:Put",
				S3: s3Entity{
					Bucket: s3Bucket{Name: settings.SourceBucket},
					Object: s3Object{Key: settings.SourceObjectKey},
				},
			},
		},
	}
}

func sampleInstructionJSON() string {
	return `{
		"instructionId": "instruction-001",
		"workerId": "worker-123",
		"financingId": "fin-123",
		"availableAmount": 120000,
		"requestedBy": "s3-notification-sample",
		"sourceSystem": "student-financing-file-gateway"
	}`
}
