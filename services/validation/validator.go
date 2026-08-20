// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// validation implements recursive struct validation.
//
// This file is part of the Validation bounded context within the Validation service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package validation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"time"

	playground "github.com/go-playground/validator/v10"

	"github.com/raywall/go-core-sdk/services/validation/types"
)

var timeType = reflect.TypeOf(time.Time{})

// Validator validates structs and nested struct graphs using validator/v10.
//
// Validator is safe for concurrent use after construction. Register custom
// validations through options passed to New rather than mutating the underlying
// validator after use.
type Validator struct {
	validate     *playground.Validate
	logger       *slog.Logger
	fieldName    FieldNameFunc
	visitedDepth int
}

// New constructs a Validator service.
//
// New registers the configured tag name, field-name resolver and custom
// validations before returning. The default configuration uses validate tags,
// json field names in errors and a JSON slog handler.
func New(configurers ...Option) (*Validator, error) {
	options := defaultOptions()
	for _, configurer := range configurers {
		if configurer == nil {
			continue
		}
		if err := configurer(&options); err != nil {
			return nil, err
		}
	}

	validate := playground.New()
	validate.SetTagName(options.tagName)
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		return options.fieldNameFunc(field)
	})
	for tag, fn := range options.customValidations {
		if err := validate.RegisterValidation(tag, fn); err != nil {
			return nil, fmt.Errorf("register validation %q: %w", tag, err)
		}
	}
	for tag, fn := range options.customValidationsC {
		if err := validate.RegisterValidationCtx(tag, fn); err != nil {
			return nil, fmt.Errorf("register validation %q: %w", tag, err)
		}
	}

	return &Validator{
		validate:     validate,
		logger:       options.logger,
		fieldName:    options.fieldNameFunc,
		visitedDepth: 256,
	}, nil
}

// Validate validates value and returns all field failures in a typed error.
//
// value must be a struct, pointer to struct or a collection containing structs.
// Nested structs are validated recursively, including structs inside pointers,
// slices, arrays and maps. If validation succeeds, Validate returns nil.
func (v *Validator) Validate(ctx context.Context, value any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if value == nil {
		return types.InvalidInputError{Reason: "value is nil"}
	}

	root := reflect.ValueOf(value)
	if isNilRoot(root) {
		return types.InvalidInputError{Reason: "value is nil"}
	}
	if !isStructGraph(root) {
		return types.InvalidInputError{Reason: "value must be a struct, pointer to struct or collection containing structs"}
	}

	collector := newFieldCollector()
	if err := v.validateValue(ctx, root, path{}, collector, make(map[visit]struct{}), 0); err != nil {
		return err
	}
	fields := collector.fields()
	if len(fields) == 0 {
		return nil
	}

	v.logger.WarnContext(ctx, "validation_failed", "invalid_fields", len(fields))
	return &types.ValidationError{Fields: fields}
}

func (v *Validator) validateValue(ctx context.Context, value reflect.Value, prefix path, collector *fieldCollector, visited map[visit]struct{}, depth int) error {
	if depth > v.visitedDepth {
		return types.InvalidInputError{Reason: "maximum validation depth exceeded"}
	}
	if !value.IsValid() {
		return nil
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if value.Kind() == reflect.Pointer {
			pointer := value.Pointer()
			if pointer != 0 {
				key := visit{typ: value.Type(), pointer: pointer}
				if _, ok := visited[key]; ok {
					return nil
				}
				visited[key] = struct{}{}
			}
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		return v.validateStruct(ctx, value, prefix, collector, visited, depth)
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := v.validateValue(ctx, value.Index(index), prefix.index(index), collector, visited, depth+1); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			if err := v.validateValue(ctx, value.MapIndex(key), prefix.mapKey(key), collector, visited, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Validator) validateStruct(ctx context.Context, value reflect.Value, prefix path, collector *fieldCollector, visited map[visit]struct{}, depth int) error {
	if value.Type() == timeType || !value.CanInterface() {
		return nil
	}

	if err := v.validate.StructCtx(ctx, value.Interface()); err != nil {
		var validationErrors playground.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, validationErr := range validationErrors {
				collector.add(fieldErrorFromValidation(validationErr, prefix))
			}
		} else {
			var invalidValidation *playground.InvalidValidationError
			if errors.As(err, &invalidValidation) {
				return types.InvalidInputError{Reason: invalidValidation.Error()}
			}
			return err
		}
	}

	structPrefix := prefix
	if structPrefix.namespace == "" {
		structPrefix = path{
			namespace:       value.Type().Name(),
			structNamespace: value.Type().Name(),
		}
	}

	for index := 0; index < value.NumField(); index++ {
		field := value.Type().Field(index)
		if field.PkgPath != "" {
			continue
		}
		childPrefix := structPrefix.field(v.fieldName(field), field.Name)
		if childPrefix.namespace == "" {
			continue
		}
		if err := v.validateValue(ctx, value.Field(index), childPrefix, collector, visited, depth+1); err != nil {
			return err
		}
	}

	return nil
}

func fieldErrorFromValidation(validationErr playground.FieldError, prefix path) types.FieldError {
	namespace := applyPrefix(prefix.namespace, validationErr.Namespace())
	structNamespace := applyPrefix(prefix.structNamespace, validationErr.StructNamespace())
	return types.FieldError{
		Namespace:       namespace,
		StructNamespace: structNamespace,
		Field:           validationErr.Field(),
		StructField:     validationErr.StructField(),
		Tag:             validationErr.Tag(),
		ActualTag:       validationErr.ActualTag(),
		Param:           validationErr.Param(),
		Kind:            validationErr.Kind().String(),
		Type:            validationErr.Type().String(),
		Value:           validationErr.Value(),
		Message:         buildMessage(namespace, validationErr.Tag(), validationErr.Param()),
	}
}

func applyPrefix(prefix string, namespace string) string {
	if prefix == "" || namespace == "" {
		return namespace
	}

	_, tail, found := strings.Cut(namespace, ".")
	if !found {
		return prefix
	}
	return prefix + "." + tail
}

func buildMessage(namespace string, tag string, param string) string {
	if param == "" {
		return fmt.Sprintf("%s failed validation %q", namespace, tag)
	}
	return fmt.Sprintf("%s failed validation %q with parameter %q", namespace, tag, param)
}

func isStructGraph(value reflect.Value) bool {
	if !value.IsValid() {
		return false
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return true
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

func isNilRoot(value reflect.Value) bool {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return true
		}
		value = value.Elem()
	}
	return false
}

type visit struct {
	typ     reflect.Type
	pointer uintptr
}

type path struct {
	namespace       string
	structNamespace string
}

func (p path) field(namespace string, structNamespace string) path {
	if namespace == "" {
		return path{}
	}
	return path{
		namespace:       joinPath(p.namespace, namespace),
		structNamespace: joinPath(p.structNamespace, structNamespace),
	}
}

func (p path) index(index int) path {
	return path{
		namespace:       fmt.Sprintf("%s[%d]", p.namespace, index),
		structNamespace: fmt.Sprintf("%s[%d]", p.structNamespace, index),
	}
}

func (p path) mapKey(key reflect.Value) path {
	keyText := fmt.Sprint(key.Interface())
	return path{
		namespace:       fmt.Sprintf("%s[%s]", p.namespace, keyText),
		structNamespace: fmt.Sprintf("%s[%s]", p.structNamespace, keyText),
	}
}

func joinPath(prefix string, field string) string {
	if prefix == "" {
		return field
	}
	if field == "" {
		return prefix
	}
	return prefix + "." + field
}

type fieldCollector struct {
	byKey map[string]types.FieldError
}

func newFieldCollector() *fieldCollector {
	return &fieldCollector{byKey: make(map[string]types.FieldError)}
}

func (c *fieldCollector) add(field types.FieldError) {
	key := field.Namespace + "|" + field.Tag + "|" + field.Param
	if _, exists := c.byKey[key]; exists {
		return
	}
	c.byKey[key] = field
}

func (c *fieldCollector) fields() []types.FieldError {
	fields := make([]types.FieldError, 0, len(c.byKey))
	for _, field := range c.byKey {
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i int, j int) bool {
		if fields[i].Namespace == fields[j].Namespace {
			return fields[i].Tag < fields[j].Tag
		}
		return fields[i].Namespace < fields[j].Namespace
	})
	return fields
}
