// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// token implements lifecycle status values for the token manager.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package token

// Status represents the current lifecycle state of a token manager.
type Status string

const (
	// StatusStopped indicates the manager has no active background refresh
	// loop.
	StatusStopped Status = "stopped"

	// StatusStarting indicates Start is fetching the initial token.
	StatusStarting Status = "starting"

	// StatusRunning indicates the manager has a token and the background
	// refresh loop is active.
	StatusRunning Status = "running"

	// StatusRefreshing indicates the manager is actively requesting a new token.
	StatusRefreshing Status = "refreshing"

	// StatusDegraded indicates a refresh failed but the manager kept the last
	// known token and will retry.
	StatusDegraded Status = "degraded"

	// StatusStopping indicates Stop has requested cancellation and the
	// background loop is shutting down.
	StatusStopping Status = "stopping"
)
