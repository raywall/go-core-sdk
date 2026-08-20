// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// environment implements environment variable access.
//
// This file is part of the Environment bounded context within the Environment
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package environment

import (
	"context"
	"log/slog"
	"strings"

	"github.com/raywall/go-core-sdk/services/environment/types"
)

// Environment reads environment variables using required and defaulted access
// patterns.
//
// Environment is safe for concurrent use when the configured LookupFunc is safe
// for concurrent use. The default os.LookupEnv implementation is safe.
type Environment struct {
	logger *slog.Logger
	lookup LookupFunc
}

// New constructs an Environment service.
func New(configurers ...Option) (*Environment, error) {
	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}
	return &Environment{
		logger: options.logger,
		lookup: options.lookup,
	}, nil
}

// Get returns a required environment variable value.
//
// Get preserves empty string values when the variable exists. It returns
// VariableNotFoundError only when the variable is absent.
func (e *Environment) Get(ctx context.Context, name string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, err := normalizeName(name)
	if err != nil {
		return "", err
	}

	value, found := e.lookup(name)
	if !found {
		e.logger.WarnContext(ctx, "environment_variable_not_found", "name", name)
		return "", types.VariableNotFoundError{Name: name}
	}
	e.logger.DebugContext(ctx, "environment_variable_loaded", "name", name, "defaulted", false)
	return value, nil
}

// GetDefault returns an environment variable value or defaultValue when absent.
//
// GetDefault preserves empty string values when the variable exists. The default
// value is used only when the variable is absent from the configured lookup
// source.
func (e *Environment) GetDefault(ctx context.Context, name string, defaultValue string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name, err := normalizeName(name)
	if err != nil {
		return "", err
	}

	value, found := e.lookup(name)
	if !found {
		e.logger.DebugContext(ctx, "environment_variable_defaulted", "name", name)
		return defaultValue, nil
	}
	e.logger.DebugContext(ctx, "environment_variable_loaded", "name", name, "defaulted", false)
	return value, nil
}

// Get returns a required environment variable value using a short-lived
// Environment service.
func Get(ctx context.Context, name string, configurers ...Option) (string, error) {
	service, err := New(configurers...)
	if err != nil {
		return "", err
	}
	return service.Get(ctx, name)
}

// GetDefault returns an environment variable value or defaultValue using a
// short-lived Environment service.
func GetDefault(ctx context.Context, name string, defaultValue string, configurers ...Option) (string, error) {
	service, err := New(configurers...)
	if err != nil {
		return "", err
	}
	return service.GetDefault(ctx, name, defaultValue)
}

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", types.InvalidInputError{Field: "Name", Reason: "is required"}
	}
	return name, nil
}
