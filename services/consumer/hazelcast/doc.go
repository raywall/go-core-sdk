// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements distributed configuration consumer adapters.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

// Package hazelcast provides helpers for reading runtime parameters from
// Hazelcast maps.
//
// The package can load Hazelcast client configuration from inline JSON, a local
// file, S3 or Secrets Manager, then expose typed getters over distributed maps.
package hazelcast
