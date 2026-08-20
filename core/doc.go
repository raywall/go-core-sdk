// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// core implements application runtime composition for go-core-sdk services.
//
// This file is part of the Core bounded context within the Core service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package core wires go-core-sdk services into an application runtime.
//
// Core is intentionally a composition layer. It does not contain business
// rules, and no service depends on it. Applications can use Core when they want
// one place to initialize shared logger, AWS-backed consumer, validation,
// decision, selector and token managers.
//
// Usage:
//
//	cfg, err := config.Load(ctx,
//		config.WithEnv("APP"),
//		config.WithAWSDefaultConfig(),
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
//	runtime, err := core.New(ctx, cfg, core.WithTokenAutoStart(true))
//	if err != nil {
//		return err
//	}
//	defer runtime.Stop()
package core
