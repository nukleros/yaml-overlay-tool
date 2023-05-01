// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/qdm12/reprint"
	"gopkg.in/yaml.v3"
)

var (
	// ErrMergeMustBeOfSameType occurs when attempting to merge two different types.
	ErrMergeMustBeOfSameType = errors.New("both values must be of same type to use 'merge' action")
	// ErrMergeUnsupportedType occurs when an unsupported YAML type is attempted to be merged.
	ErrMergeUnsupportedType = errors.New("not a supported yaml type for merging")
)

func MergeNode(yamlNodes ...*yaml.Node) error {
	if len(yamlNodes) <= 1 {
		return nil
	}

	o := yamlNodes[0]

	for _, node := range yamlNodes[1:] {
		if err := merge(o, node); err != nil {
			return err
		}
	}

	return nil
}

func merge(originalYamlNode, newYamlNode *yaml.Node, keyName ...string) error {
	switch newYamlNode.Kind {
	case yaml.DocumentNode:
		return mergeDocument(originalYamlNode, newYamlNode)
	case yaml.MappingNode:
		return mergeMap(originalYamlNode, newYamlNode)
	case yaml.SequenceNode:
		return mergeArray(originalYamlNode, newYamlNode)
	case yaml.ScalarNode:
		mergeScalar(originalYamlNode, newYamlNode, keyName...)
	case yaml.AliasNode:
		return fmt.Errorf("%s is %w", originalYamlNode.LongTag(), ErrMergeUnsupportedType)
	}

	originalYamlNode.Tag = newYamlNode.Tag

	return nil
}

func mergeDocument(originalYamlNode, newYamlNode *yaml.Node) error {
	if originalYamlNode.Content != nil && newYamlNode.Content != nil {
		mergeComments(originalYamlNode, newYamlNode, originalYamlNode.Value)

		if err := merge(originalYamlNode.Content[0], newYamlNode.Content[0], originalYamlNode.Value); err != nil {
			return err
		}
	}

	return nil
}

func mergeMap(originalYamlNode, newYamlNode *yaml.Node) error {
	for newNodeIndex := 0; newNodeIndex < len(newYamlNode.Content)-1; newNodeIndex += 2 {
		resultFound := false

		if originalYamlNode.Kind != yaml.MappingNode {
			originalYamlNode.Kind = yaml.MappingNode
			originalYamlNode.Tag = "!!map"
			originalYamlNode.Value = ""
			originalYamlNode.Content = []*yaml.Node{}
		}

		for originalNodeIndex := 0; originalNodeIndex < len(originalYamlNode.Content)-1; originalNodeIndex += 2 {
			var foundMarker bool

			if foundMarker = checkForMarkers(newYamlNode.Content[newNodeIndex].Value); !foundMarker {
				if originalYamlNode.Content[originalNodeIndex].Value != newYamlNode.Content[newNodeIndex].Value {
					continue
				}
			}

			resultFound = true

			keyName := originalYamlNode.Content[originalNodeIndex].Value
			value := originalYamlNode.Content[originalNodeIndex+1].Value

			mergeComments(originalYamlNode.Content[originalNodeIndex], newYamlNode.Content[newNodeIndex], value, keyName)

			if err := merge(originalYamlNode.Content[originalNodeIndex+1], newYamlNode.Content[newNodeIndex+1], keyName); err != nil {
				return err
			}

			if !foundMarker {
				break
			}

			mergeScalar(originalYamlNode.Content[originalNodeIndex], newYamlNode.Content[newNodeIndex], value, keyName)
		}

		if !resultFound {
			if err := addNode(originalYamlNode, newYamlNode.Content[newNodeIndex:newNodeIndex+2]...); err != nil {
				return err
			}
		}
	}

	return nil
}

func mergeArray(originalYamlNode, newYamlNode *yaml.Node) error {
	if originalYamlNode.Kind != yaml.SequenceNode {
		originalYamlNode.Kind = yaml.SequenceNode
		originalYamlNode.Tag = "!!seq"
		originalYamlNode.Value = ""
		originalYamlNode.Content = []*yaml.Node{}
	}

	if newYamlNode.Content != nil {
		mergeComments(originalYamlNode, newYamlNode, originalYamlNode.Value)

		if err := addNode(originalYamlNode, newYamlNode.Content...); err != nil {
			return err
		}
	}

	return nil
}

func mergeScalar(originalYamlNode, newYamlNode *yaml.Node, values ...string) {
	var keyName string

	var value string

	headComment := strings.TrimPrefix(originalYamlNode.HeadComment, "#")
	lineComment := strings.TrimPrefix(originalYamlNode.LineComment, "#")
	footComment := strings.TrimPrefix(originalYamlNode.FootComment, "#")

	mergeComments(originalYamlNode, newYamlNode, values...)

	switch {
	case values == nil:
		value = originalYamlNode.Value
		keyName = ""
	case len(values) > 1:
		value = values[0]
		keyName = values[1]
	case len(values) == 1:
		value = originalYamlNode.Value
		keyName = values[0]
	}

	if originalYamlNode.Kind != yaml.ScalarNode {
		originalYamlNode.Kind = yaml.ScalarNode
		originalYamlNode.Value = ""
		originalYamlNode.Content = nil
	}

	originalYamlNode.Value = format(newYamlNode.Value, value, lineComment, headComment, footComment, keyName)
}

func mergeComments(originalYamlNode, newYamlNode *yaml.Node, values ...string) {
	var keyName string

	var value string

	lineComment := strings.TrimPrefix(originalYamlNode.LineComment, "#")
	headComment := strings.TrimPrefix(originalYamlNode.HeadComment, "#")
	footComment := strings.TrimPrefix(originalYamlNode.FootComment, "#")

	switch {
	case values == nil:
		value = originalYamlNode.Value
		keyName = ""
	case len(values) > 1:
		value = values[0]
		keyName = values[1]
	case len(values) == 1:
		value = originalYamlNode.Value
		keyName = values[0]
	}

	switch {
	case newYamlNode.HeadComment != "":
		originalYamlNode.HeadComment = format(newYamlNode.HeadComment, value, lineComment, headComment, footComment, keyName)

		fallthrough
	case newYamlNode.LineComment != "":
		originalYamlNode.LineComment = format(newYamlNode.LineComment, value, lineComment, headComment, footComment, keyName)

		fallthrough
	case newYamlNode.FootComment != "":
		originalYamlNode.FootComment = format(newYamlNode.FootComment, value, lineComment, headComment, footComment, keyName)
	}
}

func addNode(originalYamlNode *yaml.Node, newNodeValues ...*yaml.Node) error {
	additionalNodes := make([]*yaml.Node, len(newNodeValues))

	if err := reprint.FromTo(&newNodeValues, &additionalNodes); err != nil {
		return fmt.Errorf("failed to insert value during merge: %w", err)
	}

	sanitizeNode(additionalNodes...)

	originalYamlNode.Content = append(originalYamlNode.Content, additionalNodes...)

	return nil
}
