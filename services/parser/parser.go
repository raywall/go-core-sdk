// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// parser implements JSON based conversion.
//
// This file is part of the Parser bounded context within the Parser service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package parser

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"

	"github.com/raywall/go-core-sdk/services/parser/types"
)

// Parser converts values by serializing the source to JSON and decoding that
// JSON into a caller-provided target.
//
// Parser is safe for concurrent use after construction. It has no mutable
// conversion state.
type Parser struct {
	logger *slog.Logger
}

// New constructs a Parser service.
func New(configurers ...Option) (*Parser, error) {
	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer != nil {
			configurer(&options)
		}
	}
	return &Parser{logger: options.logger}, nil
}

// Parse converts source into target using JSON as the intermediate format.
//
// source must be non-nil and JSON-marshalable. target must be a non-nil pointer
// accepted by json.Unmarshal. Fields are matched by their json tags, which makes
// Parse useful for DTO-to-entity conversions with compatible field names.
func (p *Parser) Parse(ctx context.Context, source any, target any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateSource(source); err != nil {
		return err
	}
	if err := validateTarget(target); err != nil {
		return err
	}

	p.logger.DebugContext(ctx, "parser_parse_started", "source_type", typeName(source), "target_type", typeName(target))
	payload, err := json.Marshal(source)
	if err != nil {
		p.logger.ErrorContext(ctx, "parser_encode_failed", "source_type", typeName(source), "error", err)
		return types.EncodeError{Err: err}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := json.Unmarshal(payload, target); err != nil {
		p.logger.ErrorContext(ctx, "parser_decode_failed", "target_type", typeName(target), "error", err)
		return types.DecodeError{Err: err}
	}

	p.logger.InfoContext(ctx, "parser_parse_completed", "source_type", typeName(source), "target_type", typeName(target), "bytes", len(payload))
	return nil
}

// Parse converts source into target using a short-lived Parser.
func Parse(ctx context.Context, source any, target any, configurers ...Option) error {
	parser, err := New(configurers...)
	if err != nil {
		return err
	}
	return parser.Parse(ctx, source, target)
}

// ParseAs converts source into a new value of type T.
func ParseAs[T any](ctx context.Context, source any, configurers ...Option) (T, error) {
	var target T
	parser, err := New(configurers...)
	if err != nil {
		return target, err
	}
	if err := parser.Parse(ctx, source, &target); err != nil {
		return target, err
	}
	return target, nil
}

func validateSource(source any) error {
	if isNil(source) {
		return types.InvalidInputError{Field: "Source", Reason: "is required"}
	}
	return nil
}

func validateTarget(target any) error {
	if isNil(target) {
		return types.InvalidInputError{Field: "Target", Reason: "is required"}
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer {
		return types.InvalidInputError{Field: "Target", Reason: "must be a pointer"}
	}
	return nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func typeName(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}
