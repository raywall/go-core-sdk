// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// token implements lifecycle management for STS access tokens.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package token provides an auto-refreshing STS token manager for applications
// that authenticate with the client credentials grant.
//
// The package is intentionally self-contained: callers import only
// github.com/raywall/go-core-sdk/services/token to construct and run a manager.
// Public token data is exposed through the service types package, while HTTP
// transport details remain inside this service's internal package.
//
// Usage:
//
//	manager, err := token.NewManager(token.Config{
//		BaseURL:      "https://sts.example.com",
//		Endpoint:     "/oauth/token",
//		ClientID:     "uuid",
//		ClientSecret: "uuid",
//		ValidateSSL:  true,
//	})
//	if err != nil {
//		return err
//	}
//	if err := manager.Start(ctx); err != nil {
//		return err
//	}
//	defer manager.Stop()
//
//	authorization := manager.Token().ToString()
//
// Thread safety: Manager is safe for concurrent use. The token pointer returned
// by Token is stable and safe to read through its methods while the manager
// refreshes it in the background.
package token
