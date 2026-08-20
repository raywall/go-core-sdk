// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// validation implements construction options for the validation service.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package validation

import (
	"log/slog"
	"os"
	"reflect"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

const defaultTagName = "validate"

// FieldNameFunc resolves the external name used to report a struct field.
type FieldNameFunc func(reflect.StructField) string

// Option customizes a Validator during construction.
type Option func(*options) error

type options struct {
	logger             *slog.Logger
	tagName            string
	fieldNameFunc      FieldNameFunc
	customValidations  map[string]playground.Func
	customValidationsC map[string]playground.FuncCtx
}

// WithLogger configures the structured logger used by Validator.
//
// The default logger writes JSON records to stdout. Passing nil keeps the
// default logger.
func WithLogger(logger *slog.Logger) Option {
	return func(options *options) error {
		if logger != nil {
			options.logger = logger
		}
		return nil
	}
}

// WithTagName configures the struct tag name used by validator/v10.
//
// The default tag name is validate. Passing an empty value keeps the default.
func WithTagName(tagName string) Option {
	return func(options *options) error {
		tagName = strings.TrimSpace(tagName)
		if tagName != "" {
			options.tagName = tagName
		}
		return nil
	}
}

// WithFieldNameFunc configures how field names are exposed in validation
// errors.
//
// The default resolver uses the json tag name when present and falls back to
// the Go field name.
func WithFieldNameFunc(fn FieldNameFunc) Option {
	return func(options *options) error {
		if fn != nil {
			options.fieldNameFunc = fn
		}
		return nil
	}
}

// WithValidation registers a custom field validation function.
//
// The tag value is the name used inside validate tags. Registration happens
// during New, before the Validator can be used concurrently.
func WithValidation(tag string, fn playground.Func) Option {
	return func(options *options) error {
		if fn == nil || strings.TrimSpace(tag) == "" {
			return nil
		}
		options.customValidations[strings.TrimSpace(tag)] = fn
		return nil
	}
}

// WithValidationCtx registers a context-aware custom field validation function.
//
// The tag value is the name used inside validate tags. Registration happens
// during New, before the Validator can be used concurrently.
func WithValidationCtx(tag string, fn playground.FuncCtx) Option {
	return func(options *options) error {
		if fn == nil || strings.TrimSpace(tag) == "" {
			return nil
		}
		options.customValidationsC[strings.TrimSpace(tag)] = fn
		return nil
	}
}

func defaultOptions() options {
	return options{
		logger:             slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		tagName:            defaultTagName,
		fieldNameFunc:      jsonFieldName,
		customValidations:  make(map[string]playground.Func),
		customValidationsC: make(map[string]playground.FuncCtx),
	}
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	name := strings.Split(tag, ",")[0]
	if name == "-" {
		return ""
	}
	if name != "" {
		return name
	}
	return field.Name
}
