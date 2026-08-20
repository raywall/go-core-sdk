package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/raywall/go-core-sdk/services/mcp/proxy"
	proxytypes "github.com/raywall/go-core-sdk/services/mcp/proxy/types"
)

func TestRunListsAndInvokesProxyTool(t *testing.T) {
	t.Parallel()

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
				Description: "Simulate payment creation for a student financing.",
				Method:      http.MethodPost,
				Path:        "/payments/simulate",
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
	})
	if err != nil {
		t.Fatalf("proxy.New() error = %v", err)
	}

	var out bytes.Buffer
	if err := run(context.Background(), &out, mcpProxy); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"tools=1 first=simulate_student_financing_payment", "status=200", `"installmentsToPay":3`} {
		if !strings.Contains(got, want) {
			t.Fatalf("run() output = %q, missing %q", got, want)
		}
	}
}
