// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements public token value objects and errors.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package types contains the public value objects and errors used by the token
// service.
//
// The package exposes a concurrency-safe Token object that can be held by
// callers while the manager refreshes it in the background. Callers should read
// token data through methods such as ToString, Snapshot and AccessToken rather
// than retaining copied field values.
package types
