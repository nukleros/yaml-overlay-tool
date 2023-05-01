// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"errors"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ErrCombineOnlyForScalars occurs when a non-scalar is attempted to be combined with a scalar value.
var ErrCombineOnlyForScalars = errors.New("combine action can only be used on scalar values")

func CombineNode(originalValue, newValue *yaml.Node) error {
	if originalValue.Kind != yaml.ScalarNode || newValue.Kind != yaml.ScalarNode {
		return ErrCombineOnlyForScalars
	}

	mergeComments(originalValue, newValue)

	switch originalValue.Tag {
	case "!!int":
		original, err := strconv.Atoi(originalValue.Value)
		if err != nil {
			return fmt.Errorf("failed to %s to int for merging: %w", originalValue.Value, err)
		}

		newValueInt, err := strconv.Atoi(newValue.Value)
		if err != nil {
			return fmt.Errorf("failed to %s to int for merging: %w", newValue.Value, err)
		}

		originalValue.Value = strconv.Itoa(original + newValueInt)

		return nil

	case "!!bool":
		original, err := strconv.ParseBool(originalValue.Value)
		if err != nil {
			return fmt.Errorf("failed to %s to bool for merging: %w", originalValue.Value, err)
		}

		newValueBool, err := strconv.ParseBool(newValue.Value)
		if err != nil {
			return fmt.Errorf("failed to %s to bool for merging: %w", newValue.Value, err)
		}

		originalValue.Value = strconv.FormatBool(original && newValueBool)

		return nil
	}

	originalValue.Value += newValue.Value

	return nil
}
