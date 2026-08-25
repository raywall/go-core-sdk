// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// rest tests the public REST consumer behavior.
//
// This file is part of the Consumer bounded context within the Consumer service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/go-core-sdk/services/consumer/rest"
)

func TestClient_RESTWithTokenAndJSONBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer abc" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer abc")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want %q", got, "application/json")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if body["id"] != "123" {
			t.Fatalf("request body id = %q, want %q", body["id"], "123")
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	client, err := rest.New(rest.Config{},
		rest.WithLogger(discardLogger()),
		rest.WithTokenProvider(fakeTokenProvider{token: fakeToken("Bearer abc")}),
	)
	if err != nil {
		t.Fatalf("rest.New() error = %v", err)
	}

	response, err := client.REST(http.MethodPost, server.URL).
		WithBody(map[string]string{"id": "123"}).
		WithToken().
		Do(context.Background())
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	var decoded struct {
		OK bool `json:"ok"`
	}
	if err := response.DecodeJSON(&decoded); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if !decoded.OK {
		t.Fatal("decoded OK = false, want true")
	}
}

func TestClient_RESTWithTokenWithoutProviderReturnsTypedError(t *testing.T) {
	t.Parallel()

	client, err := rest.New(rest.Config{}, rest.WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("rest.New() error = %v", err)
	}

	_, err = client.REST(http.MethodGet, "https://example.com").WithToken().Do(context.Background())
	var tokenErr rest.TokenRequiredError
	if !errors.As(err, &tokenErr) {
		t.Fatalf("err = %T, want TokenRequiredError", err)
	}
}

type fakeToken string

func (t fakeToken) ToString() string {
	return string(t)
}

type fakeTokenProvider struct {
	token fakeToken
}

func (p fakeTokenProvider) Token() rest.AuthorizationToken {
	return p.token
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
