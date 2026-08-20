// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// proxy implements MCP tool metadata listing.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package proxy

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

// Tools returns the configured MCP-friendly tool mappings in stable order.
func (p *Proxy) Tools(ctx context.Context) ([]types.Tool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p == nil {
		return nil, types.InvalidConfigError{Field: "Proxy", Reason: "is required"}
	}

	tools := copyTools(p.config.Tools)
	sort.SliceStable(tools, func(left int, right int) bool {
		return tools[left].Name < tools[right].Name
	})
	p.logger.InfoContext(ctx, "mcp_proxy_tools_listed", "count", len(tools))
	return tools, nil
}

func copyTools(values []types.Tool) []types.Tool {
	if len(values) == 0 {
		return nil
	}
	copied := make([]types.Tool, len(values))
	for index, tool := range values {
		copied[index] = copyTool(tool)
	}
	return copied
}

func copyTool(tool types.Tool) types.Tool {
	tool.InputSchema = append(json.RawMessage(nil), tool.InputSchema...)
	tool.Headers = copyStringMap(tool.Headers)
	return tool
}

func jsonSchemaLooksValid(schema json.RawMessage) bool {
	if len(schema) == 0 {
		return true
	}
	return json.Valid(schema)
}
