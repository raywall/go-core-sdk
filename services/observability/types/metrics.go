// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// observability/types defines metrics client contracts.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "time"

// MetricsClient sends custom metrics to an external telemetry backend.
type MetricsClient interface {
	// Count sends a count metric.
	Count(name string, value int64, tags []string, rate float64) error
	// Incr increments a counter metric.
	Incr(name string, tags []string, rate float64) error
	// Gauge sends a gauge metric.
	Gauge(name string, value float64, tags []string, rate float64) error
	// Histogram sends a histogram metric.
	Histogram(name string, value float64, tags []string, rate float64) error
	// Distribution sends a distribution metric.
	Distribution(name string, value float64, tags []string, rate float64) error
	// Timing sends a timing metric.
	Timing(name string, value time.Duration, tags []string, rate float64) error
}
