// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions_test

import (
	"strings"
	"testing"

	"github.com/vmware-tanzu-labs/yaml-overlay-tool/internal/actions"
	"gopkg.in/yaml.v3"
)

// encodeNode is a helper that encodes a yaml.Node to a string.
func encodeNode(n *yaml.Node) string {
	var buf strings.Builder

	enc := yaml.NewEncoder(&buf)
	_ = enc.Encode(n)
	_ = enc.Close()

	return strings.TrimSpace(strings.TrimPrefix(buf.String(), "---"))
}

// TestSetStylePreservesYAML11BoolStrings verifies that SetStyle does not cause
// YAML 1.1-only boolean keywords (on/off/yes/no/y/n) tagged as !!str to lose
// their quoting when re-encoded.  go-yaml v3 quotes YAML 1.2 booleans
// (true/false) automatically, but silently drops quotes for these YAML 1.1-only
// keywords when Style == 0, resulting in values that round-trip as booleans.
func TestSetStylePreservesYAML11BoolStrings(t *testing.T) {
	t.Parallel()

	const (
		wantOn  = `"on"`
		wantOff = `"off"`
		wantYes = `"yes"`
		wantNo  = `"no"`
		wantKey = "key"
	)

	cases := []struct {
		input   string
		wantKey string
		wantVal string
	}{
		{`key: "on"`, wantKey, wantOn},
		{`key: "off"`, wantKey, wantOff},
		{`key: "yes"`, wantKey, wantYes},
		{`key: "no"`, wantKey, wantNo},
		{`key: "y"`, wantKey, `"y"`},
		{`key: "n"`, wantKey, `"n"`},
		{`key: !!str on`, wantKey, wantOn},
		{`key: !!str off`, wantKey, wantOff},
		{`key: !!str yes`, wantKey, wantYes},
		{`key: !!str no`, wantKey, wantNo},
		// YAML 1.2 booleans: go-yaml already handles these; verify no regression.
		{`key: "true"`, wantKey, `"true"`},
		{`key: "false"`, wantKey, `"false"`},
	}

	stylesToTest := []struct {
		name   string
		styles actions.Styles
	}{
		{"NormalStyle", actions.Styles{actions.NormalStyle}},
		{"EmptyStyles (GetStyle==0)", actions.Styles{}},
	}

	for _, st := range stylesToTest {
		st := st
		t.Run(st.name, func(t *testing.T) {
			t.Parallel()

			for _, tc := range cases {
				tc := tc
				t.Run(tc.input, func(t *testing.T) {
					t.Parallel()

					var doc yaml.Node
					if err := yaml.Unmarshal([]byte(tc.input), &doc); err != nil {
						t.Fatalf("failed to parse: %v", err)
					}

					actions.SetStyle(st.styles, &doc)

					got := strings.TrimSpace(encodeNode(&doc))
					want := tc.wantKey + ": " + tc.wantVal

					if got != want {
						t.Errorf("SetStyle(%s) on %q:\n  got:  %s\n  want: %s", st.name, tc.input, got, want)
					}
				})
			}
		})
	}
}
