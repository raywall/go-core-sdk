// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/services/observability/types"
)

func TestIncrementAddsPrefixAndTags(t *testing.T) {
	t.Parallel()

	client := &fakeMetricsClient{}
	service, err := New(Config{
		Environment:  "prod",
		MetricPrefix: "payments",
		DefaultTags:  []string{"team:core", "env:old", "team:core"},
	}, WithMetricsClient(client))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := service.Increment(context.Background(), "events.processed", "source:s3", "env:ignored"); err != nil {
		t.Fatalf("Increment() error = %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("calls length = %d, want 1", len(client.calls))
	}
	call := client.calls[0]
	if call.operation != "increment" {
		t.Fatalf("operation = %q, want increment", call.operation)
	}
	if call.name != "payments.events.processed" {
		t.Fatalf("name = %q, want payments.events.processed", call.name)
	}
	assertTags(t, call.tags, []string{"team:core", "source:s3", "env:prod"})
}

func TestMetricErrorWrapsBackendFailure(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("backend unavailable")
	client := &fakeMetricsClient{err: backendErr}
	service, err := New(Config{Environment: "dev"}, WithMetricsClient(client))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = service.Count(context.Background(), "processed", 3)
	if err == nil {
		t.Fatal("Count() error = nil, want error")
	}
	var metricErr types.MetricError
	if !errors.As(err, &metricErr) {
		t.Fatalf("Count() error type = %T, want MetricError", err)
	}
	if !errors.Is(err, backendErr) {
		t.Fatalf("Count() error does not wrap backend error")
	}
	if metricErr.Name != "processed" || metricErr.Operation != "count" {
		t.Fatalf("MetricError = %+v", metricErr)
	}
}

func TestMetricNameIsRequired(t *testing.T) {
	t.Parallel()

	service, err := New(Config{Environment: "dev"}, WithMetricsClient(&fakeMetricsClient{}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = service.Gauge(context.Background(), " . ", 1)
	if err == nil {
		t.Fatal("Gauge() error = nil, want error")
	}
	var invalidMetric types.InvalidMetricError
	if !errors.As(err, &invalidMetric) {
		t.Fatalf("Gauge() error type = %T, want InvalidMetricError", err)
	}
}

func TestLoggerWritesJSONWithBaseFields(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	service, err := New(Config{
		ServiceName: "orders-api",
		Environment: "staging",
		Version:     "1.2.3",
		LogLevel:    slog.LevelDebug,
	}, WithMetricsClient(&fakeMetricsClient{}), WithWriter(&buffer))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	service.Logger().InfoContext(context.Background(), "order_processed", "order_id", "123")

	var record map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &record); err != nil {
		t.Fatalf("log JSON unmarshal error = %v; output = %s", err, buffer.String())
	}
	assertJSONValue(t, record, "msg", "order_processed")
	assertJSONValue(t, record, "service", "orders-api")
	assertJSONValue(t, record, "environment", "staging")
	assertJSONValue(t, record, "version", "1.2.3")
	assertJSONValue(t, record, "order_id", "123")
}

func TestTimingSendsDuration(t *testing.T) {
	t.Parallel()

	client := &fakeMetricsClient{}
	service, err := New(Config{Environment: "dev"}, WithMetricsClient(client))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	duration := 150 * time.Millisecond
	if err := service.Timing(context.Background(), "latency", duration); err != nil {
		t.Fatalf("Timing() error = %v", err)
	}

	if len(client.calls) != 1 {
		t.Fatalf("calls length = %d, want 1", len(client.calls))
	}
	call := client.calls[0]
	if call.operation != "timing" {
		t.Fatalf("operation = %q, want timing", call.operation)
	}
	if call.duration != duration {
		t.Fatalf("duration = %v, want %v", call.duration, duration)
	}
}

func assertTags(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}
	for index, tag := range want {
		if got[index] != tag {
			t.Fatalf("tags = %v, want %v", got, want)
		}
	}
}

func assertJSONValue(t *testing.T, record map[string]any, key string, want string) {
	t.Helper()
	got, ok := record[key].(string)
	if !ok {
		t.Fatalf("record[%q] = %#v, want string", key, record[key])
	}
	if got != want {
		t.Fatalf("record[%q] = %q, want %q", key, got, want)
	}
}

type fakeMetricsClient struct {
	err   error
	calls []metricCall
}

type metricCall struct {
	operation string
	name      string
	intValue  int64
	value     float64
	duration  time.Duration
	tags      []string
	rate      float64
}

func (f *fakeMetricsClient) Count(name string, value int64, tags []string, rate float64) error {
	return f.record(metricCall{operation: "count", name: name, intValue: value, tags: tags, rate: rate})
}

func (f *fakeMetricsClient) Incr(name string, tags []string, rate float64) error {
	return f.record(metricCall{operation: "increment", name: name, tags: tags, rate: rate})
}

func (f *fakeMetricsClient) Gauge(name string, value float64, tags []string, rate float64) error {
	return f.record(metricCall{operation: "gauge", name: name, value: value, tags: tags, rate: rate})
}

func (f *fakeMetricsClient) Histogram(name string, value float64, tags []string, rate float64) error {
	return f.record(metricCall{operation: "histogram", name: name, value: value, tags: tags, rate: rate})
}

func (f *fakeMetricsClient) Distribution(name string, value float64, tags []string, rate float64) error {
	return f.record(metricCall{operation: "distribution", name: name, value: value, tags: tags, rate: rate})
}

func (f *fakeMetricsClient) Timing(name string, value time.Duration, tags []string, rate float64) error {
	return f.record(metricCall{operation: "timing", name: name, duration: value, tags: tags, rate: rate})
}

func (f *fakeMetricsClient) record(call metricCall) error {
	if f.err != nil {
		return f.err
	}
	call.tags = append([]string(nil), call.tags...)
	f.calls = append(f.calls, call)
	return nil
}
