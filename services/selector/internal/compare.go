// Copyright (c) 2026 Raywall. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// internal implements selector value comparison and amount conversion.
//
// This file is part of the Selector bounded context within the Selector service.
//
// Author:  Raywall
// Created: 2026-08-20
// Updated: 2026-08-20

package internal

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/raywall/go-core-sdk/services/selector/types"
)

// CompareValues compares two field values according to kind and time layout.
func CompareValues(left any, right any, kind types.ValueKind, timeLayout string) (int, error) {
	switch kind {
	case "", types.KindString:
		leftString := fmt.Sprint(left)
		rightString := fmt.Sprint(right)
		return compareOrdered(leftString, rightString), nil
	case types.KindNumber:
		leftNumber, err := numberValue(left)
		if err != nil {
			return 0, err
		}
		rightNumber, err := numberValue(right)
		if err != nil {
			return 0, err
		}
		return compareOrdered(leftNumber, rightNumber), nil
	case types.KindTime:
		leftTime, err := timeValue(left, timeLayout)
		if err != nil {
			return 0, err
		}
		rightTime, err := timeValue(right, timeLayout)
		if err != nil {
			return 0, err
		}
		return compareOrdered(leftTime.UnixNano(), rightTime.UnixNano()), nil
	default:
		return 0, types.InvalidConfigError{Field: "Kind", Reason: "unsupported value kind"}
	}
}

// AmountValue converts a field value into the caller's minimum monetary unit.
func AmountValue(value any, decimalScale int64, path string) (int64, error) {
	if decimalScale <= 0 {
		decimalScale = 1
	}

	amount, err := amountValue(value, decimalScale)
	if err != nil {
		return 0, types.InvalidAmountError{Path: path, Value: value, Reason: err.Error()}
	}
	if amount < 0 {
		return 0, types.InvalidAmountError{Path: path, Value: value, Reason: "must not be negative"}
	}
	return amount, nil
}

func compareOrdered[T ~string | ~float64 | ~int64](left T, right T) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func numberValue(value any) (float64, error) {
	value = unwrap(value)
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint8:
		return float64(typed), nil
	case uint16:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case string:
		parsed, err := strconv.ParseFloat(normalizeDecimalString(typed), 64)
		if err != nil {
			return 0, types.IncompatibleTypeError{Value: value, Expected: "numeric value"}
		}
		return parsed, nil
	default:
		return 0, types.IncompatibleTypeError{Value: value, Expected: "numeric value"}
	}
}

func timeValue(value any, layout string) (time.Time, error) {
	value = unwrap(value)
	if layout == "" {
		layout = time.RFC3339
	}

	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case string:
		parsed, err := time.Parse(layout, typed)
		if err != nil {
			return time.Time{}, types.IncompatibleTypeError{Value: value, Expected: "time value"}
		}
		return parsed, nil
	default:
		return time.Time{}, types.IncompatibleTypeError{Value: value, Expected: "time value"}
	}
}

func amountValue(value any, decimalScale int64) (int64, error) {
	value = unwrap(value)
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return 0, fmt.Errorf("overflows int64")
		}
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case uint64:
		if typed > math.MaxInt64 {
			return 0, fmt.Errorf("overflows int64")
		}
		return int64(typed), nil
	case float32:
		return scaledFloat(float64(typed), decimalScale)
	case float64:
		return scaledFloat(typed, decimalScale)
	case string:
		parsed, err := strconv.ParseFloat(normalizeDecimalString(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("must be numeric")
		}
		return scaledFloat(parsed, decimalScale)
	default:
		return 0, fmt.Errorf("must be numeric")
	}
}

func scaledFloat(value float64, decimalScale int64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("must be finite")
	}
	scaled := value * float64(decimalScale)
	rounded := math.Round(scaled)
	if math.Abs(scaled-rounded) > 0.0000001 {
		return 0, fmt.Errorf("has more precision than decimal scale supports")
	}
	if rounded > math.MaxInt64 || rounded < math.MinInt64 {
		return 0, fmt.Errorf("overflows int64")
	}
	return int64(rounded), nil
}

func normalizeDecimalString(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
}

func unwrap(value any) any {
	reflected := reflect.ValueOf(value)
	for reflected.IsValid() && (reflected.Kind() == reflect.Interface || reflected.Kind() == reflect.Pointer) {
		if reflected.IsNil() {
			return nil
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() || !reflected.CanInterface() {
		return nil
	}
	return reflected.Interface()
}
