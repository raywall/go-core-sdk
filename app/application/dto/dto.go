// Package dto defines the data transfer objects that cross the application
// boundary. Use cases accept and return DTOs, never domain entities
// directly, so inbound/outbound adapters stay decoupled from the domain
// model.
package dto

import "time"

// CallAPIInput is the input of CallAPIUseCase.
type CallAPIInput struct {
	Method       string
	URL          string
	Headers      map[string]string
	Body         []byte
	Retries      int
	RetryDelayMs int
	TimeoutMs    int
}

// CallAPIOutput is the output of CallAPIUseCase.
type CallAPIOutput struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

// TokenOutput is the output of GetCurrentTokenUseCase.
type TokenOutput struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}
