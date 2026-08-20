// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// observability implements custom metric emission.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package observability

import (
	"context"
	"strings"
	"time"

	"github.com/raywall/go-core-sdk/services/observability/types"
)

// Count sends a count metric.
func (o *Observability) Count(ctx context.Context, name string, value int64, tags ...string) error {
	return o.send(ctx, "count", name, tags, func(metricName string, metricTags []string) error {
		return o.metrics.Count(metricName, value, metricTags, defaultMetricRate)
	})
}

// Increment increments a counter metric.
func (o *Observability) Increment(ctx context.Context, name string, tags ...string) error {
	return o.send(ctx, "increment", name, tags, func(metricName string, metricTags []string) error {
		return o.metrics.Incr(metricName, metricTags, defaultMetricRate)
	})
}

// Gauge sends a gauge metric.
func (o *Observability) Gauge(ctx context.Context, name string, value float64, tags ...string) error {
	return o.send(ctx, "gauge", name, tags, func(metricName string, metricTags []string) error {
		return o.metrics.Gauge(metricName, value, metricTags, defaultMetricRate)
	})
}

// Histogram sends a histogram metric.
func (o *Observability) Histogram(ctx context.Context, name string, value float64, tags ...string) error {
	return o.send(ctx, "histogram", name, tags, func(metricName string, metricTags []string) error {
		return o.metrics.Histogram(metricName, value, metricTags, defaultMetricRate)
	})
}

// Distribution sends a distribution metric.
func (o *Observability) Distribution(ctx context.Context, name string, value float64, tags ...string) error {
	return o.send(ctx, "distribution", name, tags, func(metricName string, metricTags []string) error {
		return o.metrics.Distribution(metricName, value, metricTags, defaultMetricRate)
	})
}

// Timing sends a timing metric.
func (o *Observability) Timing(ctx context.Context, name string, value time.Duration, tags ...string) error {
	return o.send(ctx, "timing", name, tags, func(metricName string, metricTags []string) error {
		return o.metrics.Timing(metricName, value, metricTags, defaultMetricRate)
	})
}

func (o *Observability) send(ctx context.Context, operation string, name string, tags []string, emit func(string, []string) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if o == nil || o.metrics == nil {
		return types.InvalidConfigError{Field: "MetricsClient", Reason: "is required"}
	}

	metricName, err := o.metricName(name)
	if err != nil {
		return err
	}
	metricTags := o.metricTags(tags)
	if err := emit(metricName, metricTags); err != nil {
		o.logger.ErrorContext(ctx, "metric_failed", "operation", operation, "metric", metricName, "error", err)
		return types.MetricError{Name: metricName, Operation: operation, Err: err}
	}
	o.logger.DebugContext(ctx, "metric_sent", "operation", operation, "metric", metricName, "tags", metricTags)
	return nil
}

func (o *Observability) metricName(name string) (string, error) {
	name = strings.Trim(strings.TrimSpace(name), ".")
	if name == "" {
		return "", types.InvalidMetricError{Field: "Name", Reason: "is required"}
	}
	if o.config.MetricPrefix == "" {
		return name, nil
	}
	return o.config.MetricPrefix + "." + name, nil
}

func (o *Observability) metricTags(tags []string) []string {
	all := make([]string, 0, len(o.config.DefaultTags)+len(tags)+1)
	all = append(all, o.config.DefaultTags...)
	all = append(all, tags...)

	deduped := make([]string, 0, len(all)+1)
	seen := make(map[string]struct{}, len(all)+1)
	for _, tag := range all {
		tag = strings.TrimSpace(tag)
		if tag == "" || strings.HasPrefix(tag, "env:") {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		deduped = append(deduped, tag)
	}

	envTag := "env:" + o.config.Environment
	if _, ok := seen[envTag]; !ok {
		deduped = append(deduped, envTag)
	}
	return deduped
}
