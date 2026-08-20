// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// core tests runtime service composition.
//
// This file is part of the Core bounded context within the Core service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package core_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/core"
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/observability"
)

func TestNew_ResolvesTokenCredentialsFromSecretsManagerAndStartsToken(t *testing.T) {
	t.Parallel()

	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.PostForm.Get("client_id"); got != "client-from-secret" {
			t.Fatalf("client_id = %q, want client-from-secret", got)
		}
		if got := r.PostForm.Get("client_secret"); got != "secret-from-secret" {
			t.Fatalf("client_secret = %q, want secret-from-secret", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "runtime-token",
			"token_type":    "Bearer",
			"expires_in":    300,
			"refresh_token": "refresh-runtime-token",
			"scope":         "orders.read",
			"active":        true,
		})
	}))
	t.Cleanup(sts.Close)

	cfg, err := config.Load(context.Background(),
		config.WithLogger(discardLogger()),
		config.WithToken("partner-api", config.TokenConfig{
			BaseURL:        sts.URL,
			Endpoint:       "/oauth/token",
			ValidateSSL:    true,
			SecretID:       "orders/partner-api",
			RequestTimeout: time.Second,
		}),
	)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	runtime, err := core.New(context.Background(), cfg,
		core.WithConsumerOptions(consumer.WithSecretsManagerClient(fakeSecretsManagerClient{})),
		core.WithObservabilityOptions(observability.WithMetricsClient(fakeMetricsClient{}), observability.WithLogger(discardLogger())),
		core.WithTokenAutoStart(true),
	)
	if err != nil {
		t.Fatalf("core.New: %v", err)
	}
	t.Cleanup(runtime.Stop)

	manager, ok := runtime.TokenManager("partner-api")
	if !ok {
		t.Fatal("TokenManager(partner-api) not found")
	}
	if got := manager.Token().ToString(); got != "Bearer runtime-token" {
		t.Fatalf("token = %q, want Bearer runtime-token", got)
	}
	if runtime.Consumer() == nil || runtime.Validator() == nil || runtime.Decision() == nil || runtime.Selector() == nil {
		t.Fatal("runtime services must be initialized")
	}
	if runtime.Observability() == nil {
		t.Fatal("runtime observability must be initialized")
	}
}

type fakeSecretsManagerClient struct{}

func (fakeSecretsManagerClient) GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		Name:         aws.String("orders/partner-api"),
		VersionId:    aws.String("version-1"),
		SecretString: aws.String(`{"client_id":"client-from-secret","client_secret":"secret-from-secret"}`),
	}, nil
}

func (fakeSecretsManagerClient) PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (fakeSecretsManagerClient) UpdateSecret(context.Context, *secretsmanager.UpdateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return &secretsmanager.UpdateSecretOutput{}, nil
}

type fakeMetricsClient struct{}

func (fakeMetricsClient) Count(string, int64, []string, float64) error {
	return nil
}

func (fakeMetricsClient) Incr(string, []string, float64) error {
	return nil
}

func (fakeMetricsClient) Gauge(string, float64, []string, float64) error {
	return nil
}

func (fakeMetricsClient) Histogram(string, float64, []string, float64) error {
	return nil
}

func (fakeMetricsClient) Distribution(string, float64, []string, float64) error {
	return nil
}

func (fakeMetricsClient) Timing(string, time.Duration, []string, float64) error {
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
