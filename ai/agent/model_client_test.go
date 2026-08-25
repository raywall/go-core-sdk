// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent tests OpenAI-compatible model client behavior.
//
// This file is part of the Agent bounded context within the AI package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package agent_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/go-core-sdk/ai/agent"
)

func TestOpenAICompatibleClient_Chat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/engines/v1/chat/completions" {
			t.Fatalf("path = %q, want /engines/v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		if request["model"] != "ai/test" {
			t.Fatalf("model = %v, want ai/test", request["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "ai/test",
			"choices": []map[string]any{{
				"message": map[string]any{"role": "assistant", "content": "ok"},
			}},
		})
	}))
	t.Cleanup(server.Close)

	client, err := agent.NewOpenAICompatibleClient(agent.Config{
		BaseURL: server.URL + "/engines/v1",
		APIKey:  "test-key",
		Model:   "ai/test",
	}, server.Client())
	if err != nil {
		t.Fatalf("NewOpenAICompatibleClient() error = %v", err)
	}

	response, err := client.Chat(context.Background(), agent.ChatRequest{
		Model:    "ai/test",
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if response.Message.Content != "ok" {
		t.Fatalf("Chat().Message.Content = %q, want ok", response.Message.Content)
	}
}
