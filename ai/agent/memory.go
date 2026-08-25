// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// agent implements memory stores.
//
// This file is part of the Agent bounded context within the AI package.
//
// Author:  Raywall
// Created: 2026-08-24
// Updated: 2026-08-24

package agent

import (
	"context"
	"sync"
)

// InMemory stores messages in process memory.
//
// InMemory is safe for concurrent use.
type InMemory struct {
	mu       sync.RWMutex
	limit    int
	messages []Message
}

// NewShortTermMemory constructs an in-memory store capped to the most recent messages.
func NewShortTermMemory(limit int) *InMemory {
	return &InMemory{limit: limit}
}

// NewLongTermMemory constructs an uncapped in-memory store.
func NewLongTermMemory() *InMemory {
	return &InMemory{}
}

// Load returns stored messages in replay order.
func (m *InMemory) Load(context.Context) ([]Message, error) {
	if m == nil {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return copyMessages(m.messages), nil
}

// Append stores one message.
func (m *InMemory) Append(_ context.Context, message Message) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, cloneMessage(message))
	if m.limit > 0 && len(m.messages) > m.limit {
		m.messages = append([]Message(nil), m.messages[len(m.messages)-m.limit:]...)
	}
	return nil
}

// Clear removes all stored messages.
func (m *InMemory) Clear(context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = nil
	return nil
}

func copyMessages(messages []Message) []Message {
	copied := make([]Message, len(messages))
	for i, message := range messages {
		copied[i] = cloneMessage(message)
	}
	return copied
}

func cloneMessage(message Message) Message {
	cloned := message
	cloned.ToolCalls = append([]ToolCall(nil), message.ToolCalls...)
	return cloned
}
