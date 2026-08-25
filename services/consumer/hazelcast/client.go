// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast implements the distributed configuration client.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	hz "github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/serialization"
)

// Client reads runtime parameters from Hazelcast maps.
//
// Client is safe for concurrent use when the configured MapProvider is safe for
// concurrent use.
type Client struct {
	config   Config
	logger   logger
	provider MapProvider
}

// New constructs a Hazelcast Client from Config.
func New(ctx context.Context, config Config, configurers ...Option) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	if options.sourceLoader == nil {
		options.sourceLoader = defaultSourceLoader{
			awsClient: options.awsClient,
			logger:    options.logger,
		}
	}

	hazelcastConfig, err := loadHazelcastConfig(ctx, normalized, options.sourceLoader)
	if err != nil {
		return nil, err
	}
	normalized.HazelcastConfig = hazelcastConfig

	provider := options.mapProvider
	if provider == nil {
		options.logger.InfoContext(ctx, "consumer_hazelcast_start_started", "default_map", normalized.DefaultMap)
		provider, err = options.starter(ctx, hazelcastConfig)
		if err != nil {
			options.logger.ErrorContext(ctx, "consumer_hazelcast_start_failed", "error", err)
			return nil, HazelcastError{Operation: "start_client", Err: err}
		}
		options.logger.InfoContext(ctx, "consumer_hazelcast_start_completed", "default_map", normalized.DefaultMap)
	}

	return &Client{
		config:   normalized,
		logger:   options.logger,
		provider: provider,
	}, nil
}

// Config returns a copy of the normalized Hazelcast consumer configuration.
func (c *Client) Config() Config {
	if c == nil {
		return Config{}
	}
	return c.config
}

// HazelcastConfig returns the Hazelcast client configuration used by Client.
func (c *Client) HazelcastConfig() hz.Config {
	if c == nil {
		return hz.Config{}
	}
	return c.config.HazelcastConfig
}

// Close shuts down the underlying map provider.
func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.provider == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.provider.Shutdown(ctx); err != nil {
		return HazelcastError{Operation: "shutdown", Err: err}
	}
	return nil
}

// Get retrieves a raw value from a Hazelcast map.
func (c *Client) Get(ctx context.Context, mapName string, key any) (Value, error) {
	if c == nil || c.provider == nil {
		return Value{}, InvalidConfigError{Field: "Client", Reason: "is required"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	name, err := c.resolveMapName(mapName)
	if err != nil {
		return Value{}, err
	}
	if key == nil {
		return Value{}, InvalidConfigError{Field: "Key", Reason: "is required"}
	}

	c.logger.InfoContext(ctx, "consumer_hazelcast_get_started", "map", name, "key", key)
	hazelcastMap, err := c.provider.GetMap(ctx, name)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_hazelcast_get_failed", "map", name, "key", key, "error", err)
		return Value{}, HazelcastError{Operation: "get_map", MapName: name, Err: err}
	}
	found, err := hazelcastMap.ContainsKey(ctx, key)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_hazelcast_get_failed", "map", name, "key", key, "error", err)
		return Value{}, HazelcastError{Operation: "contains_key", MapName: name, Err: err}
	}
	if !found {
		c.logger.InfoContext(ctx, "consumer_hazelcast_get_completed", "map", name, "key", key, "found", false)
		return Value{MapName: name, Key: key, Found: false}, nil
	}
	raw, err := hazelcastMap.Get(ctx, key)
	if err != nil {
		c.logger.ErrorContext(ctx, "consumer_hazelcast_get_failed", "map", name, "key", key, "error", err)
		return Value{}, HazelcastError{Operation: "get_value", MapName: name, Err: err}
	}
	c.logger.InfoContext(ctx, "consumer_hazelcast_get_completed", "map", name, "key", key, "found", true)
	return Value{MapName: name, Key: key, Raw: raw, Found: true}, nil
}

// GetString retrieves a value and converts it to string.
func (c *Client) GetString(ctx context.Context, mapName string, key any) (string, bool, error) {
	value, err := c.Get(ctx, mapName, key)
	if err != nil || !value.Found {
		return "", false, err
	}
	converted, err := stringFromValue(value.Raw)
	if err != nil {
		return "", true, ValueConversionError{MapName: value.MapName, Key: value.Key, Expected: "string", Value: value.Raw}
	}
	return converted, true, nil
}

// GetBool retrieves a value and converts it to bool.
func (c *Client) GetBool(ctx context.Context, mapName string, key any) (bool, bool, error) {
	value, err := c.Get(ctx, mapName, key)
	if err != nil || !value.Found {
		return false, false, err
	}
	converted, err := boolFromValue(value.Raw)
	if err != nil {
		return false, true, ValueConversionError{MapName: value.MapName, Key: value.Key, Expected: "bool", Value: value.Raw}
	}
	return converted, true, nil
}

// GetInt retrieves a value and converts it to int.
func (c *Client) GetInt(ctx context.Context, mapName string, key any) (int, bool, error) {
	value, err := c.Get(ctx, mapName, key)
	if err != nil || !value.Found {
		return 0, false, err
	}
	converted, err := intFromValue(value.Raw)
	if err != nil {
		return 0, true, ValueConversionError{MapName: value.MapName, Key: value.Key, Expected: "int", Value: value.Raw}
	}
	return converted, true, nil
}

// GetJSON retrieves a value and decodes it as JSON into target.
func (c *Client) GetJSON(ctx context.Context, mapName string, key any, target any) (bool, error) {
	value, err := c.Get(ctx, mapName, key)
	if err != nil || !value.Found {
		return false, err
	}
	if target == nil {
		return true, InvalidConfigError{Field: "Target", Reason: "is required"}
	}
	data, err := jsonBytesFromValue(value.Raw)
	if err != nil {
		return true, ValueConversionError{MapName: value.MapName, Key: value.Key, Expected: "json", Value: value.Raw}
	}
	if err := json.Unmarshal(data, target); err != nil {
		return true, DecodeError{Operation: "hazelcast_json", Err: err}
	}
	return true, nil
}

// GetInto retrieves a value and decodes it into target using JSON conversion.
func (c *Client) GetInto(ctx context.Context, mapName string, key any, target any) (bool, error) {
	return c.GetJSON(ctx, mapName, key, target)
}

func (c *Client) resolveMapName(mapName string) (string, error) {
	name := strings.TrimSpace(mapName)
	if name == "" {
		name = c.config.DefaultMap
	}
	if name == "" {
		return "", InvalidConfigError{Field: "MapName", Reason: "is required"}
	}
	return name, nil
}

func loadHazelcastConfig(ctx context.Context, config Config, loader SourceLoader) (hz.Config, error) {
	hazelcastConfig := config.HazelcastConfig
	data, err := loader.Load(ctx, config.Source)
	if err != nil {
		return hz.Config{}, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &hazelcastConfig); err != nil {
			return hz.Config{}, DecodeError{Operation: "hazelcast_config_json", Err: err}
		}
	}
	if err := hazelcastConfig.Validate(); err != nil {
		return hz.Config{}, InvalidConfigError{Field: "HazelcastConfig", Reason: err.Error()}
	}
	return hazelcastConfig, nil
}

func stringFromValue(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case serialization.JSON:
		return string(typed), nil
	case fmt.Stringer:
		return typed.String(), nil
	default:
		return "", fmt.Errorf("unsupported string conversion from %T", value)
	}
}

func boolFromValue(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(typed))
	case []byte:
		return strconv.ParseBool(strings.TrimSpace(string(typed)))
	case serialization.JSON:
		return strconv.ParseBool(strings.TrimSpace(string(typed)))
	default:
		return false, fmt.Errorf("unsupported bool conversion from %T", value)
	}
}

func intFromValue(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int8:
		return int(typed), nil
	case int16:
		return int(typed), nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case uint:
		return int(typed), nil
	case uint8:
		return int(typed), nil
	case uint16:
		return int(typed), nil
	case uint32:
		return int(typed), nil
	case uint64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(typed))
	case []byte:
		return strconv.Atoi(strings.TrimSpace(string(typed)))
	case serialization.JSON:
		return strconv.Atoi(strings.TrimSpace(string(typed)))
	default:
		return 0, fmt.Errorf("unsupported int conversion from %T", value)
	}
}

func jsonBytesFromValue(value any) ([]byte, error) {
	switch typed := value.(type) {
	case serialization.JSON:
		return append([]byte(nil), typed...), nil
	case json.RawMessage:
		return append([]byte(nil), typed...), nil
	case []byte:
		return append([]byte(nil), typed...), nil
	case string:
		return []byte(typed), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
}
