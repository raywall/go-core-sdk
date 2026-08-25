// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent tests inference orchestration.
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
	"io"
	"log/slog"
	"testing"

	"github.com/raywall/go-core-sdk/ai/agent"
)

func TestAgent_InferReturnsAssistantOutputAndStoresMemory(t *testing.T) {
	t.Parallel()

	shortTerm := agent.NewShortTermMemory(10)
	model := &fakeModelClient{
		responses: []agent.ChatResponse{{
			Model:   "local-model",
			Message: agent.Message{Role: agent.RoleAssistant, Content: "answer"},
		}},
	}
	service, err := agent.New(agent.Config{
		Model:        "local-model",
		SystemPrompt: "You are concise.",
	}, agent.WithLogger(discardLogger()), agent.WithModelClient(model), agent.WithShortTermMemory(shortTerm))
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}

	result, err := service.Infer(context.Background(), agent.InferenceInput{Prompt: "hello"})
	if err != nil {
		t.Fatalf("Infer() error = %v", err)
	}
	if result.Output != "answer" {
		t.Fatalf("Infer().Output = %q, want answer", result.Output)
	}
	if len(model.requests) != 1 {
		t.Fatalf("model requests = %d, want 1", len(model.requests))
	}
	if got := model.requests[0].Messages[0]; got.Role != agent.RoleSystem || got.Content != "You are concise." {
		t.Fatalf("first message = %+v, want system prompt", got)
	}

	stored, err := shortTerm.Load(context.Background())
	if err != nil {
		t.Fatalf("shortTerm.Load() error = %v", err)
	}
	if len(stored) != 2 || stored[0].Role != agent.RoleUser || stored[1].Role != agent.RoleAssistant {
		t.Fatalf("stored messages = %+v, want user and assistant", stored)
	}
}

func TestAgent_InferExecutesToolCalls(t *testing.T) {
	t.Parallel()

	model := &fakeModelClient{
		responses: []agent.ChatResponse{
			{
				Model: "local-model",
				Message: agent.Message{Role: agent.RoleAssistant, ToolCalls: []agent.ToolCall{{
					ID:   "call-1",
					Type: "function",
					Function: agent.FunctionCall{
						Name:      "lookup_balance",
						Arguments: json.RawMessage(`{"customerId":"c-1"}`),
					},
				}}},
			},
			{
				Model:   "local-model",
				Message: agent.Message{Role: agent.RoleAssistant, Content: "balance is 1200"},
			},
		},
	}
	service, err := agent.New(agent.Config{Model: "local-model"},
		agent.WithLogger(discardLogger()),
		agent.WithModelClient(model),
		agent.WithTools(agent.Tool{
			Name:        "lookup_balance",
			Description: "Looks up a customer balance.",
			Parameters:  json.RawMessage(`{"type":"object"}`),
			Handler: func(_ context.Context, args json.RawMessage) (any, error) {
				var input struct {
					CustomerID string `json:"customerId"`
				}
				if err := json.Unmarshal(args, &input); err != nil {
					return nil, err
				}
				return map[string]any{"customerId": input.CustomerID, "balance": 1200}, nil
			},
		}),
	)
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}

	result, err := service.Infer(context.Background(), agent.InferenceInput{Prompt: "what is the balance?"})
	if err != nil {
		t.Fatalf("Infer() error = %v", err)
	}
	if result.Output != "balance is 1200" {
		t.Fatalf("Infer().Output = %q, want balance is 1200", result.Output)
	}
	if result.Iterations != 2 {
		t.Fatalf("Infer().Iterations = %d, want 2", result.Iterations)
	}
	if len(result.ToolResults) != 1 || result.ToolResults[0].Error != "" {
		t.Fatalf("ToolResults = %+v, want successful tool result", result.ToolResults)
	}
	if len(model.requests) != 2 {
		t.Fatalf("model requests = %d, want 2", len(model.requests))
	}
	lastRequest := model.requests[1]
	if got := lastRequest.Messages[len(lastRequest.Messages)-1]; got.Role != agent.RoleTool || got.ToolCallID != "call-1" {
		t.Fatalf("last request message = %+v, want tool response", got)
	}
}

func TestFunctionCall_UnmarshalJSONStringArguments(t *testing.T) {
	t.Parallel()

	var call agent.FunctionCall
	if err := json.Unmarshal([]byte(`{"name":"tool","arguments":"{\"id\":\"123\"}"}`), &call); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(call.Arguments) != `{"id":"123"}` {
		t.Fatalf("Arguments = %s, want object JSON", call.Arguments)
	}
}

func TestMemory_LimitsShortTermMessages(t *testing.T) {
	t.Parallel()

	memory := agent.NewShortTermMemory(2)
	for _, content := range []string{"one", "two", "three"} {
		if err := memory.Append(context.Background(), agent.Message{Role: agent.RoleUser, Content: content}); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}
	messages, err := memory.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(messages) != 2 || messages[0].Content != "two" || messages[1].Content != "three" {
		t.Fatalf("Load() = %+v, want last two messages", messages)
	}
}

type fakeModelClient struct {
	requests  []agent.ChatRequest
	responses []agent.ChatResponse
}

func (c *fakeModelClient) Chat(_ context.Context, request agent.ChatRequest) (agent.ChatResponse, error) {
	c.requests = append(c.requests, request)
	if len(c.responses) == 0 {
		return agent.ChatResponse{Message: agent.Message{Role: agent.RoleAssistant, Content: "default"}}, nil
	}
	response := c.responses[0]
	c.responses = c.responses[1:]
	return response, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
