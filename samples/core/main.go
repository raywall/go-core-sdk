package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/core"
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/observability"
)

func main() {
	ctx := context.Background()
	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "sample-core-token",
			"token_type":    "Bearer",
			"expires_in":    300,
			"refresh_token": "refresh-sample-core-token",
			"scope":         "orders.read orders.write",
			"active":        true,
		})
	}))
	defer sts.Close()

	cfg, err := config.Load(ctx,
		config.WithServiceName("orders-file-worker"),
		config.WithEnvironment("local"),
		config.WithVersion("1.0.0"),
		config.WithAWSRegion("us-east-1"),
		config.WithObservability(config.ObservabilityConfig{
			MetricPrefix: "orders",
			DefaultTags:  []string{"team:platform"},
		}),
		config.WithToken("partner-api", config.TokenConfig{
			BaseURL:        sts.URL,
			Endpoint:       "/oauth/token",
			ValidateSSL:    true,
			SecretID:       "orders/partner-api",
			RequestTimeout: 5 * time.Second,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	runtime, err := core.New(ctx, cfg,
		core.WithConsumerOptions(consumer.WithSecretsManagerClient(fakeSecretsManagerClient{})),
		core.WithObservabilityOptions(observability.WithMetricsClient(stdoutMetricsClient{})),
		core.WithTokenAutoStart(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Stop()

	manager, _ := runtime.TokenManager("partner-api")
	if err := runtime.Observability().Increment(ctx, "core.started", "sample:true"); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("service=%s env=%s token=%q validatorReady=%t decisionReady=%t\n",
		runtime.Config().ServiceName(),
		runtime.Config().Environment(),
		manager.Token().ToString(),
		runtime.Validator() != nil,
		runtime.Decision() != nil,
	)
}

type fakeSecretsManagerClient struct{}

func (fakeSecretsManagerClient) GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &secretsmanager.GetSecretValueOutput{
		Name:         aws.String("orders/partner-api"),
		VersionId:    aws.String("version-1"),
		SecretString: aws.String(`{"client_id":"sample-client","client_secret":"sample-secret"}`),
	}, nil
}

func (fakeSecretsManagerClient) PutSecretValue(context.Context, *secretsmanager.PutSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	return &secretsmanager.PutSecretValueOutput{}, nil
}

func (fakeSecretsManagerClient) UpdateSecret(context.Context, *secretsmanager.UpdateSecretInput, ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return &secretsmanager.UpdateSecretOutput{}, nil
}

type stdoutMetricsClient struct{}

func (stdoutMetricsClient) Count(name string, value int64, tags []string, rate float64) error {
	fmt.Printf("metric=count name=%s value=%d tags=%v rate=%.1f\n", name, value, tags, rate)
	return nil
}

func (stdoutMetricsClient) Incr(name string, tags []string, rate float64) error {
	fmt.Printf("metric=increment name=%s tags=%v rate=%.1f\n", name, tags, rate)
	return nil
}

func (stdoutMetricsClient) Gauge(name string, value float64, tags []string, rate float64) error {
	fmt.Printf("metric=gauge name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return nil
}

func (stdoutMetricsClient) Histogram(name string, value float64, tags []string, rate float64) error {
	fmt.Printf("metric=histogram name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return nil
}

func (stdoutMetricsClient) Distribution(name string, value float64, tags []string, rate float64) error {
	fmt.Printf("metric=distribution name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return nil
}

func (stdoutMetricsClient) Timing(name string, value time.Duration, tags []string, rate float64) error {
	fmt.Printf("metric=timing name=%s value=%s tags=%v rate=%.1f\n", name, value, tags, rate)
	return nil
}
