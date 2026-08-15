package repository

import (
	"context"

	"github.com/raywall/sts-token-management/app/domain/entity"
)

// TokenManager is the port that exposes the always-fresh, always-valid
// token to the rest of the system. It is implemented by an application
// service that runs in the background (see application/service), and is
// consumed both by use cases and by the infrastructure RestAuthCaller
// implementation, which needs the current token to authenticate calls.
type TokenManager interface {
	// Start puts the manager to work in the background: it fetches an
	// initial token synchronously and then keeps renewing it before
	// expiration until ctx is cancelled. Start must not block the caller
	// beyond the initial fetch.
	Start(ctx context.Context) error

	// Stop terminates the background renewal loop. It is safe to call more
	// than once.
	Stop()

	// Current returns the token currently held by the manager. It must be
	// safe for concurrent use by many goroutines.
	Current(ctx context.Context) (*entity.TokenManagement, error)
}
