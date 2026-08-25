// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// hazelcast tests distributed configuration consumer behavior.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package hazelcast_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	hz "github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/serialization"
	"github.com/raywall/go-core-sdk/services/consumer/hazelcast"
)

func TestNew_LoadsInlineConfigAndUsesStarter(t *testing.T) {
	t.Parallel()

	var gotConfig hz.Config
	provider := newFakeProvider(map[string]map[any]any{"params": {"feature.enabled": "true"}})
	client, err := hazelcast.New(context.Background(), hazelcast.Config{
		Source:     hazelcast.Source{Kind: hazelcast.SourceInline, Data: []byte(`{"Cluster":{"Name":"payments","Network":{"Addresses":["127.0.0.1:5701"]}}}`)},
		DefaultMap: "params",
	}, hazelcast.WithLogger(discardLogger()), hazelcast.WithStarter(func(_ context.Context, config hz.Config) (hazelcast.MapProvider, error) {
		gotConfig = config
		return provider, nil
	}))
	if err != nil {
		t.Fatalf("hazelcast.New() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close(context.Background()) })

	if gotConfig.Cluster.Name != "payments" {
		t.Fatalf("starter config cluster name = %q, want payments", gotConfig.Cluster.Name)
	}
	if client.Config().DefaultMap != "params" {
		t.Fatalf("DefaultMap = %q, want params", client.Config().DefaultMap)
	}
}

func TestClient_GetTypedValues(t *testing.T) {
	t.Parallel()

	client := newClient(t, map[string]map[any]any{
		"params": {
			"feature.enabled": true,
			"max.items":       "42",
			"service.name":    []byte("payments"),
			"rules":           serialization.JSON(`{"limit":5,"mode":"partial"}`),
		},
	})

	name, found, err := client.GetString(context.Background(), "", "service.name")
	if err != nil {
		t.Fatalf("GetString() error = %v", err)
	}
	if !found || name != "payments" {
		t.Fatalf("GetString() = %q, %t; want payments, true", name, found)
	}

	enabled, found, err := client.GetBool(context.Background(), "", "feature.enabled")
	if err != nil {
		t.Fatalf("GetBool() error = %v", err)
	}
	if !found || !enabled {
		t.Fatalf("GetBool() = %t, %t; want true, true", enabled, found)
	}

	maxItems, found, err := client.GetInt(context.Background(), "", "max.items")
	if err != nil {
		t.Fatalf("GetInt() error = %v", err)
	}
	if !found || maxItems != 42 {
		t.Fatalf("GetInt() = %d, %t; want 42, true", maxItems, found)
	}

	var rules struct {
		Limit int    `json:"limit"`
		Mode  string `json:"mode"`
	}
	found, err = client.GetJSON(context.Background(), "", "rules", &rules)
	if err != nil {
		t.Fatalf("GetJSON() error = %v", err)
	}
	if !found || rules.Limit != 5 || rules.Mode != "partial" {
		t.Fatalf("GetJSON() = %+v, %t; want decoded rules", rules, found)
	}
}

func TestClient_GetMissingKey(t *testing.T) {
	t.Parallel()

	client := newClient(t, map[string]map[any]any{"params": {"known": "value"}})
	value, err := client.Get(context.Background(), "", "missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if value.Found {
		t.Fatalf("Get().Found = true, want false")
	}
}

func TestClient_ReturnsConversionError(t *testing.T) {
	t.Parallel()

	client := newClient(t, map[string]map[any]any{"params": {"flag": struct{}{}}})
	_, found, err := client.GetBool(context.Background(), "", "flag")
	if err == nil {
		t.Fatal("GetBool() error = nil, want conversion error")
	}
	if !found {
		t.Fatal("GetBool() found = false, want true")
	}
	var conversionErr hazelcast.ValueConversionError
	if !errors.As(err, &conversionErr) {
		t.Fatalf("GetBool() error = %T, want ValueConversionError", err)
	}
}

func TestNew_LoadsFileSource(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "hazelcast.json")
	if err := os.WriteFile(path, []byte(`{"Cluster":{"Name":"file-config"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	client, err := hazelcast.New(context.Background(), hazelcast.Config{
		Source: hazelcast.Source{Kind: hazelcast.SourceFile, Path: path},
	}, hazelcast.WithLogger(discardLogger()), hazelcast.WithStarter(fakeStarter(t, nil)))
	if err != nil {
		t.Fatalf("hazelcast.New() error = %v", err)
	}
	if got := client.HazelcastConfig().Cluster.Name; got != "file-config" {
		t.Fatalf("HazelcastConfig().Cluster.Name = %q, want file-config", got)
	}
}

func TestNew_RejectsInvalidSource(t *testing.T) {
	t.Parallel()

	_, err := hazelcast.New(context.Background(), hazelcast.Config{
		Source: hazelcast.Source{Kind: hazelcast.SourceS3, Bucket: "configs"},
	}, hazelcast.WithLogger(discardLogger()), hazelcast.WithMapProvider(newFakeProvider(nil)))
	if err == nil {
		t.Fatal("hazelcast.New() error = nil, want invalid config")
	}
	var invalid hazelcast.InvalidConfigError
	if !errors.As(err, &invalid) {
		t.Fatalf("hazelcast.New() error = %T, want InvalidConfigError", err)
	}
}

func newClient(t *testing.T, data map[string]map[any]any) *hazelcast.Client {
	t.Helper()
	client, err := hazelcast.New(context.Background(), hazelcast.Config{DefaultMap: "params"},
		hazelcast.WithLogger(discardLogger()),
		hazelcast.WithMapProvider(newFakeProvider(data)),
	)
	if err != nil {
		t.Fatalf("hazelcast.New() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close(context.Background()) })
	return client
}

func fakeStarter(t *testing.T, data map[string]map[any]any) hazelcast.Starter {
	t.Helper()
	return func(context.Context, hz.Config) (hazelcast.MapProvider, error) {
		return newFakeProvider(data), nil
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeProvider struct {
	maps     map[string]*fakeMap
	shutdown bool
}

func newFakeProvider(data map[string]map[any]any) *fakeProvider {
	provider := &fakeProvider{maps: make(map[string]*fakeMap)}
	for name, values := range data {
		copied := make(map[any]any, len(values))
		for key, value := range values {
			copied[key] = value
		}
		provider.maps[name] = &fakeMap{values: copied}
	}
	return provider
}

func (p *fakeProvider) GetMap(_ context.Context, name string) (hazelcast.Map, error) {
	if p.maps == nil {
		p.maps = make(map[string]*fakeMap)
	}
	if _, ok := p.maps[name]; !ok {
		p.maps[name] = &fakeMap{values: make(map[any]any)}
	}
	return p.maps[name], nil
}

func (p *fakeProvider) Shutdown(context.Context) error {
	p.shutdown = true
	return nil
}

type fakeMap struct {
	values map[any]any
}

func (m *fakeMap) ContainsKey(_ context.Context, key interface{}) (bool, error) {
	_, ok := m.values[key]
	return ok, nil
}

func (m *fakeMap) Get(_ context.Context, key interface{}) (interface{}, error) {
	return m.values[key], nil
}
