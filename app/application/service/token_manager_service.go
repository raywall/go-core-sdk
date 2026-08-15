// Package service hosts application-level services: long-lived orchestration
// logic that is more than a single request/response use case but still
// contains no framework or transport code.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
	"github.com/raywall/sts-token-management/app/domain/repository"
)

// TokenManagerService implements repository.TokenManager. It keeps a
// TokenManagement instance fresh in the background, without impacting the
// hosting application: after Start returns, all renewal work happens in a
// single goroutine that is guaranteed to stop when its context is
// cancelled or Stop is called.
type TokenManagerService struct {
	provider  repository.TokenProvider
	threshold time.Duration // how far ahead of expiry to renew
	interval  time.Duration // how often to check whether renewal is due
	logger    *slog.Logger

	mu       sync.RWMutex
	current  *entity.TokenManagement
	lastErr  error
	cancel   context.CancelFunc
	done     chan struct{}
	started  bool
	stopOnce sync.Once
}

// NewTokenManagerService builds a TokenManagerService.
//
//   - threshold: renew the token once it is within this duration of
//     expiring.
//   - interval: how often the background loop checks whether a renewal is
//     due. Should be smaller than threshold.
func NewTokenManagerService(provider repository.TokenProvider, threshold, interval time.Duration, logger *slog.Logger) *TokenManagerService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TokenManagerService{
		provider:  provider,
		threshold: threshold,
		interval:  interval,
		logger:    logger,
	}
}

// Start fetches an initial token synchronously (so Current is usable right
// after Start returns) and then launches the background renewal loop.
// The loop terminates deterministically when ctx is cancelled or Stop is
// called — it never leaks.
func (s *TokenManagerService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.mu.Unlock()

	if err := s.refresh(ctx); err != nil {
		s.mu.Lock()
		s.lastErr = err
		s.mu.Unlock()
		s.logger.Error("initial token fetch failed", "error", err)
		return fmt.Errorf("%w: %v", errs.ErrTokenFetchFailed, err)
	}

	go s.run(loopCtx)
	return nil
}

// run is the single background goroutine backing this service. It is
// started exactly once by Start and terminates exactly once, closing done,
// whenever loopCtx is cancelled — either via Stop() or via the parent
// context passed to Start.
func (s *TokenManagerService) run(loopCtx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			needsRenew := s.current.IsNearExpiration(s.threshold)
			s.mu.RUnlock()
			if !needsRenew {
				continue
			}
			if err := s.refresh(loopCtx); err != nil {
				s.mu.Lock()
				s.lastErr = err
				s.mu.Unlock()
				s.logger.Error("background token refresh failed", "error", err)
			}
		case <-loopCtx.Done():
			s.logger.Info("token manager background loop stopped")
			return
		}
	}
}

// refresh calls the provider and swaps the cached token atomically.
func (s *TokenManagerService) refresh(ctx context.Context) error {
	token, err := s.provider.FetchToken(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", errs.ErrTokenFetchFailed, err)
	}
	s.mu.Lock()
	s.current = token
	s.lastErr = nil
	s.mu.Unlock()
	return nil
}

// Stop terminates the background loop and waits for it to fully exit. Safe
// to call multiple times and safe to call even if Start was never called.
func (s *TokenManagerService) Stop() {
	s.stopOnce.Do(func() {
		s.mu.RLock()
		cancel, done, started := s.cancel, s.done, s.started
		s.mu.RUnlock()
		if !started {
			return
		}
		cancel()
		<-done
	})
}

// Current returns the token currently cached by the manager. It never
// triggers I/O itself — that is the background loop's job — so it is cheap
// and safe to call on every outgoing API request.
func (s *TokenManagerService) Current(ctx context.Context) (*entity.TokenManagement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return nil, errs.ErrManagerNotStarted
	}
	if s.current == nil {
		return nil, errs.ErrNoToken
	}
	if s.current.IsExpired() {
		if s.lastErr != nil {
			return nil, fmt.Errorf("%w: %v", errs.ErrTokenExpired, s.lastErr)
		}
		return nil, errs.ErrTokenExpired
	}
	tokenCopy := *s.current
	return &tokenCopy, nil
}
