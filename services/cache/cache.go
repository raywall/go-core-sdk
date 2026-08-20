// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// cache implements the typed in-memory cache.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package cache

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	cacheinternal "github.com/raywall/go-core-sdk/services/cache/internal"
	"github.com/raywall/go-core-sdk/services/cache/types"
)

// Cache stores typed entities in memory and expires them by TTL.
//
// Cache is safe for concurrent use. Start controls only the automatic cleanup
// loop; Add, Get, Remove, Clean and Query can be used before or after Start.
type Cache[T any] struct {
	config Config
	logger *slog.Logger
	clock  cacheinternal.Clock

	mu     sync.RWMutex
	items  map[string]types.Item[T]
	status Status
	cancel context.CancelFunc
	done   chan struct{}
}

// New constructs a typed cache.
//
// The returned cache is stopped and ready to accept items. Call Start when
// automatic cleanup is desired.
func New[T any](config Config, configurers ...Option) (*Cache[T], error) {
	normalized := normalizeConfig(config)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}

	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}

	return &Cache[T]{
		config: normalized,
		logger: options.logger,
		clock:  options.clock,
		items:  make(map[string]types.Item[T]),
		status: StatusStopped,
	}, nil
}

// Start starts the automatic cleanup loop.
//
// Start is idempotent. Calling Start while the cache is already running returns
// nil and keeps the existing cleanup loop.
func (c *Cache[T]) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	c.mu.Lock()
	if c.status != StatusStopped {
		c.mu.Unlock()
		cancel()
		return nil
	}
	c.status = StatusRunning
	c.cancel = cancel
	c.done = done
	c.mu.Unlock()

	go c.runCleanup(runCtx, done)
	c.logger.InfoContext(runCtx, "cache_started", "default_ttl", c.config.DefaultTTL.String(), "cleanup_interval", c.config.CleanupInterval.String())
	return nil
}

// Stop stops the automatic cleanup loop.
//
// Stop is idempotent. It does not remove cached items.
func (c *Cache[T]) Stop() {
	cancel, done := c.beginStop()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// Status returns the current lifecycle state of the cache.
func (c *Cache[T]) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// Add stores value under key using the default TTL or an optional item TTL.
//
// Passing more than one TTL is invalid. A zero TTL uses the cache default TTL.
func (c *Cache[T]) Add(ctx context.Context, key string, value T, ttl ...time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	key, err := normalizeKey(key)
	if err != nil {
		return err
	}
	itemTTL, err := c.itemTTL(ttl...)
	if err != nil {
		return err
	}

	now := c.clock.Now()
	item := types.Item[T]{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		ExpiresAt: now.Add(itemTTL),
		TTL:       itemTTL,
	}

	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()

	c.logger.DebugContext(ctx, "cache_item_added", "key", key, "ttl", itemTTL.String(), "expires_at", item.ExpiresAt)
	return nil
}

// Get retrieves a valid cached value by key.
//
// Get returns found=false when the key does not exist or the item has expired.
// Expired items found during Get are removed from the cache.
func (c *Cache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	item, found, err := c.GetItem(ctx, key)
	if err != nil || !found {
		var zero T
		return zero, found, err
	}
	return item.Value, true, nil
}

// GetItem retrieves a valid cached item with expiration metadata.
//
// GetItem returns found=false when the key does not exist or the item has
// expired. Expired items found during GetItem are removed from the cache.
func (c *Cache[T]) GetItem(ctx context.Context, key string) (types.Item[T], bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return types.Item[T]{}, false, err
	}

	key, err := normalizeKey(key)
	if err != nil {
		return types.Item[T]{}, false, err
	}

	now := c.clock.Now()
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()
	if !found {
		return types.Item[T]{}, false, nil
	}
	if item.Expired(now) {
		c.removeIfExpired(key, now)
		return types.Item[T]{}, false, nil
	}
	return item, true, nil
}

// Remove deletes a cache item by key.
//
// Remove is idempotent. Removing a key that does not exist returns nil.
func (c *Cache[T]) Remove(ctx context.Context, key string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	key, err := normalizeKey(key)
	if err != nil {
		return err
	}

	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()

	c.logger.DebugContext(ctx, "cache_item_removed", "key", key)
	return nil
}

// Clean removes all cached items.
func (c *Cache[T]) Clean(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	c.mu.Lock()
	removed := len(c.items)
	c.items = make(map[string]types.Item[T])
	c.mu.Unlock()

	c.logger.InfoContext(ctx, "cache_cleaned", "removed", removed)
	return nil
}

// Query returns all valid items that match predicate.
//
// Query removes expired records before evaluating predicate. If predicate is
// nil, all valid items are returned. The predicate is invoked after the cache
// releases its internal lock.
func (c *Cache[T]) Query(ctx context.Context, predicate types.QueryPredicate[T]) ([]types.Item[T], error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	items := c.validItems(c.clock.Now())
	if predicate == nil {
		return items, nil
	}

	matches := make([]types.Item[T], 0, len(items))
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if predicate(item) {
			matches = append(matches, item)
		}
	}
	return matches, nil
}

// Len returns the number of valid items currently stored in the cache.
func (c *Cache[T]) Len() int {
	return len(c.validItems(c.clock.Now()))
}

// Keys returns the keys for all valid items currently stored in the cache.
func (c *Cache[T]) Keys() []string {
	items := c.validItems(c.clock.Now())
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

func (c *Cache[T]) runCleanup(ctx context.Context, done chan struct{}) {
	defer func() {
		c.finishStop(done)
		close(done)
		c.logger.InfoContext(ctx, "cache_stopped")
	}()

	ticker := c.clock.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			removed := c.PurgeExpired(ctx)
			if removed > 0 {
				c.logger.DebugContext(ctx, "cache_expired_items_purged", "removed", removed)
			}
		}
	}
}

// PurgeExpired removes expired items and returns how many records were removed.
func (c *Cache[T]) PurgeExpired(ctx context.Context) int {
	if ctx == nil {
		ctx = context.Background()
	}
	now := c.clock.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for key, item := range c.items {
		if item.Expired(now) {
			delete(c.items, key)
			removed++
		}
	}
	return removed
}

func (c *Cache[T]) beginStop() (context.CancelFunc, chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.status == StatusStopped {
		return nil, nil
	}
	c.status = StatusStopping
	return c.cancel, c.done
}

func (c *Cache[T]) finishStop(done chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.done != done {
		return
	}
	c.status = StatusStopped
	c.cancel = nil
	c.done = nil
}

func (c *Cache[T]) validItems(now time.Time) []types.Item[T] {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([]types.Item[T], 0, len(c.items))
	for key, item := range c.items {
		if item.Expired(now) {
			delete(c.items, key)
			continue
		}
		items = append(items, item)
	}
	return items
}

func (c *Cache[T]) removeIfExpired(key string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if found && item.Expired(now) {
		delete(c.items, key)
	}
}

func (c *Cache[T]) itemTTL(ttl ...time.Duration) (time.Duration, error) {
	switch len(ttl) {
	case 0:
		return c.config.DefaultTTL, nil
	case 1:
		if ttl[0] < 0 {
			return 0, types.InvalidTTLError{TTL: ttl[0].String(), Reason: "must not be negative"}
		}
		if ttl[0] == 0 {
			return c.config.DefaultTTL, nil
		}
		return ttl[0], nil
	default:
		return 0, types.InvalidTTLError{Reason: "only one ttl value can be provided"}
	}
}

func normalizeKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", types.InvalidKeyError{Reason: "is required"}
	}
	return key, nil
}
