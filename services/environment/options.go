// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// environment implements construction options.
//
// This file is part of the Environment bounded context within the Environment
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package environment

import (
	"log/slog"
	"os"
)

// LookupFunc retrieves one environment variable value by name.
type LookupFunc func(name string) (string, bool)

// Option customizes Environment during construction.
type Option func(*options)

type options struct {
	logger *slog.Logger
	lookup LookupFunc
}

// WithLogger configures the structured logger used by Environment.
//
// The default logger writes JSON records to stdout. Passing nil keeps the
// default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) {
		if logger != nil {
			options.logger = logger
		}
	}
}

// WithLookupFunc configures the variable lookup function.
//
// This option is useful for tests, samples and applications that need to read
// from a virtual environment source. Passing nil keeps os.LookupEnv.
func WithLookupFunc(lookup LookupFunc) Option {
	return func(options *options) {
		if lookup != nil {
			options.lookup = lookup
		}
	}
}

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		lookup: os.LookupEnv,
	}
}
