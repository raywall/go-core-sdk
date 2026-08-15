package presentation

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// MockSTSHandler simulates an external STS token endpoint so that
// cmd/main.go can run as a fully self-contained demo, with no external
// network dependency. It is demo-only: real deployments must point the
// library's HTTPTokenProvider at an actual STS endpoint.
func MockSTSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	buf := make([]byte, 16)
	_, _ = rand.Read(buf)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": "demo-" + hex.EncodeToString(buf),
		"token_type":   "Bearer",
		"expires_in":   20, // short-lived on purpose, to make renewal visible in the demo logs
	})
}
