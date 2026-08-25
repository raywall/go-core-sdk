// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements distributed configuration consumer error types.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast

import (
	"fmt"

	"github.com/raywall/go-core-sdk/services/consumer/internal/consumererrors"
)

// InvalidConfigError is returned when Hazelcast consumer configuration is invalid.
type InvalidConfigError = consumererrors.InvalidConfigError

// DecodeError is returned when Hazelcast configuration or values cannot be decoded.
type DecodeError = consumererrors.DecodeError

// SourceError is returned when a Hazelcast configuration source cannot be read.
type SourceError struct {
	// Kind identifies the source backend.
	Kind SourceKind
	// Operation identifies the failed source operation.
	Operation string
	// Err is the wrapped source error.
	Err error
}

// Error implements the error interface.
func (e SourceError) Error() string {
	if e.Operation == "" {
		return "hazelcast config source failed"
	}
	if e.Err == nil {
		return fmt.Sprintf("hazelcast config source failed: %s: %s", e.Kind, e.Operation)
	}
	return fmt.Sprintf("hazelcast config source failed: %s: %s: %v", e.Kind, e.Operation, e.Err)
}

// Unwrap returns the wrapped source error.
func (e SourceError) Unwrap() error {
	return e.Err
}

// HazelcastError is returned when a Hazelcast client or map operation fails.
type HazelcastError struct {
	// Operation identifies the failed Hazelcast operation.
	Operation string
	// MapName identifies the map used by the operation when available.
	MapName string
	// Err is the wrapped Hazelcast error.
	Err error
}

// Error implements the error interface.
func (e HazelcastError) Error() string {
	if e.Operation == "" {
		return "hazelcast operation failed"
	}
	if e.Err == nil {
		return "hazelcast operation failed: " + e.Operation
	}
	if e.MapName == "" {
		return fmt.Sprintf("hazelcast operation failed: %s: %v", e.Operation, e.Err)
	}
	return fmt.Sprintf("hazelcast operation failed: %s: map %q: %v", e.Operation, e.MapName, e.Err)
}

// Unwrap returns the wrapped Hazelcast error.
func (e HazelcastError) Unwrap() error {
	return e.Err
}

// ValueConversionError is returned when a Hazelcast value cannot be converted.
type ValueConversionError struct {
	// MapName identifies the map used by the read operation.
	MapName string
	// Key identifies the requested key.
	Key any
	// Expected describes the expected target type.
	Expected string
	// Value is the raw value returned by Hazelcast.
	Value any
}

// Error implements the error interface.
func (e ValueConversionError) Error() string {
	return fmt.Sprintf("hazelcast value conversion failed: map %q key %v expected %s got %T", e.MapName, e.Key, e.Expected, e.Value)
}
