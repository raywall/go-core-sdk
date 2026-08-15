// Package presentation holds the HTTP controllers for the demo/test API
// (the "aplicação de teste" required by the assignment). It lives under
// app/presentation/http to match the requested directory layout, but the
// package itself is named "presentation" to avoid colliding with the
// standard library's net/http import at call sites.
package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/raywall/sts-token-management/app/application/dto"
	"github.com/raywall/sts-token-management/app/application/usecase"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
)

// TokenHandler exposes the currently managed STS token, purely to prove
// that the background manager is doing its job.
type TokenHandler struct {
	getToken *usecase.GetCurrentTokenUseCase
}

// NewTokenHandler builds a TokenHandler.
func NewTokenHandler(getToken *usecase.GetCurrentTokenUseCase) *TokenHandler {
	return &TokenHandler{getToken: getToken}
}

// ServeHTTP handles GET /token.
func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	out, err := h.getToken.Execute(r.Context())
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// callAPIRequest is the wire format accepted by ProxyHandler.
type callAPIRequest struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers"`
	Body         json.RawMessage   `json:"body"`
	Retries      int               `json:"retries"`
	RetryDelayMs int               `json:"retryDelayMs"`
	TimeoutMs    int               `json:"timeoutMs"`
}

// ProxyHandler demonstrates CallAPIUseCase: it accepts a description of an
// outbound call and executes it with the bearer token attached
// transparently, retries and timeout applied as requested.
type ProxyHandler struct {
	callAPI *usecase.CallAPIUseCase
}

// NewProxyHandler builds a ProxyHandler.
func NewProxyHandler(callAPI *usecase.CallAPIUseCase) *ProxyHandler {
	return &ProxyHandler{callAPI: callAPI}
}

// ServeHTTP handles POST /call.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var reqBody callAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	out, err := h.callAPI.Execute(r.Context(), dto.CallAPIInput{
		Method:       reqBody.Method,
		URL:          reqBody.URL,
		Headers:      reqBody.Headers,
		Body:         []byte(reqBody.Body),
		Retries:      reqBody.Retries,
		RetryDelayMs: reqBody.RetryDelayMs,
		TimeoutMs:    reqBody.TimeoutMs,
	})
	if err != nil {
		mapDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func mapDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errs.ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errs.ErrNoToken), errors.Is(err, errs.ErrManagerNotStarted), errors.Is(err, errs.ErrTokenExpired):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// HealthHandler is a trivial readiness probe used by cmd/main.go to know
// when the demo server is ready to accept traffic.
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// EchoHandler echoes back the method, headers and body it received. It
// plays the role of a "downstream API" in the local demo, so /call can
// target it and prove the Authorization header was injected correctly
// without requiring outbound internet access.
func EchoHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, _ := io.ReadAll(r.Body)
	writeJSON(w, http.StatusOK, map[string]any{
		"method":  r.Method,
		"headers": r.Header,
		"body":    string(body),
	})
}
