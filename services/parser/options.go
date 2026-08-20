// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// parser implements construction options.
//
// This file is part of the Parser bounded context within the Parser service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package parser

import (
	"log/slog"
	"os"
)

// Option customizes Parser during construction.
type Option func(*options)

type options struct {
	logger *slog.Logger
}

// WithLogger configures the structured logger used by Parser.
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

func defaultOptions() options {
	return options{
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}
