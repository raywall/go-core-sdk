// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// internal implements clock abstractions for time-dependent token behavior.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package internal

import "time"

// Timer represents a stoppable time notification used by the token refresh
// loop.
type Timer interface {
	// C returns the channel that delivers the timer event.
	C() <-chan time.Time
	// Stop prevents the timer from firing when possible.
	Stop() bool
}

// Clock provides time and timer creation for the token manager.
type Clock interface {
	// Now returns the current instant.
	Now() time.Time
	// NewTimer creates a timer that fires after the provided duration.
	NewTimer(duration time.Duration) Timer
}

// RealClock uses the system clock and standard library timers.
type RealClock struct{}

// Now returns time.Now.
func (RealClock) Now() time.Time {
	return time.Now()
}

// NewTimer creates a standard library timer.
func (RealClock) NewTimer(duration time.Duration) Timer {
	return realTimer{timer: time.NewTimer(duration)}
}

type realTimer struct {
	timer *time.Timer
}

func (t realTimer) C() <-chan time.Time {
	return t.timer.C
}

func (t realTimer) Stop() bool {
	return t.timer.Stop()
}
