// Package factory is the composition root of the library: the only place
// where domain, application, infrastructure and presentation are all wired
// together. Consumers that embed this library in their own service can
// either call Build directly or use it as a reference for wiring the
// pieces themselves with their own configuration source.
package factory

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/raywall/sts-token-management/app/application/service"
	"github.com/raywall/sts-token-management/app/application/usecase"
	"github.com/raywall/sts-token-management/app/domain/repository"
	"github.com/raywall/sts-token-management/app/infrastructure/httpclient"
	"github.com/raywall/sts-token-management/app/infrastructure/sts"
	"github.com/raywall/sts-token-management/app/main/config"
)

// Application bundles everything a host application needs: the token
// manager (for lifecycle control) and the ready-to-use use cases.
type Application struct {
	TokenManager    repository.TokenManager
	GetTokenUseCase *usecase.GetCurrentTokenUseCase
	CallAPIUseCase  *usecase.CallAPIUseCase
}

// Build wires the full dependency graph from the inside out — Domain ->
// Application -> Infrastructure -> Presentation is left to callers — and
// starts the background token manager. The returned Application is ready
// to use immediately; call app.TokenManager.Stop() during shutdown.
func Build(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Application, error) {
	provider := sts.NewHTTPTokenProvider(nil, cfg.STSTokenURL, cfg.STSClientID, cfg.STSClientSecret, cfg.STSScope)

	tokenManager := service.NewTokenManagerService(provider, cfg.RefreshThreshold, cfg.PollInterval, logger)
	if err := tokenManager.Start(ctx); err != nil {
		return nil, fmt.Errorf("factory: starting token manager: %w", err)
	}

	restAuthCaller := httpclient.NewRestAuthCallerImpl(tokenManager, nil)

	return &Application{
		TokenManager:    tokenManager,
		GetTokenUseCase: usecase.NewGetCurrentTokenUseCase(tokenManager),
		CallAPIUseCase:  usecase.NewCallAPIUseCase(restAuthCaller),
	}, nil
}
