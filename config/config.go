// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// config implements the central runtime configuration model.
//
// This file is part of the Config bounded context within the Config service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package config

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/raywall/go-core-sdk/services/cache"
	"github.com/raywall/go-core-sdk/services/consumer"
	"github.com/raywall/go-core-sdk/services/observability"
	"github.com/raywall/go-core-sdk/services/token"
)

const (
	defaultHTTPTimeout    = 30 * time.Second
	defaultCacheTTL       = 5 * time.Minute
	defaultCleanupEvery   = time.Minute
	defaultSecretIDKey    = "client_id"
	defaultSecretValueKey = "client_secret"
)

// Config contains shared runtime configuration and service-specific projections.
type Config struct {
	serviceName string
	environment string
	version     string
	awsRegion   string
	httpTimeout time.Duration
	logger      *slog.Logger

	awsConfig     *aws.Config
	customLogger  bool
	cache         CacheConfig
	observability ObservabilityConfig
	tokens        map[string]TokenConfig
}

// CacheConfig contains shared cache defaults.
type CacheConfig struct {
	// DefaultTTL is used by cache services when an item-specific TTL is absent.
	DefaultTTL time.Duration
	// CleanupInterval controls automatic cache cleanup frequency.
	CleanupInterval time.Duration
}

// ObservabilityConfig contains shared observability defaults.
type ObservabilityConfig struct {
	// ServiceName is added to structured logs when configured.
	ServiceName string
	// Environment is added to structured logs and emitted as the env metric tag.
	Environment string
	// Version is added to structured logs when configured.
	Version string
	// MetricPrefix is prepended to every metric name.
	MetricPrefix string
	// DatadogAddress is the DogStatsD address used by the default metrics client.
	DatadogAddress string
	// DefaultTags are added to every metric.
	DefaultTags []string
	// LogLevel controls the minimum level for the default JSON logger.
	LogLevel slog.Level
}

// TokenConfig contains token manager configuration with optional secret lookup metadata.
type TokenConfig struct {
	// BaseURL is the STS base URL, such as https://sts.example.com.
	BaseURL string
	// Endpoint is the token endpoint path, such as /oauth/token.
	Endpoint string
	// Headers contains additional headers sent with every STS request.
	Headers map[string]string
	// ValidateSSL controls TLS certificate verification for HTTPS requests.
	ValidateSSL bool
	// ClientID is the OAuth client identifier used by client_credentials.
	ClientID string
	// ClientSecret is the OAuth client secret used by client_credentials.
	ClientSecret string
	// SecretID is an optional Secrets Manager secret name or ARN containing credentials.
	SecretID string
	// SecretClientIDKey is the JSON field used to resolve ClientID from SecretID.
	SecretClientIDKey string
	// SecretClientSecretKey is the JSON field used to resolve ClientSecret from SecretID.
	SecretClientSecretKey string
	// RefreshBefore is how long before expiry the token should be renewed.
	RefreshBefore time.Duration
	// RequestTimeout is the timeout used by the token HTTP client.
	RequestTimeout time.Duration
	// MaxRetries is the number of retry attempts after an STS request failure.
	MaxRetries int
	// RetryBackoff is the initial delay between retry attempts.
	RetryBackoff time.Duration
}

// Load builds a Config from options, loaders and resolvers.
func Load(ctx context.Context, options ...Option) (*Config, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state := loadState{
		config: Config{
			httpTimeout: defaultHTTPTimeout,
			logger:      slog.New(slog.NewJSONHandler(os.Stdout, nil)),
			cache: CacheConfig{
				DefaultTTL:      defaultCacheTTL,
				CleanupInterval: defaultCleanupEvery,
			},
			tokens: make(map[string]TokenConfig),
		},
		resolvers: []Resolver{resolveDefaults, validate},
	}

	for _, option := range options {
		if option != nil {
			option(&state)
		}
	}
	for _, loader := range state.loaders {
		loaded, err := loader(ctx, state.config)
		if err != nil {
			return nil, err
		}
		state.config = loaded
	}
	for _, resolver := range state.resolvers {
		if err := resolver(ctx, &state.config); err != nil {
			return nil, err
		}
	}
	return state.config.clone(), nil
}

// ServiceName returns the configured service name.
func (c *Config) ServiceName() string {
	if c == nil {
		return ""
	}
	return c.serviceName
}

// Environment returns the configured runtime environment.
func (c *Config) Environment() string {
	if c == nil {
		return ""
	}
	return c.environment
}

// Version returns the configured application version.
func (c *Config) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

// AWSRegion returns the configured AWS region.
func (c *Config) AWSRegion() string {
	if c == nil {
		return ""
	}
	return c.awsRegion
}

// Logger returns the configured structured logger.
func (c *Config) Logger() *slog.Logger {
	if c == nil || c.logger == nil {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return c.logger
}

// ConfiguredLogger returns a logger explicitly configured through WithLogger.
func (c *Config) ConfiguredLogger() (*slog.Logger, bool) {
	if c == nil || c.logger == nil || !c.customLogger {
		return nil, false
	}
	return c.logger, true
}

// AWS returns the loaded AWS config and whether one was configured.
func (c *Config) AWS() (aws.Config, bool) {
	if c == nil || c.awsConfig == nil {
		return aws.Config{}, false
	}
	return *c.awsConfig, true
}

// Consumer projects shared configuration to consumer.Config.
func (c *Config) Consumer() consumer.Config {
	if c == nil {
		return consumer.Config{}
	}
	return consumer.Config{
		HTTPTimeout: c.httpTimeout,
		AWSRegion:   c.awsRegion,
	}
}

// Cache projects shared configuration to cache.Config.
func (c *Config) Cache() cache.Config {
	if c == nil {
		return cache.Config{}
	}
	return cache.Config{
		DefaultTTL:      c.cache.DefaultTTL,
		CleanupInterval: c.cache.CleanupInterval,
	}
}

// Observability projects shared configuration to observability.Config.
func (c *Config) Observability() observability.Config {
	if c == nil {
		return observability.Config{}
	}
	return observability.Config{
		ServiceName:    c.observability.ServiceName,
		Environment:    c.observability.Environment,
		Version:        c.observability.Version,
		MetricPrefix:   c.observability.MetricPrefix,
		DatadogAddress: c.observability.DatadogAddress,
		DefaultTags:    copyStringSlice(c.observability.DefaultTags),
		LogLevel:       c.observability.LogLevel,
	}
}

// Token returns a named token configuration.
func (c *Config) Token(name string) (TokenConfig, bool) {
	if c == nil {
		return TokenConfig{}, false
	}
	cfg, ok := c.tokens[strings.TrimSpace(name)]
	if !ok {
		return TokenConfig{}, false
	}
	return cfg.clone(), true
}

// Tokens returns a copy of all named token configurations.
func (c *Config) Tokens() map[string]TokenConfig {
	if c == nil {
		return nil
	}
	copied := make(map[string]TokenConfig, len(c.tokens))
	for name, cfg := range c.tokens {
		copied[name] = cfg.clone()
	}
	return copied
}

// TokenManager projects a TokenConfig to token.Config.
func (c TokenConfig) TokenManager() token.Config {
	return token.Config{
		BaseURL:        strings.TrimSpace(c.BaseURL),
		Endpoint:       strings.TrimSpace(c.Endpoint),
		Headers:        copyStringMap(c.Headers),
		ValidateSSL:    c.ValidateSSL,
		ClientID:       strings.TrimSpace(c.ClientID),
		ClientSecret:   strings.TrimSpace(c.ClientSecret),
		RefreshBefore:  c.RefreshBefore,
		RequestTimeout: c.RequestTimeout,
		MaxRetries:     c.MaxRetries,
		RetryBackoff:   c.RetryBackoff,
	}
}

func (c *Config) clone() *Config {
	if c == nil {
		return nil
	}
	copied := *c
	copied.observability = c.observability.clone()
	copied.tokens = make(map[string]TokenConfig, len(c.tokens))
	for name, cfg := range c.tokens {
		copied.tokens[name] = cfg.clone()
	}
	if c.awsConfig != nil {
		awsCfg := *c.awsConfig
		copied.awsConfig = &awsCfg
	}
	return &copied
}

func (c ObservabilityConfig) clone() ObservabilityConfig {
	c.DefaultTags = copyStringSlice(c.DefaultTags)
	return c
}

func (c TokenConfig) clone() TokenConfig {
	c.Headers = copyStringMap(c.Headers)
	return c
}

func resolveDefaults(_ context.Context, cfg *Config) error {
	cfg.serviceName = strings.TrimSpace(cfg.serviceName)
	cfg.environment = strings.TrimSpace(cfg.environment)
	cfg.version = strings.TrimSpace(cfg.version)
	cfg.awsRegion = strings.TrimSpace(cfg.awsRegion)
	cfg.observability.ServiceName = defaultIfEmpty(cfg.observability.ServiceName, cfg.serviceName)
	cfg.observability.Environment = defaultIfEmpty(cfg.observability.Environment, cfg.environment)
	cfg.observability.Version = defaultIfEmpty(cfg.observability.Version, cfg.version)
	cfg.observability.MetricPrefix = strings.Trim(strings.TrimSpace(cfg.observability.MetricPrefix), ".")
	cfg.observability.DatadogAddress = strings.TrimSpace(cfg.observability.DatadogAddress)
	cfg.observability.DefaultTags = copyStringSlice(cfg.observability.DefaultTags)
	if cfg.httpTimeout == 0 {
		cfg.httpTimeout = defaultHTTPTimeout
	}
	if cfg.logger == nil {
		cfg.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	if cfg.cache.DefaultTTL == 0 {
		cfg.cache.DefaultTTL = defaultCacheTTL
	}
	if cfg.cache.CleanupInterval == 0 {
		cfg.cache.CleanupInterval = defaultCleanupEvery
	}
	for name, tokenConfig := range cfg.tokens {
		tokenConfig.SecretClientIDKey = defaultIfEmpty(tokenConfig.SecretClientIDKey, defaultSecretIDKey)
		tokenConfig.SecretClientSecretKey = defaultIfEmpty(tokenConfig.SecretClientSecretKey, defaultSecretValueKey)
		tokenConfig.Headers = copyStringMap(tokenConfig.Headers)
		cfg.tokens[name] = tokenConfig
	}
	return nil
}

func validate(_ context.Context, cfg *Config) error {
	if cfg.httpTimeout < 0 {
		return InvalidConfigError{Field: "HTTPTimeout", Reason: "must not be negative"}
	}
	if cfg.cache.DefaultTTL < 0 {
		return InvalidConfigError{Field: "Cache.DefaultTTL", Reason: "must not be negative"}
	}
	if cfg.cache.CleanupInterval < 0 {
		return InvalidConfigError{Field: "Cache.CleanupInterval", Reason: "must not be negative"}
	}
	return nil
}

func loadAWSDefaultConfig(ctx context.Context, cfg Config) (Config, error) {
	options := []func(*awsconfig.LoadOptions) error{}
	if cfg.awsRegion != "" {
		options = append(options, awsconfig.WithRegion(cfg.awsRegion))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return Config{}, err
	}
	cfg.awsConfig = &awsCfg
	return cfg, nil
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}
