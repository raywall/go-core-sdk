// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// internal implements reflective field path access.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package internal

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/raywall/go-core-sdk/services/selector/types"
)

type pathStep struct {
	name      string
	selectors []string
}

// ValueAt reads a dot-separated path from maps, structs, pointers, slices and
// arrays.
func ValueAt(item any, path string) (any, error) {
	steps, err := parsePath(path)
	if err != nil {
		return nil, err
	}

	value := reflect.ValueOf(item)
	for _, step := range steps {
		if step.name != "" {
			next, err := fieldValue(value, path, step.name)
			if err != nil {
				return nil, err
			}
			value = next
		}
		for _, selector := range step.selectors {
			next, err := selectedValue(value, path, selector)
			if err != nil {
				return nil, err
			}
			value = next
		}
	}

	value = indirect(value)
	if !value.IsValid() || !value.CanInterface() {
		return nil, types.FieldNotFoundError{Path: path}
	}
	return value.Interface(), nil
}

func parsePath(path string) ([]pathStep, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, types.InvalidConfigError{Field: "Path", Reason: "is required"}
	}

	rawSteps := strings.Split(path, ".")
	steps := make([]pathStep, 0, len(rawSteps))
	for _, raw := range rawSteps {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, types.InvalidConfigError{Field: "Path", Reason: "contains an empty segment"}
		}

		step := pathStep{}
		for {
			open := strings.Index(raw, "[")
			if open == -1 {
				if step.name == "" {
					step.name = raw
				}
				break
			}
			if step.name == "" {
				step.name = raw[:open]
			}
			close := strings.Index(raw[open:], "]")
			if close == -1 {
				return nil, types.InvalidConfigError{Field: "Path", Reason: "contains an unclosed selector"}
			}
			close += open
			selector := strings.TrimSpace(raw[open+1 : close])
			if selector == "" {
				return nil, types.InvalidConfigError{Field: "Path", Reason: "contains an empty selector"}
			}
			step.selectors = append(step.selectors, selector)
			raw = raw[close+1:]
			if raw == "" {
				break
			}
			if !strings.HasPrefix(raw, "[") {
				return nil, types.InvalidConfigError{Field: "Path", Reason: "contains invalid selector syntax"}
			}
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func fieldValue(value reflect.Value, path string, name string) (reflect.Value, error) {
	value = indirect(value)
	if !value.IsValid() {
		return reflect.Value{}, types.FieldNotFoundError{Path: path, Segment: name}
	}

	switch value.Kind() {
	case reflect.Map:
		return mapValue(value, path, name)
	case reflect.Struct:
		return structFieldValue(value, path, name)
	default:
		return reflect.Value{}, types.IncompatibleTypeError{Path: path, Value: safeInterface(value), Expected: "map or struct"}
	}
}

func selectedValue(value reflect.Value, path string, selector string) (reflect.Value, error) {
	value = indirect(value)
	if !value.IsValid() {
		return reflect.Value{}, types.FieldNotFoundError{Path: path, Segment: selector}
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		index, err := strconv.Atoi(selector)
		if err != nil {
			return reflect.Value{}, types.IncompatibleTypeError{Path: path, Value: selector, Expected: "numeric index"}
		}
		if index < 0 || index >= value.Len() {
			return reflect.Value{}, types.FieldNotFoundError{Path: path, Segment: selector}
		}
		return value.Index(index), nil
	case reflect.Map:
		return mapValue(value, path, selector)
	default:
		return reflect.Value{}, types.IncompatibleTypeError{Path: path, Value: safeInterface(value), Expected: "slice, array or map"}
	}
}

func mapValue(value reflect.Value, path string, key string) (reflect.Value, error) {
	keyValue, ok := mapKey(value.Type().Key(), key)
	if !ok {
		return reflect.Value{}, types.IncompatibleTypeError{Path: path, Value: key, Expected: "compatible map key"}
	}
	result := value.MapIndex(keyValue)
	if !result.IsValid() {
		return reflect.Value{}, types.FieldNotFoundError{Path: path, Segment: key}
	}
	return result, nil
}

func mapKey(keyType reflect.Type, key string) (reflect.Value, bool) {
	switch keyType.Kind() {
	case reflect.String:
		return reflect.ValueOf(key).Convert(keyType), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(key, 10, keyType.Bits())
		if err != nil {
			return reflect.Value{}, false
		}
		keyValue := reflect.New(keyType).Elem()
		keyValue.SetInt(parsed)
		return keyValue, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(key, 10, keyType.Bits())
		if err != nil {
			return reflect.Value{}, false
		}
		keyValue := reflect.New(keyType).Elem()
		keyValue.SetUint(parsed)
		return keyValue, true
	default:
		return reflect.Value{}, false
	}
}

func structFieldValue(value reflect.Value, path string, name string) (reflect.Value, error) {
	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := valueType.Field(index)
		if field.PkgPath != "" {
			continue
		}
		if fieldMatches(field, name) {
			return value.Field(index), nil
		}
	}
	return reflect.Value{}, types.FieldNotFoundError{Path: path, Segment: name}
}

func fieldMatches(field reflect.StructField, name string) bool {
	jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
	if jsonName != "" && jsonName != "-" && jsonName == name {
		return true
	}
	return field.Name == name || strings.EqualFold(field.Name, name)
}

func indirect(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func safeInterface(value reflect.Value) any {
	value = indirect(value)
	if !value.IsValid() || !value.CanInterface() {
		return nil
	}
	return value.Interface()
}
