// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type Style uint

type Styles []Style

const (
	// TaggedStyle chooses YAML Tag style for rendering.
	TaggedStyle Style = 1 << iota
	// DoubleQuotedStyle chooses YAML Double Quoted style for rendering.
	DoubleQuotedStyle
	// SingleQuotedStyle chooses YAML Single Quoted style for rendering.
	SingleQuotedStyle
	// LiteralStyle chooses use YAML Literal style for rendering.
	LiteralStyle
	// FoldedStyle chooses use YAML Folded style for rendering.
	FoldedStyle
	// FlowStyle chooses YAML Flow style for rendering.
	FlowStyle
	// NormalStyle chooses YAML Normal style for rendering.
	NormalStyle
)

func (style Style) String() string {
	toString := map[Style]string{
		NormalStyle:       "normal",
		TaggedStyle:       "tagged",
		DoubleQuotedStyle: "doubleQuoted",
		SingleQuotedStyle: "singleQuoted",
		LiteralStyle:      "literal",
		FoldedStyle:       "folded",
		FlowStyle:         "flow",
	}

	return toString[style]
}

func (styles Styles) FlagMap() map[Style][]string {
	return map[Style][]string{
		NormalStyle:       {"normal", "n"},
		TaggedStyle:       {"tagged", "tag", "t"},
		DoubleQuotedStyle: {"doubleQuoted", "doubleQuote", "double", "dq"},
		SingleQuotedStyle: {"singleQuoted", "singleQoute", "single", "sq"},
		LiteralStyle:      {"literal", "l"},
		FoldedStyle:       {"folded", "fold", "fo"},
		FlowStyle:         {"flow", "fl"},
	}
}

func (styles Styles) GetStyle() Style {
	var style Style
	for _, v := range styles {
		style += v
	}

	return style
}

func SetStyle(styles Styles, yamlNode ...*yaml.Node) {
	style := styles.GetStyle()
	setStyle(style, yamlNode...)
}

func setStyle(style Style, yamlNode ...*yaml.Node) {
	for _, nodeValue := range yamlNode {
		switch nodeValue.Kind {
		case yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode, yaml.AliasNode:
			nodeValue.Style = yaml.Style(style)

			setStyle(style, nodeValue.Content...)
		case yaml.ScalarNode:
			nodeValue.Style = yaml.Style(style)
		}
	}
}

func GetStyleFromConfig(configValue ...string) Styles {
	var result Styles

	for _, styles := range configValue {
		stylesSplit := strings.Split(strings.Trim(strings.ToLower(styles), "[]"), ",")
		for _, style := range stylesSplit {
			switch style {
			case "normal", "n":
				result = append(result, NormalStyle)
			case "tagged", "tag", "t":
				result = append(result, TaggedStyle)
			case "doublequoted", "doublequote", "double", "dq":
				result = append(result, DoubleQuotedStyle)
			case "singlequoted", "singlequote", "single", "sq":
				result = append(result, SingleQuotedStyle)
			case "literal", "l":
				result = append(result, LiteralStyle)
			case "folded", "fold", "fo":
				result = append(result, FoldedStyle)
			case "flow", "fl":
				result = append(result, FlowStyle)
			}
		}
	}

	return result
}
