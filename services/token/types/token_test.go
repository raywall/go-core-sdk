// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types tests public token entity behavior.
//
// This file is part of the Token bounded context within the Token service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/raywall/go-core-sdk/services/token/types"
)

// TestToken_ToString verifies that Token builds an Authorization header value
// from the current token type and access token.
func TestToken_ToString(t *testing.T) {
	t.Parallel()

	token := types.NewToken(types.TokenSnapshot{
		AccessToken: "abc",
		TokenType:   "Bearer",
	})

	if got, want := token.ToString(), "Bearer abc"; got != want {
		t.Fatalf("Token.ToString() = %q, want %q", got, want)
	}
}

// TestToken_SnapshotCopiesScopes verifies that snapshots do not expose the
// token manager's internal scope slice.
func TestToken_SnapshotCopiesScopes(t *testing.T) {
	t.Parallel()

	requestedAt := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	token := types.NewToken(types.TokenSnapshot{
		AccessToken:  "abc",
		TokenType:    "Bearer",
		RequestedAt:  requestedAt,
		ExpiresAt:    requestedAt.Add(5 * time.Minute),
		ExpiresIn:    5 * time.Minute,
		RefreshToken: "refresh",
		Scopes:       []string{"pagamentos.read", "pedidos.read"},
		Active:       true,
	})

	snapshot := token.Snapshot()
	snapshot.Scopes[0] = "mutated"

	if got, want := token.Scopes(), []string{"pagamentos.read", "pedidos.read"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Token.Scopes() = %#v, want %#v", got, want)
	}
}

// TestToken_ReplacePublishesNewValues verifies that a stable token pointer can
// be updated in place by the manager.
func TestToken_ReplacePublishesNewValues(t *testing.T) {
	t.Parallel()

	token := types.NewToken(types.TokenSnapshot{
		AccessToken: "first",
		TokenType:   "Bearer",
		Active:      true,
	})

	token.Replace(types.TokenSnapshot{
		AccessToken: "second",
		TokenType:   "Bearer",
		Active:      true,
	})

	if got, want := token.AccessToken(), "second"; got != want {
		t.Fatalf("Token.AccessToken() = %q, want %q", got, want)
	}
}
