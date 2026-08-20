// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// samples/mcp_proxy demonstrates HTTP service exposure through MCP proxy tools.
//
// This file is part of the MCP Proxy bounded context within the MCP Proxy
// service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

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

	"github.com/raywall/go-core-sdk/services/mcp/proxy"
	proxytypes "github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

func main() {
	ctx := context.Background()
	backend := newPaymentBackend()
	defer backend.Close()

	mcpProxy, err := proxy.New(proxy.Config{
		BaseURL: backend.URL,
		DefaultHeaders: map[string]string{
			"X-App": "mcp-proxy-sample",
		},
		Tools: []proxytypes.Tool{
			{
				Name:        "simulate_student_financing_payment",
				Description: "Simulate payment creation for a student financing. Use when an agent needs to know which installments can be paid before creating a payment event.",
				Method:      http.MethodPost,
				Path:        "/payments/simulate",
				InputSchema: json.RawMessage(`{
					"type": "object",
					"required": ["financingId", "availableAmount"],
					"properties": {
						"financingId": {"type": "string"},
						"availableAmount": {"type": "integer", "minimum": 1}
					}
				}`),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := run(ctx, os.Stdout, mcpProxy); err != nil {
		log.Fatal(err)
	}
}

// ProxyUseCase lists and invokes MCP proxy tools.
type ProxyUseCase struct {
	Proxy  *proxy.Proxy
	Output io.Writer
}

func run(ctx context.Context, out io.Writer, mcpProxy *proxy.Proxy) error {
	return ProxyUseCase{Proxy: mcpProxy, Output: out}.Execute(ctx)
}

func (u ProxyUseCase) Execute(ctx context.Context) error {
	mcpProxy := u.Proxy
	tools, err := mcpProxy.Tools(ctx)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(u.Output, "tools=%d first=%s\n", len(tools), tools[0].Name); err != nil {
		return err
	}

	output, err := mcpProxy.Invoke(ctx, proxytypes.InvokeInput{
		ToolName: "simulate_student_financing_payment",
		Arguments: map[string]any{
			"financingId":     "fin-123",
			"availableAmount": 120000,
		},
		Headers: map[string]string{
			"X-Agent": "local-sample",
		},
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(u.Output, "status=%d body=%s\n", output.StatusCode, string(output.Body))
	return err
}

func newPaymentBackend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/payments/simulate" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-App") != "mcp-proxy-sample" || r.Header.Get("X-Agent") != "local-sample" {
			http.Error(w, "missing proxy headers", http.StatusBadRequest)
			return
		}

		var input struct {
			FinancingID     string `json:"financingId"`
			AvailableAmount int64  `json:"availableAmount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"financingId":        input.FinancingID,
			"approved":           true,
			"availableAmount":    input.AvailableAmount,
			"installmentsToPay":  3,
			"partialInstallment": true,
		})
	}))
}
