// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// core implements runtime service composition.
//
// This file is part of the Core bounded context within the Core service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package core

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"github.com/raywall/go-core-sdk/config"
	"github.com/raywall/go-core-sdk/services/consumer"
	consumertypes "github.com/raywall/go-core-sdk/services/consumer/types"
	"github.com/raywall/go-core-sdk/services/decision"
	"github.com/raywall/go-core-sdk/services/observability"
	"github.com/raywall/go-core-sdk/services/selector"
	"github.com/raywall/go-core-sdk/services/token"
	"github.com/raywall/go-core-sdk/services/validation"
)

// Core contains shared go-core-sdk services for one application runtime.
//
// Core is safe to share after construction. Token manager lifecycle methods
// should be called from one shutdown path.
type Core struct {
	config    *config.Config
	logger    *slog.Logger
	telemetry *observability.Observability
	consumer  *consumer.Consumer
	decision  *decision.Decision
	selector  *selector.Selector
	validator *validation.Validator
	tokens    map[string]*token.Manager
}

// New builds a Core runtime from shared Config.
func New(ctx context.Context, cfg *config.Config, configurers ...Option) (*Core, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, InvalidConfigError{Field: "Config", Reason: "is required"}
	}
	options := options{}
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}

	observabilityOptions := make([]observability.Option, 0, len(options.observabilityOptions)+1)
	if logger, ok := cfg.ConfiguredLogger(); ok {
		observabilityOptions = append(observabilityOptions, observability.WithLogger(logger))
	}
	observabilityOptions = append(observabilityOptions, options.observabilityOptions...)
	telemetry, err := observability.New(cfg.Observability(), observabilityOptions...)
	if err != nil {
		return nil, err
	}
	logger := telemetry.Logger()
	consumerOptions := []consumer.Option{consumer.WithLogger(logger)}
	if awsConfig, ok := cfg.AWS(); ok {
		consumerOptions = append(consumerOptions, consumer.WithAWSConfig(awsConfig))
	}
	consumerOptions = append(consumerOptions, options.consumerOptions...)

	consumerService, err := consumer.New(cfg.Consumer(), consumerOptions...)
	if err != nil {
		_ = telemetry.Close()
		return nil, err
	}
	decisionService, err := decision.New(decision.WithLogger(logger))
	if err != nil {
		_ = telemetry.Close()
		return nil, err
	}
	selectorService, err := selector.New(selector.WithLogger(logger))
	if err != nil {
		_ = telemetry.Close()
		return nil, err
	}
	validatorService, err := validation.New(validation.WithLogger(logger))
	if err != nil {
		_ = telemetry.Close()
		return nil, err
	}

	runtime := &Core{
		config:    cfg,
		logger:    logger,
		telemetry: telemetry,
		consumer:  consumerService,
		decision:  decisionService,
		selector:  selectorService,
		validator: validatorService,
		tokens:    make(map[string]*token.Manager),
	}
	if err := runtime.buildTokenManagers(ctx); err != nil {
		_ = telemetry.Close()
		return nil, err
	}
	if options.tokenAutoStart {
		if err := runtime.Start(ctx); err != nil {
			runtime.Stop()
			return nil, err
		}
	}
	return runtime, nil
}

// Config returns the shared runtime configuration.
func (c *Core) Config() *config.Config {
	if c == nil {
		return nil
	}
	return c.config
}

// Logger returns the shared structured logger.
func (c *Core) Logger() *slog.Logger {
	if c == nil {
		return nil
	}
	return c.logger
}

// Observability returns the shared observability service.
func (c *Core) Observability() *observability.Observability {
	if c == nil {
		return nil
	}
	return c.telemetry
}

// Consumer returns the shared consumer service.
func (c *Core) Consumer() *consumer.Consumer {
	if c == nil {
		return nil
	}
	return c.consumer
}

// Decision returns the shared decision service.
func (c *Core) Decision() *decision.Decision {
	if c == nil {
		return nil
	}
	return c.decision
}

// Selector returns the shared selector service.
func (c *Core) Selector() *selector.Selector {
	if c == nil {
		return nil
	}
	return c.selector
}

// Validator returns the shared validation service.
func (c *Core) Validator() *validation.Validator {
	if c == nil {
		return nil
	}
	return c.validator
}

// TokenManager returns a named token manager.
func (c *Core) TokenManager(name string) (*token.Manager, bool) {
	if c == nil {
		return nil, false
	}
	manager, ok := c.tokens[strings.TrimSpace(name)]
	return manager, ok
}

// TokenManagerNames returns configured token manager names in stable order.
func (c *Core) TokenManagerNames() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.tokens))
	for name := range c.tokens {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Start starts all configured token managers.
func (c *Core) Start(ctx context.Context) error {
	if c == nil {
		return nil
	}
	for _, name := range c.TokenManagerNames() {
		if err := c.tokens[name].Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops all configured token managers.
func (c *Core) Stop() {
	if c == nil {
		return
	}
	for _, manager := range c.tokens {
		manager.Stop()
	}
	_ = c.telemetry.Close()
}

func (c *Core) buildTokenManagers(ctx context.Context) error {
	for name, tokenConfig := range c.config.Tokens() {
		resolved, err := c.resolveTokenConfig(ctx, tokenConfig)
		if err != nil {
			return err
		}
		manager, err := token.NewManager(resolved.TokenManager(), token.WithLogger(c.logger))
		if err != nil {
			return err
		}
		c.tokens[name] = manager
	}
	return nil
}

func (c *Core) resolveTokenConfig(ctx context.Context, tokenConfig config.TokenConfig) (config.TokenConfig, error) {
	if strings.TrimSpace(tokenConfig.SecretID) == "" {
		return tokenConfig, nil
	}
	var secret map[string]string
	if _, err := c.consumer.GetSecretJSON(ctx, consumertypes.SecretGetInput{SecretID: tokenConfig.SecretID}, &secret); err != nil {
		return config.TokenConfig{}, err
	}
	clientIDKey := strings.TrimSpace(tokenConfig.SecretClientIDKey)
	clientSecretKey := strings.TrimSpace(tokenConfig.SecretClientSecretKey)
	if tokenConfig.ClientID == "" {
		tokenConfig.ClientID = secret[clientIDKey]
	}
	if tokenConfig.ClientSecret == "" {
		tokenConfig.ClientSecret = secret[clientSecretKey]
	}
	return tokenConfig, nil
}
