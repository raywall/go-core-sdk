// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// internal implements entity conversion for CEL activation values.
//
// This file is part of the Decision bounded context within the Decision service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package internal

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

// Activation converts named entities into CEL-friendly activation values.
func Activation(entities map[string]any) (map[string]any, error) {
	activation := make(map[string]any, len(entities))
	for name, entity := range entities {
		converted, err := Convert(entity)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		activation[name] = converted
	}
	return activation, nil
}

// Convert normalizes structs, maps, slices and scalar values into values that
// CEL can evaluate through dynamic variables.
func Convert(value any) (any, error) {
	return convertValue(reflect.ValueOf(value), make(map[visit]struct{}))
}

func convertValue(value reflect.Value, visited map[visit]struct{}) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil
		}
		if value.Kind() == reflect.Pointer {
			key := visit{typ: value.Type(), pointer: value.Pointer()}
			if _, ok := visited[key]; ok {
				return nil, fmt.Errorf("cyclic entity reference")
			}
			visited[key] = struct{}{}
		}
		value = value.Elem()
	}

	if !value.CanInterface() {
		return nil, fmt.Errorf("value cannot be interfaced")
	}
	if value.Type() == timeType {
		return value.Interface(), nil
	}

	switch value.Kind() {
	case reflect.Struct:
		return convertStruct(value, visited)
	case reflect.Map:
		return convertMap(value, visited)
	case reflect.Slice, reflect.Array:
		return convertSlice(value, visited)
	case reflect.String:
		return value.String(), nil
	case reflect.Bool:
		return value.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return value.Float(), nil
	default:
		return nil, fmt.Errorf("unsupported kind %s", value.Kind())
	}
}

func convertStruct(value reflect.Value, visited map[visit]struct{}) (map[string]any, error) {
	output := make(map[string]any, value.NumField())
	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := valueType.Field(index)
		if field.PkgPath != "" {
			continue
		}
		name := fieldName(field)
		if name == "" {
			continue
		}
		converted, err := convertValue(value.Field(index), visited)
		if err != nil {
			return nil, err
		}
		output[name] = converted
	}
	return output, nil
}

func convertMap(value reflect.Value, visited map[visit]struct{}) (map[string]any, error) {
	if value.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("map key must be string")
	}
	output := make(map[string]any, value.Len())
	for _, key := range value.MapKeys() {
		converted, err := convertValue(value.MapIndex(key), visited)
		if err != nil {
			return nil, err
		}
		output[key.String()] = converted
	}
	return output, nil
}

func convertSlice(value reflect.Value, visited map[visit]struct{}) ([]any, error) {
	output := make([]any, 0, value.Len())
	for index := 0; index < value.Len(); index++ {
		converted, err := convertValue(value.Index(index), visited)
		if err != nil {
			return nil, err
		}
		output = append(output, converted)
	}
	return output, nil
}

func fieldName(field reflect.StructField) string {
	jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
	if jsonName == "-" {
		return ""
	}
	if jsonName != "" {
		return jsonName
	}
	return field.Name
}

type visit struct {
	typ     reflect.Type
	pointer uintptr
}
