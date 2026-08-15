package service_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/raywall/sts-token-management/app/application/service"
	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
)

// fakeProvider mocks repository.TokenProvider, returning tokens with a
// short TTL and counting how many times it was called.
type fakeProvider struct {
	calls  int32
	ttl    time.Duration
	failOn int32 // if > 0, the call with this ordinal fails
}

func (f *fakeProvider) FetchToken(context.Context) (*entity.TokenManagement, error) {
	n := atomic.AddInt32(&f.calls, 1)
	if f.failOn > 0 && n == f.failOn {
		return nil, errors.New("simulated STS failure")
	}
	return entity.NewTokenManagement("token-"+time.Now().Format(time.RFC3339Nano), "Bearer", f.ttl), nil
}

func TestTokenManagerService_StartAndCurrent(t *testing.T) {
	provider := &fakeProvider{ttl: time.Hour}
	mgr := service.NewTokenManagerService(provider, 5*time.Minute, 50*time.Millisecond, nil)

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error starting manager: %v", err)
	}
	defer mgr.Stop()

	token, err := mgr.Current(context.Background())
	if err != nil {
		t.Fatalf("unexpected error reading current token: %v", err)
	}
	if token.AccessToken == "" {
		t.Error("expected a non-empty access token after Start")
	}
	if atomic.LoadInt32(&provider.calls) != 1 {
		t.Errorf("expected exactly 1 fetch after Start, got %d", provider.calls)
	}
}

func TestTokenManagerService_BackgroundRenewal(t *testing.T) {
	// TTL shorter than the renewal threshold forces a renewal on the very
	// first background tick, proving the manager refreshes proactively
	// without any caller intervention.
	provider := &fakeProvider{ttl: 80 * time.Millisecond}
	mgr := service.NewTokenManagerService(provider, 200*time.Millisecond, 20*time.Millisecond, nil)

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error starting manager: %v", err)
	}
	defer mgr.Stop()

	deadline := time.After(2 * time.Second)
	for {
		if atomic.LoadInt32(&provider.calls) >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected background renewal to happen, only got %d fetches", provider.calls)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestTokenManagerService_StopTerminatesLoop(t *testing.T) {
	provider := &fakeProvider{ttl: time.Hour}
	mgr := service.NewTokenManagerService(provider, 5*time.Minute, 10*time.Millisecond, nil)

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error starting manager: %v", err)
	}

	stopped := make(chan struct{})
	go func() {
		mgr.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return: background goroutine leaked")
	}
}

func TestTokenManagerService_Current_NotStarted(t *testing.T) {
	provider := &fakeProvider{ttl: time.Hour}
	mgr := service.NewTokenManagerService(provider, 5*time.Minute, time.Minute, nil)

	_, err := mgr.Current(context.Background())
	if !errors.Is(err, errs.ErrManagerNotStarted) {
		t.Fatalf("expected ErrManagerNotStarted, got %v", err)
	}
}

func TestTokenManagerService_Start_InitialFetchFails(t *testing.T) {
	provider := &fakeProvider{ttl: time.Hour, failOn: 1}
	mgr := service.NewTokenManagerService(provider, 5*time.Minute, time.Minute, nil)

	err := mgr.Start(context.Background())
	if !errors.Is(err, errs.ErrTokenFetchFailed) {
		t.Fatalf("expected ErrTokenFetchFailed, got %v", err)
	}
}
