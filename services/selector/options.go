// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// selector implements construction options for the selector service.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package selector

import (
	"log/slog"
	"os"
)

// Option customizes a Selector during construction.
type Option func(*options)

type options struct {
	logger *slog.Logger
}

// WithLogger configures the structured logger used by Selector.
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
	return options{logger: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
}
