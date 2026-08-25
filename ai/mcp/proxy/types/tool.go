// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy/types defines MCP tool contracts.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "encoding/json"

// Tool describes an HTTP-backed action that can be exposed by an MCP server.
type Tool struct {
	// Name is the stable tool identifier used by the MCP server.
	Name string `json:"name"`
	// Description explains when an LLM agent should use the tool.
	Description string `json:"description"`
	// Method is the HTTP method used to invoke the downstream service.
	Method string `json:"method"`
	// Path is the downstream HTTP path relative to the configured proxy BaseURL.
	Path string `json:"path"`
	// InputSchema is the JSON Schema exposed to MCP clients for tool arguments.
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
	// Headers contains HTTP headers added whenever this tool is invoked.
	Headers map[string]string `json:"headers,omitempty"`
}
