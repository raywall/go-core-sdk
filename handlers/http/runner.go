// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// http implements the HTTP runner.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/raywall/go-core-sdk/handlers"
)

// Runner executes a Processor through an HTTP server.
type Runner struct {
	config    handlers.Config
	processor handlers.Processor
	server    *http.Server
}

// NewRunner constructs an HTTP runner.
func NewRunner(config handlers.Config, processor handlers.Processor) (*Runner, error) {
	normalized := handlers.NormalizeConfig(config)
	if processor == nil {
		return nil, handlers.InvalidConfigError{Field: "Processor", Reason: "is required"}
	}
	runner := &Runner{config: normalized, processor: processor}
	runner.server = &http.Server{
		Addr:              normalized.HTTPAddress,
		Handler:           runner.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return runner, nil
}

// Handler returns the HTTP handler used by the runner.
func (r *Runner) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", r.handleHealth)
	mux.HandleFunc(r.config.HTTPPath, r.handleEvent)
	return mux
}

// Start starts the HTTP server and shuts it down when ctx is cancelled.
func (r *Runner) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := r.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (r *Runner) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (r *Runner) handleEvent(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer request.Body.Close()

	var event handlers.Event
	if err := json.NewDecoder(request.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if event.Runtime == "" {
		event.Runtime = handlers.RuntimeHTTP
	}
	if event.Trigger == "" {
		event.Trigger = r.config.Trigger
	}

	result, err := r.processor.Process(request.Context(), event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
