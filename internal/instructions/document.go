// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import "github.com/nukleros/yaml-overlay-tool/internal/overlays"

// Document is a set of overlays scoped to one document within a multi-document YAML file.
type Document struct {
	// Name is an optional description, for organization purposes only.
	Name string `yaml:"name,omitempty"`
	// Path is the zero-based index of the document within the YAML file.
	Path int `yaml:"path,omitempty"`
	// Overlays apply only to the document at Path.
	Overlays []*overlays.Overlay `yaml:"overlays,omitempty"`
}

// checkDocumentIndex reports whether docIndex is the document this entry targets.
func (document *Document) checkDocumentIndex(docIndex int) bool {
	return document.Path == docIndex
}
