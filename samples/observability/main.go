// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/observability demonstrates custom metrics and structured logs.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/raywall/go-core-sdk/services/observability"
)

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		panic(err)
	}
}

// ObservabilityUseCase emits structured logs and custom metrics.
type ObservabilityUseCase struct {
	Output io.Writer
}

func run(ctx context.Context, out io.Writer) error {
	return ObservabilityUseCase{Output: out}.Execute(ctx)
}

func (u ObservabilityUseCase) Execute(ctx context.Context) error {
	telemetry, err := observability.New(observability.Config{
		ServiceName:  "orders-worker",
		Environment:  "dev",
		Version:      "1.0.0",
		MetricPrefix: "orders",
		DefaultTags:  []string{"team:platform", "component:sample"},
	}, observability.WithMetricsClient(writerMetricsClient{out: u.Output}), observability.WithWriter(u.Output))
	if err != nil {
		return err
	}
	defer func() {
		_ = telemetry.Close()
	}()

	telemetry.Logger().InfoContext(ctx, "order_processing_started", "order_id", "order-123")
	if err := telemetry.Increment(ctx, "events.received", "source:s3"); err != nil {
		return err
	}
	if err := telemetry.Gauge(ctx, "queue.depth", 42, "queue:orders"); err != nil {
		return err
	}
	if err := telemetry.Timing(ctx, "api.latency", 125*time.Millisecond, "partner:billing"); err != nil {
		return err
	}
	telemetry.Logger().InfoContext(ctx, "order_processing_finished", "order_id", "order-123")
	return nil
}

type writerMetricsClient struct {
	out io.Writer
}

func (c writerMetricsClient) Count(name string, value int64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=count name=%s value=%d tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c writerMetricsClient) Incr(name string, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=increment name=%s tags=%v rate=%.1f\n", name, tags, rate)
	return err
}

func (c writerMetricsClient) Gauge(name string, value float64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=gauge name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c writerMetricsClient) Histogram(name string, value float64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=histogram name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c writerMetricsClient) Distribution(name string, value float64, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=distribution name=%s value=%.2f tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}

func (c writerMetricsClient) Timing(name string, value time.Duration, tags []string, rate float64) error {
	_, err := fmt.Fprintf(c.out, "metric=timing name=%s value=%s tags=%v rate=%.1f\n", name, value, tags, rate)
	return err
}
