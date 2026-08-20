// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public cache service errors.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import (
	"errors"
	"fmt"
)

var (
	// ErrCacheAlreadyStarted is returned when a cache lifecycle operation
	// requires a stopped cache but the cache is already running.
	ErrCacheAlreadyStarted = errors.New("cache already started")

	// ErrCacheStopped is returned when a cache lifecycle operation requires a
	// running cache but the cache is stopped.
	ErrCacheStopped = errors.New("cache stopped")

	// ErrItemNotFound is returned when a cache item cannot be found.
	ErrItemNotFound = errors.New("cache item not found")
)

// InvalidConfigError is returned when cache configuration is missing or
// invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the configuration value is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid cache configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid cache configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid cache configuration: %s: %s", e.Field, e.Reason)
}

// InvalidKeyError is returned when a cache key is empty or malformed.
type InvalidKeyError struct {
	// Key is the invalid key supplied by the caller.
	Key string
	// Reason explains why the key is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidKeyError) Error() string {
	if e.Reason == "" {
		return "invalid cache key"
	}
	return "invalid cache key: " + e.Reason
}

// InvalidTTLError is returned when a cache TTL is invalid.
type InvalidTTLError struct {
	// TTL is the invalid TTL supplied by the caller.
	TTL string
	// Reason explains why the TTL is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidTTLError) Error() string {
	if e.Reason == "" {
		return "invalid cache ttl"
	}
	return "invalid cache ttl: " + e.Reason
}
