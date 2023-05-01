// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rwtodd/Go.Sed/sed"
	"gopkg.in/yaml.v3"
)

func format(formatStr string, values ...interface{}) string {
	values = sanitizeValues(values...)

	return doFormat(formatStr, values...)
}

func sanitizeNode(yamlNode ...*yaml.Node) {
	for _, nodeValue := range yamlNode {
		if nodeValue == nil {
			continue
		}

		switch nodeValue.Kind {
		case yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode, yaml.AliasNode:
			sanitizeNode(nodeValue.Content...)
		case yaml.ScalarNode:
		}

		values := []*string{&nodeValue.Value, &nodeValue.HeadComment, &nodeValue.LineComment, &nodeValue.FootComment}
		for _, value := range values {
			*value = format(*value, "", "", "", "", "")
		}
	}
}

func sanitizeValues(values ...interface{}) []interface{} {
	for _, t := range []string{
		ValueMarker,
		LineCommentMarker,
		HeadCommentMarker,
		FootCommentMarker,
		KeyMarker,
	} {
		for j := range values {
			//nolint:forcetypeassert
			values[j] = strings.ReplaceAll(values[j].(string), t, "")
		}
	}

	return values
}

func doFormat(formatString string, values ...interface{}) string {
	markers := []string{
		ValueMarker,
		LineCommentMarker,
		HeadCommentMarker,
		FootCommentMarker,
		KeyMarker,
	}

	valueMap := map[string]string{}

	for i, value := range markers {
		if stringValue, ok := values[i].(string); ok {
			valueMap[value] = stringValue
		}
	}

	formatMarkerRegex := regexp.MustCompile(`(?P<marker>%[vklfh])(?P<format>{(?P<command>.*?)})?`)

	formatMarkerMatches := formatMarkerRegex.FindAllStringSubmatch(formatString, -1)

	for _, match := range formatMarkerMatches {
		result := make(map[string]string)

		for i, name := range formatMarkerRegex.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		if result["marker"] != "" {
			marker := valueMap[result["marker"]]

			if result["command"] != "" {
				var err error

				marker, err = processSedCommand(result["command"], valueMap[result["marker"]])
				if err != nil {
					log.Warningf("Skipping additional format on [%s] due to invalid sed exp [%s], %s", formatString, result["command"], err)
				}
			}

			formatString = strings.Replace(formatString, match[0], marker, 1)
		}
	}

	return formatString
}

func processSedCommand(sedCommand, sedResult string) (string, error) {
	sedEngine, err := sed.New(strings.NewReader(sedCommand))
	if err != nil {
		return sedResult, fmt.Errorf("sed error: %w", err)
	}

	sedResult, err = sedEngine.RunString(sedResult)
	if err != nil {
		return sedResult, fmt.Errorf("sed error: %w", err)
	}

	sedResult = strings.TrimSuffix(sedResult, "\n")
	sedResult = strings.TrimPrefix(sedResult, "\n")

	return sedResult, nil
}
