// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// http tests HTTP handler adapters.
//
// This file is part of the Handler bounded context within the Handler package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raywall/go-core-sdk/handlers"
	handlerhttp "github.com/raywall/go-core-sdk/handlers/http"
)

func TestRunnerHandlerProcessesEventEnvelope(t *testing.T) {
	t.Parallel()

	processor := handlers.ProcessorFunc(func(_ context.Context, event handlers.Event) (handlers.Result, error) {
		if event.Trigger != handlers.TriggerS3 {
			t.Fatalf("Trigger = %q, want %q", event.Trigger, handlers.TriggerS3)
		}
		return handlers.Result{Processed: len(event.Records)}, nil
	})
	runner, err := handlerhttp.NewRunner(handlers.Config{Trigger: handlers.TriggerS3}, processor)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	body, err := json.Marshal(handlers.Event{Records: []handlers.Record{{ID: "record-1"}}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	runner.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var result handlers.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if result.Processed != 1 {
		t.Fatalf("Processed = %d, want 1", result.Processed)
	}
}
