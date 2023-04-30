// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package overlays

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"github.com/nukleros/yaml-overlay-tool/internal/builder"
)

var (
	// ErrOnMissingNoInjectAction occurrs if there are not matches found and no inject action present.
	ErrOnMissingNoInjectAction = errors.New("no matches and no onMissing.action of 'inject'")

	// ErrOnMissingNoInjectPath occurrs if there are no matches found for the injectPath.
	ErrOnMissingNoInjectPath = errors.New("no matches and no onMissing.injectPath")
)

type OnMissing struct {
	Action     actions.OnMissingAction `yaml:"action,omitempty"`
	InjectPath Queries                 `yaml:"injectPath,omitempty"`
}

func (overlay *Overlay) onMissing(yamlNode *yaml.Node) error {
	if overlay.OnMissing.Action == actions.Default {
		if strings.EqualFold(viper.GetString("defaultOnMissingAction"), "inject") {
			overlay.OnMissing.Action = actions.Inject
		}
	}

	// check if the query has a match
	// if no match then we require an inject path
	// we need to then check if each inject path is valid (does it exist)
	// if we had an inject path(s) then we inject the value to those locations
	// if we didn't have an inject path we have an implicit onMissing: ignore and we put out a warning if not stdout option to terminal
	switch overlay.OnMissing.Action {
	case actions.Inject:
		return overlay.handleInjectPath(yamlNode)
	default:
		log.Debugf("ignoring %s at %s due to %s\n", overlay.Action, overlay.Query, ErrOnMissingNoInjectAction)

		return nil
	}
}

func (overlay *Overlay) doInjectPath(injectPath Queries, yamlNode *yaml.Node) error {
	buildYamlPaths, err := builder.NewPaths(injectPath.Paths())
	if err != nil {
		return fmt.Errorf("failed to build inject path %s, %w", injectPath, err)
	}

	updatedYaml, _ := buildYamlPaths.BuildPaths()

	err = actions.MergeNode(yamlNode, updatedYaml)
	if err != nil {
		return fmt.Errorf("failed to merge injectpath scaffolding %s with document, %w", injectPath, err)
	}

	results := injectPath.Find(yamlNode)

	for _, result := range results {
		if err := actions.ReplaceNode(result, &overlay.Value); err != nil {
			return fmt.Errorf("%w for onMissing.InjectPath", err)
		}
	}

	return nil
}

func (overlay *Overlay) handleInjectPath(yamlNode *yaml.Node) error {
	_, err := builder.NewPaths(overlay.Query.Paths())
	if err != nil {
		if errors.Is(err, builder.ErrInvalidPathSyntax) {
			if overlay.OnMissing.InjectPath == nil {
				log.Debugf("ignoring %s at %s due to %s\n", overlay.Action, overlay.Query, ErrOnMissingNoInjectPath)

				return nil
			}

			return overlay.doInjectPath(overlay.OnMissing.InjectPath, yamlNode)
		}

		return fmt.Errorf("%w, for onMissing", err)
	}

	return overlay.doInjectPath(overlay.Query, yamlNode)
}
