// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy implements downstream tool invocation.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

// Invoke calls the downstream HTTP service mapped to the requested tool.
func (p *Proxy) Invoke(ctx context.Context, input types.InvokeInput) (types.InvokeOutput, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return types.InvokeOutput{}, err
	}
	if p == nil {
		return types.InvokeOutput{}, types.InvalidConfigError{Field: "Proxy", Reason: "is required"}
	}

	toolName := strings.TrimSpace(input.ToolName)
	if toolName == "" {
		return types.InvokeOutput{}, types.InvalidInputError{Field: "ToolName", Reason: "is required"}
	}
	tool, ok := p.tools[toolName]
	if !ok {
		return types.InvokeOutput{}, types.ToolNotFoundError{Name: toolName}
	}

	body, err := encodeArguments(tool.Name, input.Arguments)
	if err != nil {
		p.logger.ErrorContext(ctx, "mcp_proxy_encode_failed", "tool", tool.Name, "error", err)
		return types.InvokeOutput{}, err
	}

	endpoint, err := p.endpointURL(tool.Path)
	if err != nil {
		return types.InvokeOutput{}, types.RequestError{ToolName: tool.Name, Operation: "resolve_url", Err: err}
	}
	request, err := http.NewRequestWithContext(ctx, tool.Method, endpoint, body)
	if err != nil {
		return types.InvokeOutput{}, types.RequestError{ToolName: tool.Name, Operation: "build_request", Err: err}
	}
	applyHeaders(request.Header, p.config.DefaultHeaders)
	applyHeaders(request.Header, tool.Headers)
	applyHeaders(request.Header, input.Headers)
	if input.Arguments != nil && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}

	p.logger.InfoContext(ctx, "mcp_proxy_invoke_started", "tool", tool.Name, "method", tool.Method, "path", tool.Path)
	response, err := p.httpClient.Do(request)
	if err != nil {
		p.logger.ErrorContext(ctx, "mcp_proxy_request_failed", "tool", tool.Name, "error", err)
		return types.InvokeOutput{}, types.RequestError{ToolName: tool.Name, Operation: "execute", Err: err}
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return types.InvokeOutput{}, types.RequestError{ToolName: tool.Name, Operation: "read_response", Err: err}
	}
	output := types.InvokeOutput{
		ToolName:    tool.Name,
		StatusCode:  response.StatusCode,
		Status:      response.Status,
		Headers:     response.Header.Clone(),
		Body:        payload,
		DecodedBody: decodeJSONBody(response.Header.Get("Content-Type"), payload),
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		p.logger.ErrorContext(ctx, "mcp_proxy_backend_failed", "tool", tool.Name, "status_code", response.StatusCode)
		return output, types.BackendError{ToolName: tool.Name, StatusCode: response.StatusCode, Body: boundedBody(payload)}
	}

	p.logger.InfoContext(ctx, "mcp_proxy_invoke_completed", "tool", tool.Name, "status_code", response.StatusCode, "bytes", len(payload))
	return output, nil
}

func encodeArguments(toolName string, arguments any) (io.Reader, error) {
	if arguments == nil {
		return nil, nil
	}
	payload, err := json.Marshal(arguments)
	if err != nil {
		return nil, types.EncodeError{ToolName: toolName, Err: err}
	}
	return bytes.NewReader(payload), nil
}

func (p *Proxy) endpointURL(path string) (string, error) {
	parsed, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	return p.baseURL.ResolveReference(parsed).String(), nil
}

func applyHeaders(headers http.Header, values map[string]string) {
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key != "" {
			headers.Set(key, value)
		}
	}
}

func decodeJSONBody(contentType string, payload []byte) any {
	if len(payload) == 0 || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func boundedBody(payload []byte) string {
	if len(payload) > maxBackendErrorBytes {
		payload = payload[:maxBackendErrorBytes]
	}
	return strings.TrimSpace(string(payload))
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}
