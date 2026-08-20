// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// observability implements structured logger creation.
//
// This file is part of the Observability bounded context within the
// Observability service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package observability

import "log/slog"

func newLogger(config Config, options options) *slog.Logger {
	if options.logger != nil {
		return options.logger
	}

	handler := slog.NewJSONHandler(options.writer, &slog.HandlerOptions{Level: config.LogLevel})
	attrs := []any{}
	if config.ServiceName != "" {
		attrs = append(attrs, "service", config.ServiceName)
	}
	if config.Environment != "" {
		attrs = append(attrs, "environment", config.Environment)
	}
	if config.Version != "" {
		attrs = append(attrs, "version", config.Version)
	}
	return slog.New(handler).With(attrs...)
}
