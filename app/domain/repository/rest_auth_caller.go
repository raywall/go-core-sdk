package repository

import (
	"context"

	"github.com/raywall/sts-token-management/app/domain/entity"
)

// RestAuthCaller is the port that facilitates calling external APIs with
// the current bearer token attached transparently. Implemented by
// infrastructure (an HTTP client wired to a TokenManager); consumed by the
// application's CallAPIUseCase.
type RestAuthCaller interface {
	// Call executes req, injecting the current valid bearer token into the
	// Authorization header, honoring req.Timeout per attempt and retrying
	// up to req.Retries times with req.RetryDelay between attempts.
	Call(ctx context.Context, req entity.APIRequest) (*entity.APIResponse, error)
}
