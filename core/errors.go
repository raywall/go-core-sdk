// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// core implements public runtime composition errors.
//
// This file is part of the Core bounded context within the Core service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package core

import "fmt"

// InvalidConfigError is returned when Core cannot be built from the supplied configuration.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid core configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid core configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid core configuration: %s: %s", e.Field, e.Reason)
}

// TokenNotFoundError is returned when a named token manager is not configured.
type TokenNotFoundError struct {
	// Name is the requested token manager name.
	Name string
}

// Error implements the error interface.
func (e TokenNotFoundError) Error() string {
	if e.Name == "" {
		return "token manager not found"
	}
	return "token manager not found: " + e.Name
}
