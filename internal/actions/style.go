// Copyright 2021 VMware, Inc.
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

// Canonical style names and short aliases, shared between String(), FlagMap(),
// and GetStyleFromConfig() so they can't drift out of sync with each other.
const (
	styleNormal       = "normal"
	styleTagged       = "tagged"
	styleDoubleQuoted = "doubleQuoted"
	styleSingleQuoted = "singleQuoted"
	styleLiteral      = "literal"
	styleFolded       = "folded"
	styleFlow         = "flow"

	styleAliasTag    = "tag"
	styleAliasDouble = "double"
	styleAliasSingle = "single"
	styleAliasFolded = "fo"
	styleAliasFlow   = "fl"

	styleAliasDoubleQuoted = "dq"
	styleAliasSingleQuoted = "sq"
	styleAliasFold         = "fold"
)

func (s Style) String() string {
	toString := map[Style]string{
		NormalStyle:       styleNormal,
		TaggedStyle:       styleTagged,
		DoubleQuotedStyle: styleDoubleQuoted,
		SingleQuotedStyle: styleSingleQuoted,
		LiteralStyle:      styleLiteral,
		FoldedStyle:       styleFolded,
		FlowStyle:         styleFlow,
	}

	return toString[s]
}

func (ss Styles) FlagMap() map[Style][]string {
	return map[Style][]string{
		NormalStyle:       {styleNormal, "n"},
		TaggedStyle:       {styleTagged, styleAliasTag, "t"},
		DoubleQuotedStyle: {styleDoubleQuoted, "doubleQuote", styleAliasDouble, styleAliasDoubleQuoted},
		SingleQuotedStyle: {styleSingleQuoted, "singleQoute", styleAliasSingle, styleAliasSingleQuoted},
		LiteralStyle:      {styleLiteral, "l"},
		FoldedStyle:       {styleFolded, styleAliasFold, styleAliasFolded},
		FlowStyle:         {styleFlow, styleAliasFlow},
	}
}

func (ss Styles) GetStyle() Style {
	var s Style
	for _, v := range ss {
		s += v
	}

	return s
}

// yaml11OnlyBooleans are YAML 1.1-only keywords that go-yaml v3's encoder does
// not automatically quote when tag is !!str and style is 0.  (YAML 1.2 keywords
// such as true/false/null/~ are already handled by the encoder itself.)
var yaml11OnlyBooleans = map[string]bool{ //nolint:gochecknoglobals
	"y": true, "Y": true, "yes": true, "Yes": true, "YES": true,
	"n": true, "N": true, "no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
}

func SetStyle(ss Styles, n ...*yaml.Node) {
	s := ss.GetStyle()
	setStyle(s, n...)
}

func setStyle(s Style, n ...*yaml.Node) {
	for _, nv := range n {
		switch nv.Kind {
		case yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode, yaml.AliasNode:
			nv.Style = yaml.Style(s) //nolint:gosec // s is a sum of at most 7 known power-of-two flags (max 127), well within uint32 range

			setStyle(s, nv.Content...)
		case yaml.ScalarNode:
			newStyle := yaml.Style(s) //nolint:gosec // s is a sum of at most 7 known power-of-two flags (max 127), well within uint32 range
			// go-yaml v3 does not quote YAML 1.1-only boolean keywords (on/off/yes/no/y/n)
			// when tag is !!str and style is 0; the encoder only auto-quotes YAML 1.2 keywords.
			// Force double-quoting so the string type survives a round-trip.
			if nv.Tag == "!!str" && yaml11OnlyBooleans[nv.Value] &&
				newStyle&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle) == 0 {
				newStyle = yaml.DoubleQuotedStyle
			}

			nv.Style = newStyle
		}
	}
}

func GetStyleFromConfig(v ...string) Styles {
	var result Styles

	for _, ss := range v {
		ssp := strings.Split(strings.Trim(strings.ToLower(ss), "[]"), ",")
		for _, s := range ssp {
			switch s {
			case styleNormal, "n":
				result = append(result, NormalStyle)
			case styleTagged, styleAliasTag, "t":
				result = append(result, TaggedStyle)
			case "doublequoted", "doublequote", styleAliasDouble, styleAliasDoubleQuoted:
				result = append(result, DoubleQuotedStyle)
			case "singlequoted", "singlequote", styleAliasSingle, styleAliasSingleQuoted:
				result = append(result, SingleQuotedStyle)
			case styleLiteral, "l":
				result = append(result, LiteralStyle)
			case styleFolded, styleAliasFold, styleAliasFolded:
				result = append(result, FoldedStyle)
			case styleFlow, styleAliasFlow:
				result = append(result, FlowStyle)
			}
		}
	}

	return result
}
