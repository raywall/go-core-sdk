// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// config implements public configuration errors.
//
// This file is part of the Config bounded context within the Config service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package config

import "fmt"

// InvalidConfigError is returned when shared runtime configuration is invalid.
type InvalidConfigError struct {
	// Field identifies the invalid configuration field.
	Field string
	// Reason explains why the field is invalid.
	Reason string
}

// Error implements the error interface.
func (e InvalidConfigError) Error() string {
	if e.Field == "" {
		return "invalid config"
	}
	if e.Reason == "" {
		return fmt.Sprintf("invalid config: %s", e.Field)
	}
	return fmt.Sprintf("invalid config: %s: %s", e.Field, e.Reason)
}
