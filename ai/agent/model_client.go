// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent implements OpenAI-compatible model clients.
//
// This file is part of the Agent bounded context within the AI package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// OpenAICompatibleClient calls an OpenAI-compatible chat completions endpoint.
type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAICompatibleClient constructs a model client for OpenAI-compatible APIs.
func NewOpenAICompatibleClient(config Config, httpClient *http.Client) (*OpenAICompatibleClient, error) {
	normalized := normalizeConfig(config)
	if normalized.BaseURL == "" {
		return nil, InvalidConfigError{Field: "BaseURL", Reason: "is required"}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: normalized.RequestTimeout}
	}
	return &OpenAICompatibleClient{
		baseURL:    strings.TrimRight(normalized.BaseURL, "/"),
		apiKey:     normalized.APIKey,
		httpClient: httpClient,
	}, nil
}

// Chat sends a chat completion request.
func (c *OpenAICompatibleClient) Chat(ctx context.Context, input ChatRequest) (ChatResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request := openAIChatRequest{
		Model:       input.Model,
		Messages:    input.Messages,
		Temperature: input.Temperature,
		MaxTokens:   optionalInt(input.MaxTokens),
		Tools:       openAITools(input.Tools),
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return ChatResponse{}, ModelError{Operation: "encode_request", Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, ModelError{Operation: "build_request", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.apiKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ChatResponse{}, ModelError{Operation: "execute_request", Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, ModelError{Operation: "read_response", Err: err}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ChatResponse{}, ModelError{Operation: "chat_completion", StatusCode: resp.StatusCode, Body: boundedBody(body)}
	}

	var output openAIChatResponse
	if err := json.Unmarshal(body, &output); err != nil {
		return ChatResponse{}, ModelError{Operation: "decode_response", Err: err}
	}
	if len(output.Choices) == 0 {
		return ChatResponse{}, ModelError{Operation: "decode_response", Err: InvalidConfigError{Field: "choices", Reason: "is empty"}}
	}
	return ChatResponse{
		Message: output.Choices[0].Message,
		Model:   output.Model,
	}, nil
}

type openAIChatRequest struct {
	Model       string       `json:"model"`
	Messages    []Message    `json:"messages"`
	Tools       []openAITool `json:"tools,omitempty"`
	Temperature *float64     `json:"temperature,omitempty"`
	MaxTokens   *int         `json:"max_tokens,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIFunctionTool `json:"function"`
}

type openAIFunctionTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func openAITools(tools []Tool) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	converted := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		converted = append(converted, openAITool{
			Type: "function",
			Function: openAIFunctionTool{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	return converted
}

func normalizeConfig(config Config) Config {
	normalized := config
	normalized.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	normalized.APIKey = strings.TrimSpace(config.APIKey)
	normalized.Model = strings.TrimSpace(config.Model)
	normalized.SystemPrompt = strings.TrimSpace(config.SystemPrompt)
	if normalized.RequestTimeout == 0 {
		normalized.RequestTimeout = defaultRequestTimeout
	}
	if normalized.MaxIterations == 0 {
		normalized.MaxIterations = defaultMaxIterations
	}
	return normalized
}

func validateConfig(config Config) error {
	if config.Model == "" {
		return InvalidConfigError{Field: "Model", Reason: "is required"}
	}
	if config.RequestTimeout < 0 {
		return InvalidConfigError{Field: "RequestTimeout", Reason: "must not be negative"}
	}
	if config.MaxIterations < 0 {
		return InvalidConfigError{Field: "MaxIterations", Reason: "must not be negative"}
	}
	return nil
}

func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func boundedBody(body []byte) string {
	const limit = 4096
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit])
}
