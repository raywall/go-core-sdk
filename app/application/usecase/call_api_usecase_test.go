package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/raywall/sts-token-management/app/application/dto"
	"github.com/raywall/sts-token-management/app/application/usecase"
	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
)

// fakeRestAuthCaller mocks repository.RestAuthCaller for use case tests.
type fakeRestAuthCaller struct {
	gotRequest entity.APIRequest
	response   *entity.APIResponse
	err        error
}

func (f *fakeRestAuthCaller) Call(_ context.Context, req entity.APIRequest) (*entity.APIResponse, error) {
	f.gotRequest = req
	return f.response, f.err
}

func TestCallAPIUseCase_Execute_Success(t *testing.T) {
	caller := &fakeRestAuthCaller{
		response: &entity.APIResponse{
			StatusCode: 200,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       []byte(`{"ok":true}`),
		},
	}
	uc := usecase.NewCallAPIUseCase(caller)

	out, err := uc.Execute(context.Background(), dto.CallAPIInput{
		Method:       "POST",
		URL:          "https://api.example.com/resource",
		Headers:      map[string]string{"X-Custom": "value"},
		Body:         []byte(`{"a":1}`),
		Retries:      3,
		RetryDelayMs: 100,
		TimeoutMs:    5000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", out.StatusCode)
	}
	if string(out.Body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", out.Body)
	}

	// Ensure the DTO was correctly translated into a domain APIRequest.
	if caller.gotRequest.Method != entity.MethodPost {
		t.Errorf("expected POST, got %s", caller.gotRequest.Method)
	}
	if caller.gotRequest.URL != "https://api.example.com/resource" {
		t.Errorf("unexpected URL: %s", caller.gotRequest.URL)
	}
	if caller.gotRequest.Retries != 3 {
		t.Errorf("expected 3 retries, got %d", caller.gotRequest.Retries)
	}
	if caller.gotRequest.RetryDelay.Milliseconds() != 100 {
		t.Errorf("expected 100ms retry delay, got %v", caller.gotRequest.RetryDelay)
	}
	if caller.gotRequest.Timeout.Milliseconds() != 5000 {
		t.Errorf("expected 5000ms timeout, got %v", caller.gotRequest.Timeout)
	}
}

func TestCallAPIUseCase_Execute_InvalidRequest(t *testing.T) {
	caller := &fakeRestAuthCaller{}
	uc := usecase.NewCallAPIUseCase(caller)

	_, err := uc.Execute(context.Background(), dto.CallAPIInput{Method: "GET", URL: ""})
	if !errors.Is(err, errs.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}

	_, err = uc.Execute(context.Background(), dto.CallAPIInput{Method: "TRACE", URL: "https://example.com"})
	if !errors.Is(err, errs.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest for unsupported method, got %v", err)
	}
}

func TestCallAPIUseCase_Execute_PropagatesCallerError(t *testing.T) {
	wantErr := errors.New("boom")
	caller := &fakeRestAuthCaller{err: wantErr}
	uc := usecase.NewCallAPIUseCase(caller)

	_, err := uc.Execute(context.Background(), dto.CallAPIInput{Method: "GET", URL: "https://example.com"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}
