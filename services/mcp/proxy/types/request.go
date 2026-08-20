// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy/types defines invocation request and response contracts.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types

import "net/http"

// InvokeInput describes one MCP tool invocation.
type InvokeInput struct {
	// ToolName identifies the registered tool to invoke.
	ToolName string
	// Arguments is encoded as JSON and sent to the downstream HTTP service.
	Arguments any
	// Headers contains per-call HTTP headers. These override default and
	// tool-level headers with the same name.
	Headers map[string]string
}

// InvokeOutput contains the downstream service response.
type InvokeOutput struct {
	// ToolName identifies the invoked tool.
	ToolName string
	// StatusCode is the downstream HTTP status code.
	StatusCode int
	// Status is the downstream HTTP status text.
	Status string
	// Headers contains downstream response headers.
	Headers http.Header
	// Body contains the raw downstream response body.
	Body []byte
	// DecodedBody contains a JSON-decoded representation when the response body
	// is valid JSON. Non-JSON responses keep this field nil.
	DecodedBody any
}
