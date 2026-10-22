// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/instructions"
	"gopkg.in/yaml.v3"
)

// buildDocNode parses a single YAML document string into a DocumentNode.
func buildDocNode(t *testing.T, src string) *yaml.Node {
	t.Helper()

	var doc yaml.Node

	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("buildDocNode: %v", err)
	}

	return &doc
}

// TestDocumentSeparators exercises the two separator bugs fixed together:
//
//  1. encodeNodes: when the first document node was emptied by a delete overlay,
//     the subsequent (first real) document was missing its leading "---\n".
//
//  2. openOutputFile: multiple YamlFiles sharing the same outputPath were each
//     opened with O_TRUNC, so the last write silently discarded all prior content.
//
// The test constructs node state that matches the real-world RBAC scenario:
// a two-doc source file where the first doc is deleted (Namespace), leaving only
// a ServiceAccount, combined with a separate RBAC file (ClusterRole + ClusterRoleBinding)
// targeting the same output path.
//
// Does not call t.Parallel(): PostProcessHandler reads the global viper config
// (encodeNodes reads "indentLevel"/"outputStyle"/"stdout"), which races with any
// other test in this package that also touches viper. See TestExecute.
func TestDocumentSeparators(t *testing.T) { //nolint:paralleltest // must not run concurrently; see comment above
	outputFile := filepath.Join(t.TempDir(), "combined.yaml")

	// Simulate ns-and-sa.yaml after a delete overlay removes the Namespace document.
	// nodes[0] is an empty DocumentNode (Namespace deleted); nodes[1] is the SA.
	emptyDoc := &yaml.Node{Kind: yaml.DocumentNode} // Content == nil, len == 0
	saDoc := buildDocNode(t, `apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-sa
  namespace: test-system`)

	// Simulate rbac.yaml with ClusterRole and ClusterRoleBinding documents.
	crDoc := buildDocNode(t, `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-role
rules: []`)
	crbDoc := buildDocNode(t, `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: test-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: test-role
subjects: []`)

	// OutputPath is absolute, so openOutputFile does not consult viper for the directory.
	pChan := make(chan *instructions.YamlFile, 2)

	pChan <- &instructions.YamlFile{
		Path:       "ns-and-sa.yaml",
		OutputPath: outputFile,
		Nodes:      []*yaml.Node{emptyDoc, saDoc},
	}

	pChan <- &instructions.YamlFile{
		Path:       "rbac.yaml",
		OutputPath: outputFile,
		Nodes:      []*yaml.Node{crDoc, crbDoc},
	}

	close(pChan)

	if err := instructions.PostProcessHandler(&instructions.Config{}, pChan); err != nil {
		t.Fatalf("PostProcessHandler() error = %v", err)
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	// Case 1: the first real document (SA) must have a leading "---\n" even though
	// it was preceded by an empty node that was skipped.
	if !bytes.HasPrefix(content, []byte("---\n")) {
		t.Errorf("output does not start with '---\\n':\n%s", content)
	}

	// Case 2: all three documents (SA + CR + CRB) must be present -- not just
	// whichever YamlFile happened to write last.
	dec := yaml.NewDecoder(bytes.NewReader(content))

	var docCount int

	for {
		var node yaml.Node

		if err := dec.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			t.Fatalf("decode document %d: %v", docCount, err)
		}

		docCount++
	}

	const wantDocs = 3 // SA + CR + CRB; Namespace was deleted

	if docCount != wantDocs {
		t.Errorf("expected %d documents in output, got %d:\n%s", wantDocs, docCount, content)
	}
}
