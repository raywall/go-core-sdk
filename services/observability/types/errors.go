// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// observability/types defines public observability errors.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "fmt"

// InvalidConfigError is returned when observability configuration is invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid observability config"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid observability config: %s", e.Field)
	}
	return fmt.Sprintf("invalid observability config: %s: %s", e.Field, e.Reason)
}

// InvalidMetricError is returned when a metric request is invalid.
type InvalidMetricError struct {
	// Field identifies the invalid metric field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidMetricError) Error() string {
	if e.Field == "" {
		return "invalid metric"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid metric: %s", e.Field)
	}
	return fmt.Sprintf("invalid metric: %s: %s", e.Field, e.Reason)
}

// MetricError wraps a metric backend failure.
type MetricError struct {
	// Name is the normalized metric name sent to the backend.
	Name string
	// Operation identifies the metric operation.
	Operation string
	// Err is the wrapped backend error.
	Err error
}

// Error implements the error interface.
func (e MetricError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("metric %s failed for %s", e.Operation, e.Name)
	}
	return fmt.Sprintf("metric %s failed for %s: %v", e.Operation, e.Name, e.Err)
}

// Unwrap returns the wrapped backend error.
func (e MetricError) Unwrap() error {
	return e.Err
}
