package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/core"
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/observability"
)

func TestRunComposesCoreRuntime(t *testing.T) {
	t.Parallel()

	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "sample-core-token",
			"token_type":   "Bearer",
			"expires_in":   300,
			"active":       true,
		})
	}))
	defer sts.Close()

	ctx := context.Background()
	cfg, err := config.Load(ctx,
		config.WithServiceName("orders-file-worker"),
		config.WithEnvironment("local"),
		config.WithVersion("1.0.0"),
		config.WithAWSRegion("us-east-1"),
		config.WithToken("partner-api", config.TokenConfig{
			BaseURL:        sts.URL,
			Endpoint:       "/oauth/token",
			ValidateSSL:    true,
			SecretID:       "orders/partner-api",
			RequestTimeout: time.Second,
		}),
	)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	var out bytes.Buffer
	runtime, err := core.New(ctx, cfg,
		core.WithConsumerOptions(consumer.WithSecretsManagerClient(fakeSecretsManagerClient{})),
		core.WithObservabilityOptions(observability.WithMetricsClient(stdoutMetricsClient{out: &out})),
		core.WithTokenAutoStart(true),
	)
	if err != nil {
		t.Fatalf("core.New() error = %v", err)
	}

	if err := run(ctx, CoreUseCase{Runtime: runtime, Output: &out}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"metric=increment name=core.started", `token="Bearer sample-core-token"`, "validatorReady=true", "decisionReady=true"} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
