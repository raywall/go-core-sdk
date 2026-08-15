// Command main runs a self-contained demo API that exercises the STS token
// management library end to end, with no external network dependency:
//
//  1. A mock STS endpoint (/mock/sts/token) simulates the external token
//     issuer, handing out short-lived tokens so renewal is visible in logs.
//  2. The library's factory wires a real TokenManager against that mock
//     endpoint and starts renewing tokens in the background.
//  3. A demo API exposes:
//     GET  /health  — readiness probe
//     GET  /token   — the token currently held by the manager
//     POST /echo    — echoes back method/headers/body (acts as a fake
//     downstream API so /call has something to hit locally)
//     POST /call    — runs CallAPIUseCase against any URL (e.g. point it at
//     this very server's /echo) with the bearer token attached
//     transparently, retries and timeout as requested.
//
// In a real deployment, STS_TOKEN_URL would point at your organization's
// actual STS instead of the bundled mock.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raywall/sts-token-management/app/main/config"
	"github.com/raywall/sts-token-management/app/main/factory"
	presentation "github.com/raywall/sts-token-management/app/presentation/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Start the mock STS on its own loopback listener. Listening happens
	// synchronously here, so by the time factory.Build() performs the
	// manager's initial token fetch below, the socket is already accepting
	// connections (the OS backlog absorbs the brief startup race).
	mockLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		logger.Error("failed to start mock STS listener", "error", err)
		os.Exit(1)
	}
	mockMux := http.NewServeMux()
	mockMux.HandleFunc("/mock/sts/token", presentation.MockSTSHandler)
	mockServer := &http.Server{Handler: mockMux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := mockServer.Serve(mockLn); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("mock STS server stopped unexpectedly", "error", err)
		}
	}()
	logger.Info("mock STS endpoint listening", "addr", mockLn.Addr().String())

	// 2. Load config, pointing STS_TOKEN_URL at the mock unless the
	// environment overrides it with a real STS endpoint.
	cfg := config.Load()
	if os.Getenv("STS_TOKEN_URL") == "" {
		cfg.STSTokenURL = "http://" + mockLn.Addr().String() + "/mock/sts/token"
	}

	// 3. Wire the library and start the background token manager.
	app, err := factory.Build(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to build application", "error", err)
		os.Exit(1)
	}
	defer app.TokenManager.Stop()
	logger.Info("token manager started", "refreshThreshold", cfg.RefreshThreshold, "pollInterval", cfg.PollInterval)

	// 4. Wire the demo presentation layer.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", presentation.HealthHandler)
	mux.HandleFunc("/echo", presentation.EchoHandler)
	mux.Handle("/token", presentation.NewTokenHandler(app.GetTokenUseCase))
	mux.Handle("/call", presentation.NewProxyHandler(app.CallAPIUseCase))

	server := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           presentation.LoggingMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		_ = mockServer.Shutdown(shutdownCtx)
	}()

	logger.Info("demo API listening", "addr", cfg.ServerAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("demo API server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
