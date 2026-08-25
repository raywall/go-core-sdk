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
	consumeraws "github.com/raywall/go-core-sdk/services/consumer/aws"
	consumerrest "github.com/raywall/go-core-sdk/services/consumer/rest"
	"github.com/raywall/go-core-sdk/services/observability"
)

// Option customizes Core during construction.
type Option func(*options)

type options struct {
	awsOptions           []consumeraws.Option
	observabilityOptions []observability.Option
	restOptions          []consumerrest.Option
	tokenAutoStart       bool
}

// WithAWSOptions appends options used when Core builds the AWS consumer service.
func WithAWSOptions(configurers ...consumeraws.Option) Option {
	return func(options *options) {
		options.awsOptions = append(options.awsOptions, configurers...)
	}
}

// WithRESTOptions appends options used when Core builds the REST consumer service.
func WithRESTOptions(configurers ...consumerrest.Option) Option {
	return func(options *options) {
		options.restOptions = append(options.restOptions, configurers...)
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
