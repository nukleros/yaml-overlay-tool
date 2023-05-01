// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/op/go-logging"
	"gopkg.in/yaml.v3"
)

var log = logging.MustGetLogger("overlays") //nolint:gochecknoglobals

// ErrInvalidAction occurs when user passes a action that is not one of merge, replace, delete, combine.
var ErrInvalidAction = errors.New("invalid overlay action")

type Action int

const (
	// Invalid overlay action.
	Invalid = iota
	// Merge overlay action.
	Merge
	// Replace overlay action.
	Replace
	// Delete overlay action.
	Delete
	// Combine overlay action.
	Combine
)

func (action Action) String() string {
	toString := map[Action]string{
		Invalid: "",
		Merge:   "merge",
		Replace: "replace",
		Delete:  "delete",
		Combine: "combine",
	}

	return toString[action]
}

func (action *Action) UnmarshalYAML(yamlNode *yaml.Node) error {
	var decodedYaml string

	if err := yamlNode.Decode(&decodedYaml); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, yamlNode.Line, yamlNode.Column)
	}

	decodedYaml = strings.ToLower(decodedYaml)

	toID := map[string]Action{
		"":        Invalid,
		"merge":   Merge,
		"replace": Replace,
		"delete":  Delete,
		"combine": Combine,
	}

	if toID[decodedYaml] == Invalid {
		return fmt.Errorf("%w, %s", ErrInvalidAction, decodedYaml)
	}

	*action = toID[decodedYaml]

	return nil
}

func (action Action) MarshalYAML() (interface{}, error) {
	return action.String(), nil
}

func (action *Action) Set(value string) error {
	if err := yaml.Unmarshal([]byte(value), action); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (action *Action) Type() string {
	return "actions.Action"
}

type OnMissingAction int

const (
	// Default onMissing action. settable via viepr config, defaults to ignore.
	Default = iota
	// Ignore onMissing action.
	Ignore
	// Inject onMissing action.
	Inject
)

func (action OnMissingAction) String() string {
	toString := map[OnMissingAction]string{
		Default: "default",
		Ignore:  "ignore",
		Inject:  "inject",
	}

	return toString[action]
}

func (action *OnMissingAction) UnmarshalYAML(value *yaml.Node) error {
	var decodedYaml string

	if err := value.Decode(&decodedYaml); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, value.Line, value.Column)
	}

	decodedYaml = strings.ToLower(decodedYaml)

	toID := map[string]OnMissingAction{
		"default": Default,
		"ignore":  Ignore,
		"inject":  Inject,
	}

	*action = toID[decodedYaml]

	return nil
}

func (action OnMissingAction) MarshalYAML() (interface{}, error) {
	return action.String(), nil
}

func (action *OnMissingAction) Set(val string) error {
	if err := yaml.Unmarshal([]byte(val), action); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (action *OnMissingAction) Type() string {
	return "actions.OnMissingAction"
}
