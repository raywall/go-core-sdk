// Package httpclient implements the domain RestAuthCaller port: it performs
// HTTP calls with the current bearer token attached transparently, with
// configurable retries, inter-attempt delay and per-attempt timeout.
package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
	"github.com/raywall/sts-token-management/app/domain/repository"
)

// RestAuthCallerImpl implements repository.RestAuthCaller.
type RestAuthCallerImpl struct {
	client       *http.Client
	tokenManager repository.TokenManager
}

// NewRestAuthCallerImpl builds a RestAuthCallerImpl. client may be nil, in
// which case http.DefaultClient's transport is reused with no client-wide
// timeout (per-attempt timeouts are applied via context instead, since each
// call can request a different one).
func NewRestAuthCallerImpl(tokenManager repository.TokenManager, client *http.Client) *RestAuthCallerImpl {
	if client == nil {
		client = &http.Client{}
	}
	return &RestAuthCallerImpl{client: client, tokenManager: tokenManager}
}

// Call executes req against the target URL, attaching the current bearer
// token, retrying on transport errors or 5xx responses.
func (c *RestAuthCallerImpl) Call(ctx context.Context, req entity.APIRequest) (*entity.APIResponse, error) {
	if req.URL == "" || !req.Method.Valid() {
		return nil, fmt.Errorf("%w: method=%q url=%q", errs.ErrInvalidRequest, req.Method, req.URL)
	}

	token, err := c.tokenManager.Current(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: could not obtain bearer token: %v", errs.ErrRequestFailed, err)
	}

	attempts := req.Retries + 1
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := c.doOnce(ctx, req, token)
		if err == nil && resp.StatusCode < http.StatusInternalServerError {
			// Success, or a 4xx that the caller should handle themselves —
			// retrying a client error would not help.
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("server responded with status %d", resp.StatusCode)
		}

		if attempt == attempts-1 {
			break
		}
		if req.RetryDelay > 0 {
			select {
			case <-time.After(req.RetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("%w: %v", errs.ErrMaxRetriesExceeded, lastErr)
}

// doOnce performs a single HTTP attempt, applying req.Timeout (if set) as a
// per-attempt deadline derived from ctx.
func (c *RestAuthCallerImpl) doOnce(ctx context.Context, req entity.APIRequest, token *entity.TokenManagement) (*entity.APIResponse, error) {
	attemptCtx := ctx
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		attemptCtx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	var bodyReader io.Reader
	if len(req.Body) > 0 {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(attemptCtx, string(req.Method), req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	// Injected last so no custom header can accidentally shadow auth.
	httpReq.Header.Set("Authorization", token.BearerHeader())

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &entity.APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}
