// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent implements inference orchestration.
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
	"fmt"
	"log/slog"
	"strings"
)

// Agent orchestrates prompts, model calls, tools and memory.
//
// Agent is safe for concurrent use when the configured model client, memories
// and tools are safe for concurrent use.
type Agent struct {
	config          Config
	logger          *slog.Logger
	modelClient     ModelClient
	shortTermMemory Memory
	longTermMemory  Memory
	tools           []Tool
	toolsByName     map[string]Tool
}

// New constructs an Agent from Config.
func New(config Config, configurers ...Option) (*Agent, error) {
	normalized := normalizeConfig(config)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}

	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}
	if err := validateTools(options.tools); err != nil {
		return nil, err
	}

	modelClient := options.modelClient
	if modelClient == nil {
		client, err := NewOpenAICompatibleClient(normalized, options.httpClient)
		if err != nil {
			return nil, err
		}
		modelClient = client
	}

	tools := normalizeTools(options.tools)
	toolsByName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		toolsByName[tool.Name] = tool
	}

	return &Agent{
		config:          normalized,
		logger:          options.logger,
		modelClient:     modelClient,
		shortTermMemory: options.shortTermMemory,
		longTermMemory:  options.longTermMemory,
		tools:           tools,
		toolsByName:     toolsByName,
	}, nil
}

// Config returns a copy of the normalized agent configuration.
func (a *Agent) Config() Config {
	if a == nil {
		return Config{}
	}
	return a.config
}

// Infer runs one inference against the configured model.
func (a *Agent) Infer(ctx context.Context, input InferenceInput) (InferenceResult, error) {
	if a == nil || a.modelClient == nil {
		return InferenceResult{}, InvalidConfigError{Field: "Agent", Reason: "is required"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	userMessage := Message{Role: RoleUser, Content: strings.TrimSpace(input.Prompt)}
	if userMessage.Content == "" && len(input.Messages) == 0 {
		return InferenceResult{}, InvalidConfigError{Field: "Prompt", Reason: "is required"}
	}

	messages, err := a.buildMessages(ctx, input)
	if err != nil {
		return InferenceResult{}, err
	}
	if userMessage.Content != "" {
		messages = append(messages, userMessage)
		if err := a.appendMemory(ctx, userMessage); err != nil {
			return InferenceResult{}, err
		}
	}

	result := InferenceResult{
		Messages: copyMessages(messages),
		Model:    a.config.Model,
	}

	for iteration := 1; iteration <= a.config.MaxIterations; iteration++ {
		a.logger.InfoContext(ctx, "agent_inference_iteration_started", "model", a.config.Model, "iteration", iteration, "tools", len(a.tools))
		response, err := a.modelClient.Chat(ctx, ChatRequest{
			Model:       a.config.Model,
			Messages:    messages,
			Tools:       a.tools,
			Temperature: a.config.Temperature,
			MaxTokens:   a.config.MaxTokens,
		})
		if err != nil {
			a.logger.ErrorContext(ctx, "agent_inference_iteration_failed", "model", a.config.Model, "iteration", iteration, "error", err)
			return InferenceResult{}, err
		}
		assistantMessage := response.Message
		if assistantMessage.Role == "" {
			assistantMessage.Role = RoleAssistant
		}
		messages = append(messages, assistantMessage)
		result.Messages = append(result.Messages, cloneMessage(assistantMessage))
		result.Iterations = iteration
		if response.Model != "" {
			result.Model = response.Model
		}

		if len(assistantMessage.ToolCalls) == 0 {
			result.Output = assistantMessage.Content
			if err := a.appendMemory(ctx, assistantMessage); err != nil {
				return InferenceResult{}, err
			}
			a.logger.InfoContext(ctx, "agent_inference_completed", "model", result.Model, "iterations", result.Iterations, "tools", len(result.ToolResults))
			return result, nil
		}

		toolMessages, toolResults := a.executeToolCalls(ctx, assistantMessage.ToolCalls)
		for _, toolMessage := range toolMessages {
			if err := a.appendMemory(ctx, toolMessage); err != nil {
				return InferenceResult{}, err
			}
		}
		messages = append(messages, toolMessages...)
		result.Messages = append(result.Messages, toolMessages...)
		result.ToolResults = append(result.ToolResults, toolResults...)
	}

	return InferenceResult{}, ModelError{Operation: "max_iterations", Err: InvalidConfigError{Field: "MaxIterations", Reason: "exceeded before final assistant response"}}
}

func (a *Agent) buildMessages(ctx context.Context, input InferenceInput) ([]Message, error) {
	messages := make([]Message, 0, 1+len(input.Messages)+1)
	if a.config.SystemPrompt != "" {
		messages = append(messages, Message{Role: RoleSystem, Content: a.config.SystemPrompt})
	}
	if a.longTermMemory != nil {
		stored, err := a.longTermMemory.Load(ctx)
		if err != nil {
			return nil, err
		}
		messages = append(messages, stored...)
	}
	if a.shortTermMemory != nil {
		stored, err := a.shortTermMemory.Load(ctx)
		if err != nil {
			return nil, err
		}
		messages = append(messages, stored...)
	}
	messages = append(messages, copyMessages(input.Messages)...)
	return messages, nil
}

func (a *Agent) appendMemory(ctx context.Context, message Message) error {
	if a.shortTermMemory != nil {
		if err := a.shortTermMemory.Append(ctx, message); err != nil {
			return err
		}
	}
	if a.longTermMemory != nil {
		if err := a.longTermMemory.Append(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) executeToolCalls(ctx context.Context, calls []ToolCall) ([]Message, []ToolResult) {
	messages := make([]Message, 0, len(calls))
	results := make([]ToolResult, 0, len(calls))
	for _, call := range calls {
		name := strings.TrimSpace(call.Function.Name)
		tool, ok := a.toolsByName[name]
		result := ToolResult{Call: call}
		if !ok {
			result.Error = fmt.Sprintf("tool %q not found", name)
		} else {
			output, err := tool.Handler(ctx, call.Function.Arguments)
			if err != nil {
				result.Error = ToolError{Name: name, Err: err}.Error()
			} else {
				data, err := json.Marshal(output)
				if err != nil {
					result.Error = ToolError{Name: name, Err: err}.Error()
				} else {
					result.Output = string(data)
				}
			}
		}
		content := result.Output
		if result.Error != "" {
			content = `{"error":` + strconvQuote(result.Error) + `}`
		}
		messages = append(messages, Message{
			Role:       RoleTool,
			Content:    content,
			Name:       name,
			ToolCallID: call.ID,
		})
		results = append(results, result)
		a.logger.InfoContext(ctx, "agent_tool_executed", "tool", name, "error", result.Error != "")
	}
	return messages, results
}

func validateTools(tools []Tool) error {
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			return InvalidConfigError{Field: "Tool.Name", Reason: "is required"}
		}
		if tool.Handler == nil {
			return InvalidConfigError{Field: "Tool.Handler", Reason: "is required"}
		}
		if _, ok := seen[name]; ok {
			return InvalidConfigError{Field: "Tool.Name", Reason: "must be unique"}
		}
		seen[name] = struct{}{}
	}
	return nil
}

func normalizeTools(tools []Tool) []Tool {
	normalized := make([]Tool, len(tools))
	for i, tool := range tools {
		normalized[i] = tool
		normalized[i].Name = strings.TrimSpace(tool.Name)
	}
	return normalized
}

func strconvQuote(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return string(data)
}
