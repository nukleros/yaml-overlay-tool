// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"slices"
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
)

// Paths and overlay names used to follow overlays through a merge.
const (
	listedPath   = "/listed.yaml"
	adHocPath    = "/ad-hoc.yaml"
	firstCommon  = "first-common"
	secondCommon = "second-common"
	adHocName    = "ad-hoc"
	commonName   = "common"
)

// namedOverlay returns an overlay that is identified only by its name.
func namedOverlay(name string) *overlays.Overlay {
	return &overlays.Overlay{Name: name}
}

// overlayNames returns the names of the overlays, in order.
func overlayNames(overlayList []*overlays.Overlay) []string {
	names := make([]string, 0, len(overlayList))

	for _, overlay := range overlayList {
		names = append(names, overlay.Name)
	}

	return names
}

// TestMergeInstructionSets checks that every instructions file becomes one layer on every
// YAML file, in order, and that YAML files keep the order they were first listed in.
func TestMergeInstructionSets(t *testing.T) {
	t.Parallel()

	documentOverlay := &Document{Path: 0, Overlays: []*overlays.Overlay{namedOverlay("first-document")}}

	first := &Instructions{
		CommonOverlays: []*overlays.Overlay{namedOverlay(firstCommon)},
		YamlFiles: YamlFiles{
			{
				Path:      listedPath,
				Overlays:  []*overlays.Overlay{namedOverlay("first-file")},
				Documents: []*Document{documentOverlay},
			},
		},
	}

	second := &Instructions{
		CommonOverlays: []*overlays.Overlay{namedOverlay(secondCommon)},
	}

	adHocFiles := YamlFiles{{Path: adHocPath, OutputPath: stdInOutputPath}}

	merged := mergeInstructionSets([]*Instructions{first, second}, adHocFiles)

	if len(merged) != 2 || merged[0].Path != listedPath || merged[1].Path != adHocPath {
		t.Fatalf("merged paths = %v, want [/listed.yaml /ad-hoc.yaml]", merged)
	}

	if merged[1].OutputPath != stdInOutputPath {
		t.Errorf("ad hoc output path = %q, want stdin.yaml", merged[1].OutputPath)
	}

	wantLayers := map[string][][]string{
		listedPath: {{firstCommon, "first-file"}, {secondCommon}},
		adHocPath:  {{firstCommon}, {secondCommon}},
	}

	for _, yamlFile := range merged {
		gotLayers := make([][]string, 0, len(yamlFile.layers))

		for _, layer := range yamlFile.layers {
			gotLayers = append(gotLayers, overlayNames(layer.overlays))
		}

		if !slices.EqualFunc(gotLayers, wantLayers[yamlFile.Path], slices.Equal[[]string]) {
			t.Errorf("%s layers = %v, want %v", yamlFile.Path, gotLayers, wantLayers[yamlFile.Path])
		}
	}

	listedFirstLayer := merged[0].layers[0]
	if len(listedFirstLayer.documents) != 1 || listedFirstLayer.documents[0] != documentOverlay {
		t.Errorf("document overlays were not kept in the first instructions file's layer")
	}
}

// TestAppendAdHocOverlay checks that the --query overlay runs as the last common overlay
// of the last instructions file.
func TestAppendAdHocOverlay(t *testing.T) {
	t.Parallel()

	adHoc := namedOverlay(adHocName)

	t.Run("with instructions files", func(t *testing.T) {
		t.Parallel()

		sets := []*Instructions{
			{CommonOverlays: []*overlays.Overlay{namedOverlay(firstCommon)}},
			{CommonOverlays: []*overlays.Overlay{namedOverlay(secondCommon)}},
		}

		sets = appendAdHocOverlay(sets, adHoc)

		if got := overlayNames(sets[1].CommonOverlays); !slices.Equal(got, []string{secondCommon, adHocName}) {
			t.Errorf("last set common overlays = %v, want [second-common ad-hoc]", got)
		}

		if got := overlayNames(sets[0].CommonOverlays); !slices.Equal(got, []string{firstCommon}) {
			t.Errorf("first set common overlays = %v, want [first-common]", got)
		}
	})

	t.Run("without instructions files", func(t *testing.T) {
		t.Parallel()

		sets := appendAdHocOverlay(nil, adHoc)

		if len(sets) != 1 || !slices.Equal(overlayNames(sets[0].CommonOverlays), []string{adHocName}) {
			t.Errorf("sets = %v, want one set with the ad hoc overlay", sets)
		}
	})
}

// TestLayersDoNotShareOverlays checks that building the layer for one YAML file cannot
// change the layer of another, even when the common overlays slice has spare capacity
// (as it does after the --query overlay is appended).
func TestLayersDoNotShareOverlays(t *testing.T) {
	t.Parallel()

	common := make([]*overlays.Overlay, 0, 4)
	common = append(common, namedOverlay(commonName))

	set := &Instructions{
		CommonOverlays: common,
		YamlFiles: YamlFiles{
			{Path: "/one.yaml", Overlays: []*overlays.Overlay{namedOverlay("one")}},
			{Path: "/two.yaml", Overlays: []*overlays.Overlay{namedOverlay("two")}},
		},
	}

	layerOne := set.layerFor("/one.yaml")
	layerTwo := set.layerFor("/two.yaml")

	if got := overlayNames(layerOne.overlays); !slices.Equal(got, []string{commonName, "one"}) {
		t.Errorf("layer for /one.yaml = %v, want [common one]", got)
	}

	if got := overlayNames(layerTwo.overlays); !slices.Equal(got, []string{commonName, "two"}) {
		t.Errorf("layer for /two.yaml = %v, want [common two]", got)
	}
}
