// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// internal implements clock abstractions for cache expiration.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package internal

import "time"

// Ticker represents a stoppable periodic time source.
type Ticker interface {
	// C returns the channel that receives periodic ticks.
	C() <-chan time.Time
	// Stop stops the ticker.
	Stop()
}

// Clock provides wall-clock access and ticker creation.
type Clock interface {
	// Now returns the current instant.
	Now() time.Time
	// NewTicker creates a ticker with the provided interval.
	NewTicker(interval time.Duration) Ticker
}

// RealClock uses the system clock.
type RealClock struct{}

// Now returns time.Now.
func (RealClock) Now() time.Time {
	return time.Now()
}

// NewTicker creates a standard library ticker.
func (RealClock) NewTicker(interval time.Duration) Ticker {
	return realTicker{ticker: time.NewTicker(interval)}
}

type realTicker struct {
	ticker *time.Ticker
}

func (t realTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t realTicker) Stop() {
	t.ticker.Stop()
}
