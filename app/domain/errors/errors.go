// Package errs defines the domain-typed errors for the STS token management
// library. It is named "errs" (rather than "errors") so it never shadows
// the standard library "errors" package at call sites.
package errs

import "errors"

var (
	// ErrNoToken means the token manager has not produced a token yet
	// (e.g. Current was called before Start finished its first fetch).
	ErrNoToken = errors.New("token management: no token available yet")

	// ErrTokenExpired means the cached token is no longer valid and the
	// background manager has not been able to renew it.
	ErrTokenExpired = errors.New("token management: token expired")

	// ErrTokenFetchFailed wraps failures returned by the STS endpoint while
	// requesting a new token.
	ErrTokenFetchFailed = errors.New("token management: failed to fetch token from STS")

	// ErrInvalidRequest means the caller built an APIRequest that violates
	// RestAuthCaller's contract (e.g. empty URL or unsupported method).
	ErrInvalidRequest = errors.New("token management: invalid API request")

	// ErrRequestFailed wraps transport-level failures of an API call.
	ErrRequestFailed = errors.New("token management: API request failed")

	// ErrMaxRetriesExceeded means every attempt (initial + retries) failed.
	ErrMaxRetriesExceeded = errors.New("token management: max retries exceeded")

	// ErrManagerNotStarted means a caller tried to use the token manager
	// before calling Start.
	ErrManagerNotStarted = errors.New("token management: manager not started")
)
