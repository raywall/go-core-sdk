// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// handlers implements configuration helpers.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package handlers

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultHTTPAddress        = ":8080"
	defaultHTTPPath           = "/events"
	defaultSQSMaxMessages     = int32(10)
	defaultSQSWaitTimeSeconds = int32(10)
)

// LoadConfigFromEnv loads handler settings from environment variables.
//
// The prefix is optional. For prefix APP, the loader reads variables such as
// APP_HANDLER_RUNTIME and APP_HANDLER_TRIGGER. Without a prefix, it reads
// HANDLER_RUNTIME and HANDLER_TRIGGER.
func LoadConfigFromEnv(prefix string) Config {
	key := envKey(prefix, "HANDLER_")
	return NormalizeConfig(Config{
		Runtime:              Runtime(os.Getenv(key("RUNTIME"))),
		Trigger:              Trigger(os.Getenv(key("TRIGGER"))),
		HTTPAddress:          os.Getenv(key("HTTP_ADDRESS")),
		HTTPPath:             os.Getenv(key("HTTP_PATH")),
		SQSQueueURL:          os.Getenv(key("SQS_QUEUE_URL")),
		SQSMaxMessages:       envInt32(key("SQS_MAX_MESSAGES")),
		SQSWaitTimeSeconds:   envInt32(key("SQS_WAIT_TIME_SECONDS")),
		SQSVisibilityTimeout: envInt32(key("SQS_VISIBILITY_TIMEOUT")),
	})
}

// NormalizeConfig returns a copy of config with defaults applied.
func NormalizeConfig(config Config) Config {
	normalized := config
	normalized.Runtime = Runtime(strings.TrimSpace(string(normalized.Runtime)))
	normalized.Trigger = Trigger(strings.TrimSpace(string(normalized.Trigger)))
	normalized.HTTPAddress = strings.TrimSpace(normalized.HTTPAddress)
	normalized.HTTPPath = strings.TrimSpace(normalized.HTTPPath)
	normalized.SQSQueueURL = strings.TrimSpace(normalized.SQSQueueURL)
	if normalized.HTTPAddress == "" {
		normalized.HTTPAddress = defaultHTTPAddress
	}
	if normalized.HTTPPath == "" {
		normalized.HTTPPath = defaultHTTPPath
	}
	if normalized.SQSMaxMessages == 0 {
		normalized.SQSMaxMessages = defaultSQSMaxMessages
	}
	if normalized.SQSWaitTimeSeconds == 0 {
		normalized.SQSWaitTimeSeconds = defaultSQSWaitTimeSeconds
	}
	return normalized
}

func envKey(prefix string, base string) func(string) string {
	trimmed := strings.Trim(strings.ToUpper(strings.TrimSpace(prefix)), "_")
	return func(name string) string {
		if trimmed == "" {
			return base + name
		}
		return trimmed + "_" + base + name
	}
}

func envInt32(name string) int32 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int32(parsed)
}
