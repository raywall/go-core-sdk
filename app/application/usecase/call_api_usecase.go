package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raywall/sts-token-management/app/application/dto"
	"github.com/raywall/sts-token-management/app/domain/entity"
	errs "github.com/raywall/sts-token-management/app/domain/errors"
	"github.com/raywall/sts-token-management/app/domain/repository"
)

// CallAPIUseCase facilitates calling external APIs with the bearer token
// attached transparently, delegating the actual HTTP work to the
// RestAuthCaller port.
type CallAPIUseCase struct {
	restAuthCaller repository.RestAuthCaller
}

// NewCallAPIUseCase builds the use case.
func NewCallAPIUseCase(restAuthCaller repository.RestAuthCaller) *CallAPIUseCase {
	return &CallAPIUseCase{restAuthCaller: restAuthCaller}
}

// Execute validates the input, converts it into a domain APIRequest and
// delegates to RestAuthCaller.
func (uc *CallAPIUseCase) Execute(ctx context.Context, in dto.CallAPIInput) (dto.CallAPIOutput, error) {
	method := entity.HTTPMethod(in.Method)
	if in.URL == "" || !method.Valid() {
		return dto.CallAPIOutput{}, fmt.Errorf("%w: method=%q url=%q", errs.ErrInvalidRequest, in.Method, in.URL)
	}

	req := entity.APIRequest{
		Method:     method,
		URL:        in.URL,
		Headers:    in.Headers,
		Body:       in.Body,
		Retries:    in.Retries,
		RetryDelay: time.Duration(in.RetryDelayMs) * time.Millisecond,
		Timeout:    time.Duration(in.TimeoutMs) * time.Millisecond,
	}

	resp, err := uc.restAuthCaller.Call(ctx, req)
	if err != nil {
		return dto.CallAPIOutput{}, err
	}

	return dto.CallAPIOutput{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}, nil
}
