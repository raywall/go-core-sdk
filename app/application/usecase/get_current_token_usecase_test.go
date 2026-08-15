package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/raywall/sts-token-management/app/application/usecase"
	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
)

// fakeTokenManager mocks repository.TokenManager for use case tests.
type fakeTokenManager struct {
	token *entity.TokenManagement
	err   error
}

func (f *fakeTokenManager) Start(context.Context) error { return nil }
func (f *fakeTokenManager) Stop()                       {}
func (f *fakeTokenManager) Current(context.Context) (*entity.TokenManagement, error) {
	return f.token, f.err
}

func TestGetCurrentTokenUseCase_Execute_Success(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	manager := &fakeTokenManager{
		token: &entity.TokenManagement{
			AccessToken: "abc123",
			TokenType:   "Bearer",
			ExpiresAt:   expiresAt,
		},
	}
	uc := usecase.NewGetCurrentTokenUseCase(manager)

	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken != "abc123" {
		t.Errorf("expected access token abc123, got %s", out.AccessToken)
	}
	if !out.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected expiry %v, got %v", expiresAt, out.ExpiresAt)
	}
}

func TestGetCurrentTokenUseCase_Execute_PropagatesError(t *testing.T) {
	manager := &fakeTokenManager{err: errs.ErrTokenExpired}
	uc := usecase.NewGetCurrentTokenUseCase(manager)

	_, err := uc.Execute(context.Background())
	if !errors.Is(err, errs.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}
