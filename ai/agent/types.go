// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent implements public contracts for agent inference.
//
// This file is part of the Agent bounded context within the AI package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package agent

import (
	"context"
	"encoding/json"
	"time"
)

const (
	// RoleSystem identifies a system instruction message.
	RoleSystem = "system"
	// RoleUser identifies a user message.
	RoleUser = "user"
	// RoleAssistant identifies an assistant message.
	RoleAssistant = "assistant"
	// RoleTool identifies a tool result message.
	RoleTool = "tool"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultMaxIterations  = 4
)

// Config controls agent construction and model inference.
type Config struct {
	// BaseURL is the OpenAI-compatible API base URL, such as
	// http://localhost:12434/engines/v1.
	BaseURL string
	// APIKey optionally configures the Authorization bearer token.
	APIKey string
	// Model is the chat model identifier sent to the API.
	Model string
	// SystemPrompt is prepended to each inference as a system message.
	SystemPrompt string
	// Temperature controls model creativity when set.
	Temperature *float64
	// MaxTokens optionally limits generated tokens.
	MaxTokens int
	// RequestTimeout controls outbound model API calls.
	RequestTimeout time.Duration
	// MaxIterations limits tool-calling loops for one inference.
	MaxIterations int
}

// Message is a chat message exchanged with the model.
type Message struct {
	// Role is one of system, user, assistant or tool.
	Role string `json:"role"`
	// Content is the textual message content.
	Content string `json:"content,omitempty"`
	// Name optionally identifies a tool or participant.
	Name string `json:"name,omitempty"`
	// ToolCallID links a tool result to the model-requested tool call.
	ToolCallID string `json:"tool_call_id,omitempty"`
	// ToolCalls contains tool calls requested by the assistant.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a model-requested tool invocation.
type ToolCall struct {
	// ID is the provider-assigned tool call identifier.
	ID string `json:"id,omitempty"`
	// Type is usually function.
	Type string `json:"type,omitempty"`
	// Function contains the function-call payload.
	Function FunctionCall `json:"function"`
}

// FunctionCall contains a function name and JSON arguments.
type FunctionCall struct {
	// Name identifies the registered tool.
	Name string `json:"name"`
	// Arguments contains raw JSON arguments supplied by the model.
	Arguments json.RawMessage `json:"arguments"`
}

// UnmarshalJSON decodes OpenAI-compatible function-call arguments.
//
// Providers usually encode arguments as a JSON string containing an object.
// This method normalizes both string and raw-object forms into Arguments.
func (c *FunctionCall) UnmarshalJSON(data []byte) error {
	var raw struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Name = raw.Name
	if len(raw.Arguments) == 0 {
		c.Arguments = nil
		return nil
	}
	var asString string
	if err := json.Unmarshal(raw.Arguments, &asString); err == nil {
		c.Arguments = json.RawMessage(asString)
		return nil
	}
	c.Arguments = append(json.RawMessage(nil), raw.Arguments...)
	return nil
}

// Tool defines a function exposed to the model.
type Tool struct {
	// Name is the function name sent to the model.
	Name string
	// Description explains when the model should call the tool.
	Description string
	// Parameters is a JSON Schema object describing accepted arguments.
	Parameters json.RawMessage
	// Handler executes the tool with decoded JSON arguments.
	Handler ToolHandler
}

// ToolHandler executes a tool call.
type ToolHandler func(context.Context, json.RawMessage) (any, error)

// InferenceInput describes one agent inference.
type InferenceInput struct {
	// Prompt is the user prompt for this inference.
	Prompt string
	// Messages optionally adds extra conversation messages before Prompt.
	Messages []Message
}

// InferenceResult contains the final model response and execution trace.
type InferenceResult struct {
	// Output is the final assistant text.
	Output string
	// Messages contains all messages sent or produced during inference.
	Messages []Message
	// ToolResults contains tools executed during inference.
	ToolResults []ToolResult
	// Model is the model used for inference.
	Model string
	// Iterations is the number of model calls performed.
	Iterations int
}

// ToolResult contains a completed tool execution.
type ToolResult struct {
	// Call is the model-requested tool call.
	Call ToolCall
	// Output contains the JSON-encoded tool result.
	Output string
	// Error contains the tool error text when execution failed.
	Error string
}

// ChatRequest is the provider-neutral model request.
type ChatRequest struct {
	// Model is the chat model identifier.
	Model string
	// Messages contains chat history and current prompt.
	Messages []Message
	// Tools contains available tools.
	Tools []Tool
	// Temperature controls model creativity when set.
	Temperature *float64
	// MaxTokens optionally limits generated tokens.
	MaxTokens int
}

// ChatResponse is the provider-neutral model response.
type ChatResponse struct {
	// Message is the assistant message returned by the model.
	Message Message
	// Model is the model reported by the provider.
	Model string
}

// ModelClient defines a chat completion provider.
type ModelClient interface {
	// Chat sends a chat completion request to the model provider.
	Chat(context.Context, ChatRequest) (ChatResponse, error)
}

// Memory stores messages that should influence future inferences.
type Memory interface {
	// Load returns stored messages in replay order.
	Load(context.Context) ([]Message, error)
	// Append stores one message.
	Append(context.Context, Message) error
	// Clear removes all stored messages.
	Clear(context.Context) error
}
