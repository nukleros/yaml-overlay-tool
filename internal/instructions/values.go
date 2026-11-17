// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"fmt"
	"os"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"gopkg.in/yaml.v3"
)

// getValues reads the values files used to render templated instructions files and
// merges them into one set of values. Each file is merged over the files before it, so
// later files win.
func getValues(fileNames []string) (interface{}, error) {
	var values interface{}

	yamlValues := make([]*yaml.Node, len(fileNames))

	for index, fileName := range fileNames {
		yamlValue, err := readValuesFile(fileName)
		if err != nil {
			return nil, err
		}

		yamlValues[index] = yamlValue
	}

	if len(yamlValues) != 0 {
		// MergeNode merges every following node into the first one.
		if err := actions.MergeNode(yamlValues...); err != nil {
			return nil, fmt.Errorf("failed to merge yaml values: %w", err)
		}

		// Round-trip through YAML to turn the merged node into plain Go values for templates.
		marshaled, err := yaml.Marshal(yamlValues[0])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal yaml values: %w", err)
		}

		if err := yaml.Unmarshal(marshaled, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal yaml values: %w", err)
		}
	}

	return values, nil
}

// readValuesFile decodes the first YAML document of a values file.
func readValuesFile(fileName string) (*yaml.Node, error) {
	reader, err := ReadStream(fileName)
	if err != nil {
		return nil, err
	}

	if file, isFile := reader.(*os.File); isFile {
		defer CloseFile(file)
	}

	var yamlValue yaml.Node

	if err := yaml.NewDecoder(reader).Decode(&yamlValue); err != nil {
		return nil, fmt.Errorf("failed to decode yaml values: %w", err)
	}

	return &yamlValue, nil
}
