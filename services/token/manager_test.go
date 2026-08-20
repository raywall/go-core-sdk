// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// token tests token manager lifecycle behavior.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package token_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/services/token"
	"github.com/raywall/go-core-sdk/services/token/types"
)

// TestNewManager_ValidatesConfig verifies that required configuration is
// rejected before any STS call can be attempted.
func TestNewManager_ValidatesConfig(t *testing.T) {
	t.Parallel()

	_, err := token.NewManager(token.Config{})
	if err == nil {
		t.Fatal("NewManager() error = nil, want validation error")
	}

	var invalidConfig types.InvalidConfigError
	if !errors.As(err, &invalidConfig) {
		t.Fatalf("NewManager() error type = %T, want types.InvalidConfigError", err)
	}
}

// TestManager_StartFetchesToken verifies the initial STS request and the public
// token snapshot produced by Start.
func TestManager_StartFetchesToken(t *testing.T) {
	t.Parallel()

	requests := make(chan url.Values, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertSTSRequest(t, r, "orders-api") {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requests <- r.PostForm
		writeTokenResponse(w, "token-1", 300, true)
	}))
	t.Cleanup(server.Close)

	manager := newTestManager(t, server.URL, token.Config{
		Endpoint: "/oauth/token",
		Headers:  map[string]string{"X-App": "orders-api"},
	})

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Manager.Start() error = %v", err)
	}
	t.Cleanup(manager.Stop)

	select {
	case form := <-requests:
		if got, want := form.Get("grant_type"), "client_credentials"; got != want {
			t.Fatalf("grant_type = %q, want %q", got, want)
		}
		if got, want := form.Get("client_id"), "client-id"; got != want {
			t.Fatalf("client_id = %q, want %q", got, want)
		}
		if got, want := form.Get("client_secret"), "client-secret"; got != want {
			t.Fatalf("client_secret = %q, want %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for STS request")
	}

	snapshot := manager.Token().Snapshot()
	if got, want := snapshot.AccessToken, "token-1"; got != want {
		t.Fatalf("AccessToken = %q, want %q", got, want)
	}
	if got, want := manager.Token().ToString(), "Bearer token-1"; got != want {
		t.Fatalf("Token.ToString() = %q, want %q", got, want)
	}
	if got, want := manager.Status(), token.StatusRunning; got != want {
		t.Fatalf("Manager.Status() = %q, want %q", got, want)
	}
}

// TestManager_RefreshUpdatesStableTokenPointer verifies that Refresh updates
// the same token pointer rather than replacing it.
func TestManager_RefreshUpdatesStableTokenPointer(t *testing.T) {
	t.Parallel()

	var sequence atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertSTSRequest(t, r, "") {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		id := sequence.Add(1)
		writeTokenResponse(w, "token-"+strconv.FormatInt(id, 10), 300, true)
	}))
	t.Cleanup(server.Close)

	manager := newTestManager(t, server.URL, token.Config{Endpoint: "/oauth/token"})
	stablePointer := manager.Token()

	if err := manager.Refresh(context.Background()); err != nil {
		t.Fatalf("first Manager.Refresh() error = %v", err)
	}
	if err := manager.Refresh(context.Background()); err != nil {
		t.Fatalf("second Manager.Refresh() error = %v", err)
	}

	if stablePointer != manager.Token() {
		t.Fatal("Manager.Token() returned a different pointer after refresh")
	}
	if got, want := stablePointer.AccessToken(), "token-2"; got != want {
		t.Fatalf("AccessToken = %q, want %q", got, want)
	}
}

// TestManager_StopIsIdempotent verifies Stop transitions the manager to stopped
// and can be called more than once.
func TestManager_StopIsIdempotent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertSTSRequest(t, r, "") {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		writeTokenResponse(w, "token-1", 300, true)
	}))
	t.Cleanup(server.Close)

	manager := newTestManager(t, server.URL, token.Config{Endpoint: "/oauth/token"})
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Manager.Start() error = %v", err)
	}

	manager.Stop()
	manager.Stop()

	if got, want := manager.Status(), token.StatusStopped; got != want {
		t.Fatalf("Manager.Status() = %q, want %q", got, want)
	}
}

// TestManager_AutomaticallyRefreshesBeforeExpiry verifies that the background
// loop renews a short-lived token before it expires.
func TestManager_AutomaticallyRefreshesBeforeExpiry(t *testing.T) {
	t.Parallel()

	requests := make(chan int64, 4)
	var sequence atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertSTSRequest(t, r, "") {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		id := sequence.Add(1)
		requests <- id
		writeTokenResponse(w, "token-"+strconv.FormatInt(id, 10), 1, true)
	}))
	t.Cleanup(server.Close)

	manager := newTestManager(t, server.URL, token.Config{
		Endpoint:      "/oauth/token",
		RefreshBefore: 900 * time.Millisecond,
		RetryBackoff:  10 * time.Millisecond,
	})
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Manager.Start() error = %v", err)
	}
	t.Cleanup(manager.Stop)

	expectRequest(t, requests, 1)
	expectRequest(t, requests, 2)

	waitForAccessToken(t, manager, "token-2")
}

// TestManager_KeepsPreviousTokenAfterRefreshFailure verifies degraded behavior:
// the manager reports the error but keeps the last valid token available.
func TestManager_KeepsPreviousTokenAfterRefreshFailure(t *testing.T) {
	t.Parallel()

	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertSTSRequest(t, r, "") {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if fail.Load() {
			http.Error(w, "temporary outage", http.StatusBadGateway)
			return
		}
		writeTokenResponse(w, "token-1", 300, true)
	}))
	t.Cleanup(server.Close)

	manager := newTestManager(t, server.URL, token.Config{
		Endpoint:     "/oauth/token",
		MaxRetries:   1,
		RetryBackoff: time.Millisecond,
	})
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("Manager.Start() error = %v", err)
	}
	t.Cleanup(manager.Stop)

	fail.Store(true)
	if err := manager.Refresh(context.Background()); err == nil {
		t.Fatal("Manager.Refresh() error = nil, want STS failure")
	}

	if got, want := manager.Token().AccessToken(), "token-1"; got != want {
		t.Fatalf("AccessToken = %q, want %q", got, want)
	}
	if got, want := manager.Status(), token.StatusDegraded; got != want {
		t.Fatalf("Manager.Status() = %q, want %q", got, want)
	}
	if manager.LastError() == nil {
		t.Fatal("Manager.LastError() = nil, want refresh error")
	}
}

func newTestManager(t *testing.T, baseURL string, overrides token.Config) *token.Manager {
	t.Helper()

	config := token.Config{
		BaseURL:        baseURL,
		Endpoint:       overrides.Endpoint,
		Headers:        overrides.Headers,
		ValidateSSL:    true,
		ClientID:       "client-id",
		ClientSecret:   "client-secret",
		RefreshBefore:  overrides.RefreshBefore,
		RequestTimeout: time.Second,
		MaxRetries:     overrides.MaxRetries,
		RetryBackoff:   overrides.RetryBackoff,
	}
	if config.Endpoint == "" {
		config.Endpoint = "/oauth/token"
	}

	manager, err := token.NewManager(
		config,
		token.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func assertSTSRequest(t *testing.T, r *http.Request, appHeader string) bool {
	t.Helper()

	if got, want := r.Method, http.MethodPost; got != want {
		t.Errorf("method = %q, want %q", got, want)
		return false
	}
	if got, want := r.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; got != want {
		t.Errorf("content-type = %q, want %q", got, want)
		return false
	}
	if appHeader != "" {
		if got := r.Header.Get("X-App"); got != appHeader {
			t.Errorf("X-App = %q, want %q", got, appHeader)
			return false
		}
	}
	if err := r.ParseForm(); err != nil {
		t.Errorf("ParseForm() error = %v", err)
		return false
	}
	return true
}

func writeTokenResponse(w http.ResponseWriter, accessToken string, expiresIn int64, active bool) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"refresh_token": "refresh-" + accessToken,
		"scope":         "pagamentos.read pedidos.read expedicao.read",
		"active":        active,
	}
	_ = json.NewEncoder(w).Encode(response)
}

func expectRequest(t *testing.T, requests <-chan int64, want int64) {
	t.Helper()

	select {
	case got := <-requests:
		if got != want {
			t.Fatalf("request sequence = %d, want %d", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for request sequence %d", want)
	}
}

func waitForAccessToken(t *testing.T, manager *token.Manager, want string) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if got := manager.Token().AccessToken(); got == want {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for access token %q, got %q", want, manager.Token().AccessToken())
		}
	}
}
