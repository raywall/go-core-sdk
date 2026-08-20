// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// config implements shared runtime configuration loading and projection.
//
// This file is part of the Config bounded context within the Config service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package config centralizes application configuration for go-core-sdk services.
//
// The package follows the same broad shape as the AWS SDK configuration loader:
// callers provide options, optional loaders collect external configuration, and
// resolvers normalize the final Config. The resulting value can be projected
// into service-specific configs without making those services depend on this
// package.
//
// Usage:
//
//	cfg, err := config.Load(ctx,
//		config.WithEnv("APP"),
//		config.WithServiceName("orders-worker"),
//		config.WithAWSRegion("us-east-1"),
//		config.WithToken("partner-api", config.TokenConfig{
//			BaseURL:  "https://sts.example.com",
//			Endpoint: "/oauth/token",
//			SecretID: "orders/partner-api",
//		}),
//	)
//	if err != nil {
//		return err
//	}
//
//	consumerConfig := cfg.Consumer()
//
// Thread safety: Config values returned by Load are immutable by convention.
// Methods that expose maps copy them before returning service-specific values.
package config
