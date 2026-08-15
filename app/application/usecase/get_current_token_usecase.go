// Package usecase implements the application use cases. Each use case
// orchestrates against domain ports (repository interfaces) and translates
// between DTOs and domain entities; it never contains transport code.
package usecase

import (
	"context"

	"github.com/raywall/sts-token-management/app/application/dto"
	"github.com/raywall/sts-token-management/app/domain/repository"
)

// GetCurrentTokenUseCase exposes the token currently held by the background
// TokenManager, translated into a DTO.
type GetCurrentTokenUseCase struct {
	tokenManager repository.TokenManager
}

// NewGetCurrentTokenUseCase builds the use case.
func NewGetCurrentTokenUseCase(tokenManager repository.TokenManager) *GetCurrentTokenUseCase {
	return &GetCurrentTokenUseCase{tokenManager: tokenManager}
}

// Execute returns the current, valid token as a DTO.
func (uc *GetCurrentTokenUseCase) Execute(ctx context.Context) (dto.TokenOutput, error) {
	token, err := uc.tokenManager.Current(ctx)
	if err != nil {
		return dto.TokenOutput{}, err
	}
	return dto.TokenOutput{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresAt:   token.ExpiresAt,
	}, nil
}
