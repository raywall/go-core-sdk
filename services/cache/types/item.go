// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types implements cache item metadata.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "time"

// Item stores one cached value and its expiration metadata.
type Item[T any] struct {
	// Key identifies the cached value.
	Key string
	// Value is the cached entity.
	Value T
	// CreatedAt is the instant when the item was added to the cache.
	CreatedAt time.Time
	// ExpiresAt is the instant when the item expires.
	ExpiresAt time.Time
	// TTL is the lifetime configured for this item.
	TTL time.Duration
}

// Expired reports whether the item is expired at now.
func (i Item[T]) Expired(now time.Time) bool {
	return !i.ExpiresAt.After(now)
}

// QueryPredicate decides whether an item should be returned from Query.
//
// Query only invokes the predicate for items that are still valid at query time.
type QueryPredicate[T any] func(Item[T]) bool
