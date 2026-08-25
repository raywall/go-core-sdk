// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// handlers implements public handler errors.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package handlers

import "fmt"

// InvalidConfigError is returned when handler configuration is invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the configuration value is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid handler configuration"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid handler configuration: %s", e.Field)
	}
	return fmt.Sprintf("invalid handler configuration: %s: %s", e.Field, e.Reason)
}
