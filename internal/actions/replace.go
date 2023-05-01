// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"fmt"

	"github.com/qdm12/reprint"
	"gopkg.in/yaml.v3"
)

func ReplaceNode(orignalYamlNode, replacementYamlNode *yaml.Node) error {
	originalValue := *orignalYamlNode

	if err := reprint.FromTo(replacementYamlNode, orignalYamlNode); err != nil {
		return fmt.Errorf("failed to replace value: %w", err)
	}

	mergeComments(&originalValue, orignalYamlNode)
	orignalYamlNode.HeadComment = originalValue.HeadComment
	orignalYamlNode.LineComment = originalValue.LineComment
	orignalYamlNode.FootComment = originalValue.FootComment

	return nil
}
