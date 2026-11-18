// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"slices"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
	"gopkg.in/yaml.v3"
)

// overlayLayer holds the overlays that one instructions file contributes to one YAML file.
//
// Layers are applied in the order the instructions files were given on the command line,
// and each layer is applied completely (common overlays, then file overlays, then
// document overlays) before the next layer starts. That is what lets a later
// instructions file win over an earlier one, even when the earlier file's overlay is
// more specific.
type overlayLayer struct {
	// overlays apply to every document in the YAML file, subject to each overlay's own
	// documentIndex and documentQuery qualifiers. They are the instructions file's
	// commonOverlays followed by the overlays it lists for this YAML file.
	overlays []*overlays.Overlay
	// documents hold overlays that apply to a single document in the YAML file.
	documents []*Document
}

// isEmpty reports whether the layer has no overlays to apply.
func (layer *overlayLayer) isEmpty() bool {
	return len(layer.overlays) == 0 && len(layer.documents) == 0
}

// queue sends this layer's overlays for the document at nodeIndex to the work stream:
// overlays for the whole file first, then overlays for that specific document.
func (layer *overlayLayer) queue(stream *overlays.WorkStream, node *yaml.Node, nodeIndex int, filePath string) {
	for _, overlay := range layer.overlays {
		if overlay.CheckDocumentIndex(nodeIndex) {
			stream.AddWorkload(overlay, node, nodeIndex, filePath)
		}
	}

	for _, document := range layer.documents {
		if !document.checkDocumentIndex(nodeIndex) {
			continue
		}

		for _, overlay := range document.Overlays {
			stream.AddWorkload(overlay, node, nodeIndex, filePath)
		}
	}
}

// mergeInstructionSets combines the instructions files, in the order given, and the ad hoc
// files from --path or stdin into one list of YAML files with one entry per path.
//
// Every instructions file becomes one overlay layer on every YAML file, because its
// commonOverlays apply to all YAML files, wherever they were listed. Layers with nothing
// to apply are skipped.
func mergeInstructionSets(instructionSets []*Instructions, adHocFiles YamlFiles) YamlFiles {
	index := newYamlFileIndex()

	for _, instructionSet := range instructionSets {
		for _, yamlFile := range instructionSet.YamlFiles {
			index.add(yamlFile)
		}
	}

	// Ad hoc files are added last so that, as before, their output path wins.
	for _, yamlFile := range adHocFiles {
		index.add(yamlFile)
	}

	for _, mergedFile := range index.ordered {
		for _, instructionSet := range instructionSets {
			layer := instructionSet.layerFor(mergedFile.Path)
			if !layer.isEmpty() {
				mergedFile.layers = append(mergedFile.layers, layer)
			}
		}
	}

	return index.ordered
}

// layerFor builds the overlay layer that these instructions contribute to the YAML file at
// yamlFilePath: the common overlays, then the overlays and document overlays listed for
// that path, if any.
func (instructions *Instructions) layerFor(yamlFilePath string) *overlayLayer {
	layer := &overlayLayer{
		// Cloned so that the layers of different YAML files never share a backing array,
		// where appending for one file could overwrite the overlays of another.
		overlays: slices.Clone(instructions.CommonOverlays),
	}

	for _, yamlFile := range instructions.YamlFiles {
		if yamlFile.Path == yamlFilePath {
			layer.overlays = append(layer.overlays, yamlFile.Overlays...)
			layer.documents = append(layer.documents, yamlFile.Documents...)
		}
	}

	return layer
}

// yamlFileIndex collects one merged YAML file per path, keeping the order in which each
// path was first seen so that output order matches the order files were listed.
type yamlFileIndex struct {
	ordered YamlFiles
	byPath  map[string]*YamlFile
}

// newYamlFileIndex returns an empty yamlFileIndex.
func newYamlFileIndex() *yamlFileIndex {
	return &yamlFileIndex{
		byPath: make(map[string]*YamlFile),
	}
}

// add records source under its path. Overlays are not copied here (see layerFor). The
// first non-empty name is kept and the last non-empty outputPath wins, matching how
// duplicate entries within one instructions file are merged.
func (index *yamlFileIndex) add(source *YamlFile) {
	merged, found := index.byPath[source.Path]
	if !found {
		merged = &YamlFile{Path: source.Path}
		index.byPath[source.Path] = merged
		index.ordered = append(index.ordered, merged)
	}

	if merged.Name == "" {
		merged.Name = source.Name
	}

	if source.OutputPath != "" {
		merged.OutputPath = source.OutputPath
	}
}
