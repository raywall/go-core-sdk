package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/services/token"
)

func TestRunStartsTokenManagerAndPrintsAuthorization(t *testing.T) {
	t.Parallel()

	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.PostForm.Get("grant_type") != "client_credentials" {
			http.Error(w, "invalid grant", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "sample-access-token",
			"token_type":   "Bearer",
			"expires_in":   300,
			"active":       true,
		})
	}))
	defer sts.Close()

	manager, err := token.NewManager(token.Config{
		BaseURL:        sts.URL,
		Endpoint:       "/oauth/token",
		ClientID:       "client-id",
		ClientSecret:   "client-secret",
		ValidateSSL:    true,
		RefreshBefore:  30 * time.Second,
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("token.NewManager() error = %v", err)
	}

	var out bytes.Buffer
	if err := run(context.Background(), &out, manager); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `authorization="Bearer sample-access-token"`) {
		t.Fatalf("run() output = %q", got)
	}
}
