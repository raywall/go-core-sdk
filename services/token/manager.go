// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// token implements the STS token manager lifecycle.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package token

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	tokeninternal "github.com/raywall/go-core-sdk/services/token/internal"
	"github.com/raywall/go-core-sdk/services/token/types"
)

// Manager keeps an STS token available and refreshes it before expiry.
//
// Manager is safe for concurrent use. Start creates one background goroutine,
// and that goroutine is guaranteed to stop when Stop is called or when the
// Start context is cancelled.
type Manager struct {
	config  Config
	client  *tokeninternal.Client
	clock   tokeninternal.Clock
	logger  *slog.Logger
	token   *types.Token
	refresh sync.Mutex

	mu            sync.RWMutex
	status        Status
	cancel        context.CancelFunc
	done          chan struct{}
	nextRefreshAt time.Time
	lastErr       error
}

// NewManager constructs a token manager from Config.
//
// The returned manager is stopped and does not call STS until Start or Refresh
// is invoked. NewManager validates required configuration fields and prepares
// the private HTTP client used for STS calls.
func NewManager(config Config, options ...Option) (*Manager, error) {
	normalized := normalizeConfig(config)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}

	managerOptions := defaultOptions()
	for _, option := range options {
		if option != nil {
			option(&managerOptions)
		}
	}

	client, err := tokeninternal.NewClient(tokeninternal.ClientConfig{
		BaseURL:        normalized.BaseURL,
		Endpoint:       normalized.Endpoint,
		Headers:        normalized.Headers,
		ValidateSSL:    normalized.ValidateSSL,
		RequestTimeout: normalized.RequestTimeout,
		HTTPClient:     managerOptions.httpClient,
	})
	if err != nil {
		return nil, err
	}

	return &Manager{
		config: normalized,
		client: client,
		clock:  managerOptions.clock,
		logger: managerOptions.logger,
		token:  types.NewToken(types.TokenSnapshot{}),
		status: StatusStopped,
	}, nil
}

// Start fetches the initial token and starts the background refresh loop.
//
// Start returns only after the first token has been fetched successfully. If the
// initial fetch fails after retries, the manager remains stopped and the error
// is returned. Calling Start while the manager is starting or running returns
// types.ErrManagerAlreadyStarted.
func (m *Manager) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	runCtx, cancel := context.WithCancel(ctx)
	if err := m.beginStart(cancel); err != nil {
		cancel()
		return err
	}

	m.logger.InfoContext(runCtx, "token_manager_starting")
	if err := m.refreshWithRetry(runCtx); err != nil {
		cancel()
		m.finishStartFailure(err)
		m.logger.ErrorContext(runCtx, "token_manager_initial_refresh_failed", "error", err)
		return err
	}

	done := make(chan struct{})
	if err := m.finishStart(done); err != nil {
		cancel()
		return err
	}

	go m.run(runCtx, done)
	m.logger.InfoContext(runCtx, "token_manager_started", "next_refresh_at", m.nextRefreshTime())
	return nil
}

// Stop cancels the background refresh loop and waits for it to finish.
//
// Stop is idempotent. Calling Stop on an already stopped manager is a no-op.
func (m *Manager) Stop() {
	cancel, done := m.beginStop()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// Refresh requests a new token immediately.
//
// Refresh can be called while the manager is running or stopped. When running,
// it temporarily moves the manager to StatusRefreshing and then returns to
// StatusRunning after a successful refresh. When stopped, it updates the token
// without starting the background loop.
func (m *Manager) Refresh(ctx context.Context) error {
	return m.refreshNow(ctx, true)
}

func (m *Manager) refreshNow(ctx context.Context, manual bool) error {
	if ctx == nil {
		ctx = context.Background()
	}

	previousStatus := m.Status()
	if previousStatus == StatusStarting || previousStatus == StatusStopping {
		return types.ErrManagerStopped
	}
	if previousStatus != StatusStopped {
		m.setStatus(StatusRefreshing)
	}

	if err := m.refreshWithRetry(ctx); err != nil {
		m.recordRefreshFailure(ctx, err)
		return err
	}

	if previousStatus != StatusStopped {
		m.setStatus(StatusRunning)
	}
	if manual {
		m.logger.InfoContext(ctx, "token_manager_manual_refresh_completed", "next_refresh_at", m.nextRefreshTime())
	}
	return nil
}

// Status returns the current lifecycle state of the manager.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// LastError returns the last refresh error observed by the manager.
//
// A nil value means the last refresh attempt completed successfully or no
// refresh failure has been observed yet.
func (m *Manager) LastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastErr
}

// Token returns the stable token pointer managed by Manager.
//
// The pointer is available immediately after NewManager. Before the first
// successful refresh it contains the zero token value, and Token.ToString
// returns an empty string.
func (m *Manager) Token() *types.Token {
	return m.token
}

// TimeUntilRefresh returns how long remains before the next automatic refresh.
//
// A zero value means the token is missing, the next refresh is due now or the
// manager is stopped.
func (m *Manager) TimeUntilRefresh() time.Duration {
	next := m.nextRefreshTime()
	if next.IsZero() || m.Status() == StatusStopped {
		return 0
	}
	remaining := next.Sub(m.clock.Now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (m *Manager) run(ctx context.Context, done chan struct{}) {
	defer func() {
		m.markStopped(done)
		close(done)
		m.logger.InfoContext(ctx, "token_manager_stopped")
	}()

	for {
		wait := m.TimeUntilRefresh()
		timer := m.clock.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
			if err := m.refreshNow(ctx, false); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				m.scheduleRetrySoon()
			}
		}
	}
}

func (m *Manager) refreshWithRetry(ctx context.Context) error {
	m.refresh.Lock()
	defer m.refresh.Unlock()

	var lastErr error
	attempts := m.config.MaxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := m.fetchAndPublish(ctx); err != nil {
			lastErr = err
			m.logger.ErrorContext(ctx, "token_refresh_attempt_failed", "attempt", attempt, "max_attempts", attempts, "error", err)
			if attempt < attempts {
				if waitErr := m.waitBackoff(ctx, attempt); waitErr != nil {
					return waitErr
				}
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (m *Manager) fetchAndPublish(ctx context.Context) error {
	requestedAt := m.clock.Now()
	response, err := m.client.FetchToken(ctx, tokeninternal.Credentials{
		ClientID:     m.config.ClientID,
		ClientSecret: m.config.ClientSecret,
	})
	if err != nil {
		m.mu.Lock()
		m.lastErr = err
		m.mu.Unlock()
		return err
	}

	snapshot := response.ToSnapshot(requestedAt)
	m.token.Replace(snapshot)

	m.mu.Lock()
	m.nextRefreshAt = m.computeNextRefresh(snapshot)
	m.lastErr = nil
	m.mu.Unlock()

	m.logger.InfoContext(ctx, "token_refresh_completed", "expires_at", snapshot.ExpiresAt, "next_refresh_at", m.nextRefreshTime(), "scopes", snapshot.Scopes, "active", snapshot.Active)
	return nil
}

func (m *Manager) waitBackoff(ctx context.Context, attempt int) error {
	delay := m.config.RetryBackoff
	for range attempt - 1 {
		delay *= 2
	}

	timer := m.clock.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C():
		return nil
	}
}

func (m *Manager) computeNextRefresh(snapshot types.TokenSnapshot) time.Time {
	refreshAt := snapshot.ExpiresAt.Add(-m.config.RefreshBefore)
	if refreshAt.After(snapshot.RequestedAt) {
		return refreshAt
	}

	halfLife := snapshot.ExpiresIn / 2
	if halfLife <= 0 {
		return m.clock.Now()
	}
	return snapshot.RequestedAt.Add(halfLife)
}

func (m *Manager) beginStart(cancel context.CancelFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status != StatusStopped {
		return types.ErrManagerAlreadyStarted
	}
	m.status = StatusStarting
	m.cancel = cancel
	m.lastErr = nil
	return nil
}

func (m *Manager) finishStart(done chan struct{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusStopped {
		return types.ErrManagerStopped
	}
	m.status = StatusRunning
	m.done = done
	return nil
}

func (m *Manager) finishStartFailure(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.status = StatusStopped
	m.cancel = nil
	m.done = nil
	m.lastErr = err
}

func (m *Manager) beginStop() (context.CancelFunc, chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusStopped {
		return nil, nil
	}
	cancel := m.cancel
	done := m.done
	if done == nil {
		m.status = StatusStopped
		m.cancel = nil
		return cancel, nil
	}
	m.status = StatusStopping
	return cancel, done
}

func (m *Manager) markStopped(done chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.done != done {
		return
	}
	m.status = StatusStopped
	m.cancel = nil
	m.done = nil
}

func (m *Manager) recordRefreshFailure(ctx context.Context, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastErr = err
	if m.token.ToString() != "" {
		m.status = StatusDegraded
	} else {
		m.status = StatusStopped
	}
	m.logger.ErrorContext(ctx, "token_refresh_failed", "error", err, "status", m.status)
}

func (m *Manager) scheduleRetrySoon() {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.clock.Now().Add(m.config.RetryBackoff)
	m.nextRefreshAt = next
}

func (m *Manager) setStatus(status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}

func (m *Manager) nextRefreshTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nextRefreshAt
}
