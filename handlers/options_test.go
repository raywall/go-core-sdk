// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// handlers tests configuration helpers.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package handlers_test

import (
	"testing"

	"github.com/raywall/go-core-sdk/handlers"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("APP_HANDLER_RUNTIME", "http")
	t.Setenv("APP_HANDLER_TRIGGER", "s3")
	t.Setenv("APP_HANDLER_HTTP_ADDRESS", ":9090")
	t.Setenv("APP_HANDLER_SQS_MAX_MESSAGES", "3")

	config := handlers.LoadConfigFromEnv("APP")

	if config.Runtime != handlers.RuntimeHTTP {
		t.Fatalf("Runtime = %q, want %q", config.Runtime, handlers.RuntimeHTTP)
	}
	if config.Trigger != handlers.TriggerS3 {
		t.Fatalf("Trigger = %q, want %q", config.Trigger, handlers.TriggerS3)
	}
	if config.HTTPAddress != ":9090" {
		t.Fatalf("HTTPAddress = %q, want :9090", config.HTTPAddress)
	}
	if config.SQSMaxMessages != 3 {
		t.Fatalf("SQSMaxMessages = %d, want 3", config.SQSMaxMessages)
	}
}
