// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/agent implements an agent prototyping sample.
//
// This file is part of the Agent sample bounded context within the Samples service.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/raywall/go-core-sdk/ai/agent"
)

func main() {
	if err := run(context.Background(), os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, out io.Writer) error {
	modelAPI := newOpenAICompatibleModelAPI()
	defer modelAPI.Close()

	shortTerm := agent.NewShortTermMemory(8)
	longTerm := agent.NewLongTermMemory()
	if err := longTerm.Append(ctx, agent.Message{
		Role:    agent.RoleSystem,
		Content: "Known business fact: student financing payments may be partial.",
	}); err != nil {
		return err
	}

	service, err := agent.New(agent.Config{
		// For Docker Model Runner, use: http://localhost:12434/engines/v1.
		BaseURL:       modelAPI.URL + "/engines/v1",
		Model:         "ai/sample-agent",
		SystemPrompt:  "You are a payment testing assistant. Use tools when useful and answer briefly.",
		MaxIterations: 4,
	}, agent.WithShortTermMemory(shortTerm), agent.WithLongTermMemory(longTerm), agent.WithTools(paymentCapacityTool()))
	if err != nil {
		return err
	}

	return PaymentAssistantUseCase{
		Agent:  service,
		Output: out,
	}.Execute(ctx, "Can the worker pay 1200 using the available amount?")
}

// ConversationalAgent defines the application port used by the use case.
type ConversationalAgent interface {
	Infer(context.Context, agent.InferenceInput) (agent.InferenceResult, error)
}

// PaymentAssistantUseCase demonstrates using an agent behind an application port.
type PaymentAssistantUseCase struct {
	Agent  ConversationalAgent
	Output io.Writer
}

// Execute asks the agent to reason with tools and writes the final answer.
func (u PaymentAssistantUseCase) Execute(ctx context.Context, prompt string) error {
	result, err := u.Agent.Infer(ctx, agent.InferenceInput{Prompt: prompt})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(u.Output, "answer=%s tools=%d iterations=%d\n", result.Output, len(result.ToolResults), result.Iterations)
	return err
}

func paymentCapacityTool() agent.Tool {
	return agent.Tool{
		Name:        "payment_capacity",
		Description: "Calculates whether an available amount can cover a requested payment.",
		Parameters: json.RawMessage(`{
			"type":"object",
			"properties":{
				"availableAmount":{"type":"integer"},
				"requestedAmount":{"type":"integer"}
			},
			"required":["availableAmount","requestedAmount"]
		}`),
		Handler: func(_ context.Context, args json.RawMessage) (any, error) {
			var input struct {
				AvailableAmount int `json:"availableAmount"`
				RequestedAmount int `json:"requestedAmount"`
			}
			if err := json.Unmarshal(args, &input); err != nil {
				return nil, err
			}
			return map[string]any{
				"allowed":         input.AvailableAmount >= input.RequestedAmount,
				"availableAmount": input.AvailableAmount,
				"requestedAmount": input.RequestedAmount,
				"remainingAmount": input.AvailableAmount - input.RequestedAmount,
			}, nil
		},
	}
}

func newOpenAICompatibleModelAPI() *httptest.Server {
	requests := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/engines/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model": "ai/sample-agent",
				"choices": []map[string]any{{
					"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id":   "call-1",
							"type": "function",
							"function": map[string]any{
								"name":      "payment_capacity",
								"arguments": `{"availableAmount":1500,"requestedAmount":1200}`,
							},
						}},
					},
				}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "ai/sample-agent",
			"choices": []map[string]any{{
				"message": map[string]any{
					"role":    "assistant",
					"content": "yes, the payment can be covered with 300 remaining",
				},
			}},
		})
	}))
}
