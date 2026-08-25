// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/hazelcast implements a distributed configuration consumer sample.
//
// This file is part of the Hazelcast sample bounded context within the
// Samples service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	consumerhazelcast "github.com/raywall/go-core-sdk/services/consumer/hazelcast"
)

const defaultMapName = "runtime-parameters"

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, out io.Writer) error {
	configPath, cleanup, err := sampleHazelcastConfigFile()
	if err != nil {
		return err
	}
	defer cleanup()

	client, err := consumerhazelcast.New(ctx, consumerhazelcast.Config{
		Source: consumerhazelcast.Source{
			Kind: consumerhazelcast.SourceFile,
			Path: configPath,
		},
		DefaultMap: defaultMapName,
	}, consumerhazelcast.WithMapProvider(newMemoryProvider(map[string]map[any]any{
		defaultMapName: {
			"payment.enabled":          true,
			"payment.max-installments": "5",
			"payment.strategy":         "oldest-first",
			"payment.rules":            `{"allowPartial":true,"minimumAmount":1000}`,
		},
	})))
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	return RuntimeParametersUseCase{
		Reader: client,
		Output: out,
	}.Execute(ctx)
}

func sampleHazelcastConfigFile() (string, func(), error) {
	file, err := os.CreateTemp("", "go-core-sdk-hazelcast-*.json")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	cleanup := func() {
		_ = os.Remove(path)
	}
	if _, err := file.WriteString(`{"Cluster":{"Name":"sample-cluster","Network":{"Addresses":["127.0.0.1:5701"]}}}`); err != nil {
		_ = file.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

// ParameterReader defines the application port used to read runtime parameters.
type ParameterReader interface {
	GetString(context.Context, string, any) (string, bool, error)
	GetBool(context.Context, string, any) (bool, bool, error)
	GetInt(context.Context, string, any) (int, bool, error)
	GetJSON(context.Context, string, any, any) (bool, error)
}

// RuntimeParametersUseCase demonstrates reading distributed parameters through a port.
type RuntimeParametersUseCase struct {
	Reader ParameterReader
	Output io.Writer
}

// Execute reads parameters and writes a compact result.
func (u RuntimeParametersUseCase) Execute(ctx context.Context) error {
	enabled, found, err := u.Reader.GetBool(ctx, "", "payment.enabled")
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("payment.enabled not found")
	}

	maxInstallments, found, err := u.Reader.GetInt(ctx, "", "payment.max-installments")
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("payment.max-installments not found")
	}

	strategy, found, err := u.Reader.GetString(ctx, "", "payment.strategy")
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("payment.strategy not found")
	}

	var rules PaymentRules
	found, err = u.Reader.GetJSON(ctx, "", "payment.rules", &rules)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("payment.rules not found")
	}

	_, err = fmt.Fprintf(u.Output, "enabled=%t maxInstallments=%d strategy=%s allowPartial=%t minimumAmount=%d\n",
		enabled,
		maxInstallments,
		strategy,
		rules.AllowPartial,
		rules.MinimumAmount,
	)
	return err
}

// PaymentRules contains payment parameters decoded from a Hazelcast map value.
type PaymentRules struct {
	AllowPartial  bool `json:"allowPartial"`
	MinimumAmount int  `json:"minimumAmount"`
}

type memoryProvider struct {
	maps map[string]*memoryMap
}

func newMemoryProvider(data map[string]map[any]any) *memoryProvider {
	provider := &memoryProvider{maps: make(map[string]*memoryMap)}
	for name, values := range data {
		copied := make(map[any]any, len(values))
		for key, value := range values {
			copied[key] = value
		}
		provider.maps[name] = &memoryMap{values: copied}
	}
	return provider
}

func (p *memoryProvider) GetMap(_ context.Context, name string) (consumerhazelcast.Map, error) {
	if p.maps == nil {
		p.maps = make(map[string]*memoryMap)
	}
	if _, ok := p.maps[name]; !ok {
		p.maps[name] = &memoryMap{values: make(map[any]any)}
	}
	return p.maps[name], nil
}

func (p *memoryProvider) Shutdown(context.Context) error {
	return nil
}

type memoryMap struct {
	values map[any]any
}

func (m *memoryMap) ContainsKey(_ context.Context, key interface{}) (bool, error) {
	_, ok := m.values[key]
	return ok, nil
}

func (m *memoryMap) Get(_ context.Context, key interface{}) (interface{}, error) {
	return m.values[key], nil
}
