package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/raywall/go-core-sdk/services/token"
)

func main() {
	sts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost || r.PostForm.Get("grant_type") != "client_credentials" {
			http.Error(w, "invalid token request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "sample-access-token",
			"token_type":    "Bearer",
			"expires_in":    300,
			"refresh_token": "sample-refresh-token",
			"scope":         "orders.read orders.write",
			"active":        true,
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
		RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer manager.Stop()

	fmt.Printf("status=%s authorization=%q refreshIn=%s\n",
		manager.Status(),
		manager.Token().ToString(),
		manager.TimeUntilRefresh().Round(time.Second),
	)
}
