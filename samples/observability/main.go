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
	"os"
	"time"

	"github.com/raywall/go-core-sdk/services/observability"
)

func main() {
	ctx := context.Background()
	telemetry, err := observability.New(observability.Config{
		ServiceName:  "orders-worker",
		Environment:  "dev",
		Version:      "1.0.0",
		MetricPrefix: "orders",
		DefaultTags:  []string{"team:platform", "component:sample"},
	}, observability.WithMetricsClient(stdoutMetricsClient{}), observability.WithWriter(os.Stdout))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := telemetry.Close(); err != nil {
			panic(err)
		}
	}()

	telemetry.Logger().InfoContext(ctx, "order_processing_started", "order_id", "order-123")
	if err := telemetry.Increment(ctx, "events.received", "source:s3"); err != nil {
		panic(err)
	}
	if err := telemetry.Gauge(ctx, "queue.depth", 42, "queue:orders"); err != nil {
		panic(err)
	}
	if err := telemetry.Timing(ctx, "api.latency", 125*time.Millisecond, "partner:billing"); err != nil {
		panic(err)
	}
	telemetry.Logger().InfoContext(ctx, "order_processing_finished", "order_id", "order-123")
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
