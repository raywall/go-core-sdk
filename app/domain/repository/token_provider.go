// Package repository holds the domain ports: interfaces that describe what
// the domain needs from the outside world, without knowing how those needs
// are fulfilled. Infrastructure implements them; application orchestrates
// against them.
package repository

import (
	"context"

	"github.com/raywall/sts-token-management/app/domain/entity"
)

// TokenProvider is the output port through which the domain asks an
// external STS (Security Token Service) for a brand new access token.
// Implemented by infrastructure (e.g. an HTTP client that talks to the
// real STS endpoint).
type TokenProvider interface {
	// FetchToken requests a new token from the STS. It must return
	// errs.ErrTokenFetchFailed (wrapped) on any failure.
	FetchToken(ctx context.Context) (*entity.TokenManagement, error)
}
