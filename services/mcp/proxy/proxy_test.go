// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package proxy_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/raywall/go-core-sdk/services/mcp/proxy"
	"github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

func TestProxy_ToolsReturnsStableCopies(t *testing.T) {
	t.Parallel()

	service := newProxy(t, "http://example.com", []types.Tool{
		{Name: "simulate_payment", Description: "Simulate a payment.", Method: http.MethodPost, Path: "/payments/simulate"},
		{Name: "get_status", Description: "Get status.", Method: http.MethodPost, Path: "/status"},
	})

	tools, err := service.Tools(context.Background())
	if err != nil {
		t.Fatalf("Tools() error = %v", err)
	}
	if got := []string{tools[0].Name, tools[1].Name}; !reflect.DeepEqual(got, []string{"get_status", "simulate_payment"}) {
		t.Fatalf("tool names = %v", got)
	}

	tools[0].Name = "mutated"
	toolsAgain, err := service.Tools(context.Background())
	if err != nil {
		t.Fatalf("Tools() second call error = %v", err)
	}
	if toolsAgain[0].Name != "get_status" {
		t.Fatalf("tools must be copied, got %q", toolsAgain[0].Name)
	}
}

func TestProxy_InvokeSendsJSONAndMergesHeaders(t *testing.T) {
	t.Parallel()

	var captured invokeCapture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.defaultHeader = r.Header.Get("X-Default")
		captured.toolHeader = r.Header.Get("X-Tool")
		captured.callHeader = r.Header.Get("X-Call")
		captured.overrideHeader = r.Header.Get("X-Override")
		captured.contentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		captured.body = string(body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"approved": true, "amount": 120000})
	}))
	t.Cleanup(server.Close)

	service := newProxy(t, server.URL, []types.Tool{
		{
			Name:        "simulate_payment",
			Description: "Simulate whether a payment can be created.",
			Method:      http.MethodPost,
			Path:        "/payments/simulate",
			Headers:     map[string]string{"X-Tool": "tool", "X-Override": "tool"},
		},
	}, proxyConfigOption(func(config *proxy.Config) {
		config.DefaultHeaders = map[string]string{"X-Default": "default", "X-Override": "default"}
	}))

	output, err := service.Invoke(context.Background(), types.InvokeInput{
		ToolName:  "simulate_payment",
		Arguments: map[string]any{"financingId": "fin-123", "amount": 120000},
		Headers:   map[string]string{"X-Call": "call", "X-Override": "call"},
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	if captured.method != http.MethodPost || captured.path != "/payments/simulate" {
		t.Fatalf("request = %s %s", captured.method, captured.path)
	}
	if captured.defaultHeader != "default" || captured.toolHeader != "tool" || captured.callHeader != "call" || captured.overrideHeader != "call" {
		t.Fatalf("headers = %+v", captured)
	}
	if captured.contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", captured.contentType)
	}
	var sent map[string]any
	if err := json.Unmarshal([]byte(captured.body), &sent); err != nil {
		t.Fatalf("request JSON error = %v", err)
	}
	if sent["financingId"] != "fin-123" {
		t.Fatalf("request body = %v", sent)
	}
	if output.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want 200", output.StatusCode)
	}
	decoded, ok := output.DecodedBody.(map[string]any)
	if !ok || decoded["approved"] != true {
		t.Fatalf("DecodedBody = %#v", output.DecodedBody)
	}
}

func TestProxy_InvokeReturnsToolNotFound(t *testing.T) {
	t.Parallel()

	service := newProxy(t, "http://example.com", []types.Tool{{Name: "known", Method: http.MethodPost, Path: "/known"}})

	_, err := service.Invoke(context.Background(), types.InvokeInput{ToolName: "missing"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
	var notFound types.ToolNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Invoke() error type = %T, want ToolNotFoundError", err)
	}
}

func TestProxy_InvokeReturnsBackendError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "downstream failed", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	service := newProxy(t, server.URL, []types.Tool{{Name: "fail", Method: http.MethodPost, Path: "/fail"}})

	output, err := service.Invoke(context.Background(), types.InvokeInput{ToolName: "fail"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
	var backendErr types.BackendError
	if !errors.As(err, &backendErr) {
		t.Fatalf("Invoke() error type = %T, want BackendError", err)
	}
	if output.StatusCode != http.StatusBadGateway || backendErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = output %d error %d", output.StatusCode, backendErr.StatusCode)
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config proxy.Config
	}{
		{name: "missing base url", config: proxy.Config{}},
		{name: "relative base url", config: proxy.Config{BaseURL: "/relative"}},
		{name: "duplicate tools", config: proxy.Config{BaseURL: "http://example.com", Tools: []types.Tool{
			{Name: "same", Method: http.MethodPost, Path: "/one"},
			{Name: "same", Method: http.MethodPost, Path: "/two"},
		}}},
		{name: "invalid schema", config: proxy.Config{BaseURL: "http://example.com", Tools: []types.Tool{
			{Name: "tool", Method: http.MethodPost, Path: "/tool", InputSchema: json.RawMessage(`{`)},
		}}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := proxy.New(test.config, proxy.WithLogger(discardLogger()))
			if err == nil {
				t.Fatal("New() error = nil, want error")
			}
			var invalid types.InvalidConfigError
			if !errors.As(err, &invalid) {
				t.Fatalf("New() error type = %T, want InvalidConfigError", err)
			}
		})
	}
}

func TestProxy_InvokeRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	service := newProxy(t, "http://example.com", []types.Tool{{Name: "known", Method: http.MethodPost, Path: "/known"}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.Invoke(ctx, types.InvokeInput{ToolName: "known"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Invoke() error = %v, want context.Canceled", err)
	}
}

type invokeCapture struct {
	method         string
	path           string
	defaultHeader  string
	toolHeader     string
	callHeader     string
	overrideHeader string
	contentType    string
	body           string
}

type proxyConfigOption func(*proxy.Config)

func newProxy(t *testing.T, baseURL string, tools []types.Tool, configOptions ...proxyConfigOption) *proxy.Proxy {
	t.Helper()
	config := proxy.Config{
		BaseURL: baseURL,
		Tools:   tools,
	}
	for _, option := range configOptions {
		option(&config)
	}
	service, err := proxy.New(config, proxy.WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("proxy.New() error = %v", err)
	}
	return service
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
