// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// cache implements a temporary in-memory cache with TTL expiration.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

// Package cache provides a typed in-memory cache for short-lived runtime data.
//
// The service is useful in environments such as AWS Lambda, where the runtime
// may stay warm across invocations and can reuse temporary data while still
// respecting per-item TTL expiration.
//
// Usage:
//
//	store, err := cache.New[Customer](cache.Config{
//		DefaultTTL:      5 * time.Minute,
//		CleanupInterval: time.Minute,
//	})
//	if err != nil {
//		return err
//	}
//	if err := store.Start(ctx); err != nil {
//		return err
//	}
//	defer store.Stop()
//
//	if err := store.Add(ctx, "customer-123", customer); err != nil {
//		return err
//	}
//	customer, found, err := store.Get(ctx, "customer-123")
//
// Thread safety: Cache is safe for concurrent use. Query predicates are invoked
// after the cache releases its internal lock.
package cache
