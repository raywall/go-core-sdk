// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// types tests cache error values.
//
// This file is part of the Cache bounded context within the Cache service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package types_test

import (
	"errors"
	"testing"

	"github.com/raywall/go-core-sdk/services/cache/types"
)

// TestSentinelErrors verifies exported sentinel errors can be matched with
// errors.Is.
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	if !errors.Is(types.ErrItemNotFound, types.ErrItemNotFound) {
		t.Fatal("errors.Is(ErrItemNotFound, ErrItemNotFound) = false, want true")
	}
}
