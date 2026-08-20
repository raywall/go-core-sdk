// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// cache tests typed in-memory cache behavior.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package cache

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	cacheinternal "github.com/raywall/go-core-sdk/services/cache/internal"
	"github.com/raywall/go-core-sdk/services/cache/types"
)

type customer struct {
	ID     string
	Active bool
}

// TestCache_AddGetAndMetadata verifies values can be added, read and inspected
// with TTL metadata.
func TestCache_AddGetAndMetadata(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[customer](t, clock, Config{DefaultTTL: time.Minute, CleanupInterval: time.Minute})

	if err := store.Add(context.Background(), " c-1 ", customer{ID: "c-1", Active: true}); err != nil {
		t.Fatalf("Cache.Add() error = %v", err)
	}

	got, found, err := store.Get(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("Cache.Get() error = %v", err)
	}
	if !found || got.ID != "c-1" {
		t.Fatalf("Cache.Get() = %+v, %v; want customer c-1, true", got, found)
	}

	item, found, err := store.GetItem(context.Background(), "c-1")
	if err != nil {
		t.Fatalf("Cache.GetItem() error = %v", err)
	}
	if !found {
		t.Fatal("Cache.GetItem() found = false, want true")
	}
	if item.TTL != time.Minute || !item.ExpiresAt.Equal(clock.Now().Add(time.Minute)) {
		t.Fatalf("item metadata = %+v, want one minute TTL", item)
	}
}

// TestCache_ExpiredItemsAreNotReturned verifies expired items are removed when
// accessed.
func TestCache_ExpiredItemsAreNotReturned(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[string](t, clock, Config{DefaultTTL: time.Minute, CleanupInterval: time.Minute})

	if err := store.Add(context.Background(), "token", "value", 10*time.Second); err != nil {
		t.Fatalf("Cache.Add() error = %v", err)
	}
	clock.Advance(11 * time.Second)

	got, found, err := store.Get(context.Background(), "token")
	if err != nil {
		t.Fatalf("Cache.Get() error = %v", err)
	}
	if found || got != "" {
		t.Fatalf("Cache.Get() = %q, %v; want zero, false", got, found)
	}
	if got := store.Len(); got != 0 {
		t.Fatalf("Cache.Len() = %d, want 0", got)
	}
}

// TestCache_RemoveCleanAndQuery verifies explicit item operations and query
// filtering over valid records.
func TestCache_RemoveCleanAndQuery(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[customer](t, clock, Config{DefaultTTL: time.Minute, CleanupInterval: time.Minute})

	_ = store.Add(context.Background(), "a", customer{ID: "a", Active: true})
	_ = store.Add(context.Background(), "b", customer{ID: "b", Active: false})
	if err := store.Remove(context.Background(), "b"); err != nil {
		t.Fatalf("Cache.Remove() error = %v", err)
	}

	matches, err := store.Query(context.Background(), func(item types.Item[customer]) bool {
		if err := store.Remove(context.Background(), "missing"); err != nil {
			t.Fatalf("Cache.Remove() inside predicate error = %v", err)
		}
		return item.Value.Active
	})
	if err != nil {
		t.Fatalf("Cache.Query() error = %v", err)
	}
	if len(matches) != 1 || matches[0].Key != "a" {
		t.Fatalf("Cache.Query() = %+v, want only key a", matches)
	}

	if err := store.Clean(context.Background()); err != nil {
		t.Fatalf("Cache.Clean() error = %v", err)
	}
	if got := store.Len(); got != 0 {
		t.Fatalf("Cache.Len() = %d, want 0", got)
	}
}

// TestCache_DefaultAndItemTTL verifies Add uses the default TTL unless a custom
// item TTL is provided.
func TestCache_DefaultAndItemTTL(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[string](t, clock, Config{DefaultTTL: time.Minute, CleanupInterval: time.Minute})

	_ = store.Add(context.Background(), "default", "a")
	_ = store.Add(context.Background(), "custom", "b", 2*time.Minute)

	defaultItem, _, _ := store.GetItem(context.Background(), "default")
	customItem, _, _ := store.GetItem(context.Background(), "custom")

	if defaultItem.TTL != time.Minute {
		t.Fatalf("default TTL = %s, want 1m", defaultItem.TTL)
	}
	if customItem.TTL != 2*time.Minute {
		t.Fatalf("custom TTL = %s, want 2m", customItem.TTL)
	}
}

// TestCache_StartStopAndAutomaticPurge verifies lifecycle operations are
// idempotent and the cleanup loop removes expired items.
func TestCache_StartStopAndAutomaticPurge(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[string](t, clock, Config{DefaultTTL: 10 * time.Second, CleanupInterval: time.Second})

	if err := store.Start(context.Background()); err != nil {
		t.Fatalf("Cache.Start() error = %v", err)
	}
	if err := store.Start(context.Background()); err != nil {
		t.Fatalf("second Cache.Start() error = %v", err)
	}
	if got, want := store.Status(), StatusRunning; got != want {
		t.Fatalf("Cache.Status() = %q, want %q", got, want)
	}

	if err := store.Add(context.Background(), "key", "value"); err != nil {
		t.Fatalf("Cache.Add() error = %v", err)
	}
	clock.Advance(11 * time.Second)
	clock.Tick(t)
	waitForLen(t, store, 0)

	store.Stop()
	store.Stop()
	if got, want := store.Status(), StatusStopped; got != want {
		t.Fatalf("Cache.Status() = %q, want %q", got, want)
	}
}

// TestCache_ReturnsTypedErrors verifies invalid inputs return exported typed
// errors.
func TestCache_ReturnsTypedErrors(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[string](t, clock, Config{DefaultTTL: time.Minute, CleanupInterval: time.Minute})

	if err := store.Add(context.Background(), "", "value"); !isInvalidKey(err) {
		t.Fatalf("Cache.Add empty key error = %T %[1]v, want InvalidKeyError", err)
	}
	if err := store.Add(context.Background(), "key", "value", -time.Second); !isInvalidTTL(err) {
		t.Fatalf("Cache.Add invalid ttl error = %T %[1]v, want InvalidTTLError", err)
	}
	if _, err := New[string](Config{DefaultTTL: -time.Second}); !isInvalidConfig(err) {
		t.Fatalf("cache.New invalid config error = %T %[1]v, want InvalidConfigError", err)
	}
}

// TestCache_IsRaceSafe exercises concurrent Add, Get and Remove calls. The race
// detector verifies shared state safety.
func TestCache_IsRaceSafe(t *testing.T) {
	t.Parallel()

	clock := newManualClock(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC))
	store := newTestCache[int](t, clock, Config{DefaultTTL: time.Minute, CleanupInterval: time.Minute})

	var wg sync.WaitGroup
	errs := make(chan error, 300)
	for workerID := range 25 {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			key := "key"
			if err := store.Add(context.Background(), key, workerID); err != nil {
				errs <- err
				return
			}
			if _, _, err := store.Get(context.Background(), key); err != nil {
				errs <- err
				return
			}
			if err := store.Remove(context.Background(), key); err != nil {
				errs <- err
				return
			}
		}(workerID)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent cache operation error = %v", err)
		}
	}
}

func newTestCache[T any](t *testing.T, clock *manualClock, config Config) *Cache[T] {
	t.Helper()

	store, err := New[T](
		config,
		WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))),
		withClock(clock),
	)
	if err != nil {
		t.Fatalf("cache.New() error = %v", err)
	}
	return store
}

func waitForLen[T any](t *testing.T, store *Cache[T], want int) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if got := store.Len(); got == want {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for cache length %d, got %d", want, store.Len())
		}
	}
}

func isInvalidKey(err error) bool {
	var target types.InvalidKeyError
	return errors.As(err, &target)
}

func isInvalidTTL(err error) bool {
	var target types.InvalidTTLError
	return errors.As(err, &target)
}

func isInvalidConfig(err error) bool {
	var target types.InvalidConfigError
	return errors.As(err, &target)
}

type manualClock struct {
	mu          sync.Mutex
	now         time.Time
	ticker      *manualTicker
	tickerReady chan struct{}
	readyOnce   sync.Once
}

func newManualClock(now time.Time) *manualClock {
	return &manualClock{
		now:         now,
		tickerReady: make(chan struct{}),
	}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) NewTicker(_ time.Duration) cacheinternal.Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ticker = &manualTicker{ch: make(chan time.Time, 10)}
	c.readyOnce.Do(func() { close(c.tickerReady) })
	return c.ticker
}

func (c *manualClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

func (c *manualClock) Tick(t *testing.T) {
	t.Helper()

	select {
	case <-c.tickerReady:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cache ticker")
	}

	now := c.Now()
	select {
	case c.ticker.ch <- now:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out sending cache ticker event")
	}
}

type manualTicker struct {
	ch chan time.Time
}

func (t *manualTicker) C() <-chan time.Time {
	return t.ch
}

func (t *manualTicker) Stop() {}
