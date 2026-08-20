// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// cache implements cache lifecycle status values.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package cache

// Status represents the lifecycle state of a cache instance.
type Status string

const (
	// StatusStopped indicates the cache has no active automatic cleanup loop.
	StatusStopped Status = "stopped"

	// StatusRunning indicates the cache automatic cleanup loop is running.
	StatusRunning Status = "running"

	// StatusStopping indicates Stop has requested cancellation and the cleanup
	// loop is shutting down.
	StatusStopping Status = "stopping"
)
