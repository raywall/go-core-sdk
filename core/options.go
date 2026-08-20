// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// core implements construction options.
//
// This file is part of the Core bounded context within the Core service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package core

import (
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/observability"
)

// Option customizes Core during construction.
type Option func(*options)

type options struct {
	consumerOptions      []consumer.Option
	observabilityOptions []observability.Option
	tokenAutoStart       bool
}

// WithConsumerOptions appends options used when Core builds the consumer service.
func WithConsumerOptions(configurers ...consumer.Option) Option {
	return func(options *options) {
		options.consumerOptions = append(options.consumerOptions, configurers...)
	}
}

// WithObservabilityOptions appends options used when Core builds observability.
func WithObservabilityOptions(configurers ...observability.Option) Option {
	return func(options *options) {
		options.observabilityOptions = append(options.observabilityOptions, configurers...)
	}
}

// WithTokenAutoStart controls whether New starts configured token managers.
func WithTokenAutoStart(enabled bool) Option {
	return func(options *options) {
		options.tokenAutoStart = enabled
	}
}
