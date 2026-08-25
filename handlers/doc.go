// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// handlers implements runtime-neutral handler contracts.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

// Package handlers defines runtime-neutral contracts for application handlers.
//
// The package keeps business processors independent from deployment targets.
// Runtime adapters such as handlers/lambda, handlers/http and handlers/worker
// convert platform-specific inputs into Event values and call the same
// Processor implementation.
package handlers
