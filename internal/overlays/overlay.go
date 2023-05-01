// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package overlays

import (
	"fmt"

	"github.com/op/go-logging"
	"gopkg.in/yaml.v3"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
)

var log = logging.MustGetLogger("overlays") //nolint:gochecknoglobals

type Overlay struct {
	Name          string          `yaml:"name,omitempty"`
	Query         Queries         `yaml:"query,omitempty"`
	Value         yaml.Node       `yaml:"value,omitempty"`
	Action        actions.Action  `yaml:"action,omitempty"`
	DocumentQuery DocumentQueries `yaml:"documentQuery,omitempty"`
	OnMissing     OnMissing       `yaml:"onMissing,omitempty"`
	DocumentIndex []int           `yaml:"documentIndex,omitempty"`
}

func (overlay *Overlay) Apply(yamlNode *yaml.Node) error {
	log.Debugf("Checking Document Queries for [%q]", overlay.Name)

	if ok := overlay.DocumentQuery.checkQueries(yamlNode); !ok {
		return nil
	}

	results := overlay.Query.Find(yamlNode)

	if results == nil {
		log.Debugf("No results found checking onMissing")

		return overlay.onMissing(yamlNode)
	}

	return overlay.doAction(yamlNode, results)
}

func (overlay *Overlay) CheckDocumentIndex(currentIndex int) bool {
	if overlay.DocumentIndex != nil {
		for _, documentIndex := range overlay.DocumentIndex {
			if currentIndex == documentIndex {
				return true
			}
		}

		return false
	}

	return true
}

func (overlay *Overlay) doAction(root *yaml.Node, yamlNodes []*yaml.Node) error {
	for _, yamlNode := range yamlNodes {
		log.Debugf("applying overlay [%q], %s at line %d column %d\n", overlay.Name, overlay.Action, yamlNode.Line, yamlNode.Column)

		switch overlay.Action {
		case actions.Delete:
			actions.DeleteNode(root, yamlNode)
		case actions.Replace:
			if err := actions.ReplaceNode(yamlNode, &overlay.Value); err != nil {
				return fmt.Errorf("%w, skipping replace on line %d column %d", err, yamlNode.Line, yamlNode.Column)
			}
		case actions.Combine:
			if err := actions.CombineNode(yamlNode, &overlay.Value); err != nil {
				return fmt.Errorf("%w, skipping combine on line %d, column %d", err, yamlNode.Line, yamlNode.Column)
			}
		default:
			if err := actions.MergeNode(yamlNode, &overlay.Value); err != nil {
				return fmt.Errorf("%w, skipping merge on line %d, column %d", err, yamlNode.Line, yamlNode.Column)
			}
		}
	}

	return nil
}

func (overlay *Overlay) UnmarshalYAML(value *yaml.Node) error {
	type decodedOverlay Overlay

	if err := value.Decode((*decodedOverlay)(overlay)); err != nil {
		return fmt.Errorf("failed to unmarshal overlay due to %w at line %d column %d", err, value.Line, value.Column)
	}

	switch "" {
	case overlay.Name:
		overlay.Name = overlay.Query.String()
	case overlay.Action.String():
		overlay.Action = actions.Merge
	}

	return nil
}
