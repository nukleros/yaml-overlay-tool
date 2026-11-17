// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ErrFoundDirectoryWithPathOutput occurs when a yamlFiles entry points at a directory but
// its outputPath is a file name.
var ErrFoundDirectoryWithPathOutput = errors.New("found directory as input path with file as output path")

// YamlFile is a YAML file to apply overlays to, along with the overlays specific to it.
type YamlFile struct {
	// Name is an optional description, for organization purposes only.
	Name string `yaml:"name,omitempty"`
	// Overlays apply to every document in this YAML file.
	Overlays []*overlays.Overlay `yaml:"overlays,omitempty"`
	// Documents hold overlays for specific documents within this YAML file.
	Documents []*Document `yaml:"documents,omitempty"`
	// Nodes are the YAML documents read from Path, one per document in the file.
	Nodes []*yaml.Node `yaml:"-"`
	// Path is the YAML file or directory to process, or "-" for stdin. A relative path is
	// resolved against the directory of the instructions file it appears in.
	Path string `yaml:"path,omitempty"`
	// OutputPath optionally changes where the overlaid file is written.
	OutputPath string `yaml:"outputPath,omitempty"`

	// layers are the overlays that will actually be applied, one layer per instructions
	// file, in the order those files were given. They are built by mergeInstructionSets.
	// Overlays and Documents above are only what one instructions file listed.
	layers []*overlayLayer
}

// YamlFiles is a list of YAML files.
type YamlFiles []*YamlFile

// queueOverlays sends every overlay for this YAML file to the work stream in the order
// they must be applied, then closes the stream.
func (yamlFile *YamlFile) queueOverlays(stream *overlays.WorkStream) {
	for nodeIndex, node := range yamlFile.Nodes {
		for _, layer := range yamlFile.layers {
			layer.queue(stream, node, nodeIndex, yamlFile.Path)
		}
	}

	stream.CloseStream()
}

// UnmarshalYAML decodes the yamlFiles list of an instructions file. Directories are
// expanded into the YAML files they contain and duplicate paths are merged. The files
// themselves are read later, after all instructions files have been merged, so that a
// file named in several instructions files is read only once.
func (yamlFiles *YamlFiles) UnmarshalYAML(value *yaml.Node) error {
	var decoded []*YamlFile

	if err := value.Decode(&decoded); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, value.Line, value.Column)
	}

	files := YamlFiles(decoded)
	if err := files.expandDirectories(); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, value.Line, value.Column)
	}

	files.mergeDuplicates()

	*yamlFiles = files

	return nil
}

// readYamlFiles reads the YAML documents of every file in the list.
func (yamlFiles YamlFiles) readYamlFiles() error {
	for _, yamlFile := range yamlFiles {
		if err := yamlFile.readYamlFile(); err != nil {
			return err
		}
	}

	return nil
}

// readYamlFile reads every YAML document in the file (or stdin) into Nodes.
func (yamlFile *YamlFile) readYamlFile() error {
	reader, err := ReadStream(yamlFile.Path)
	if err != nil {
		return err
	}

	if file, isFile := reader.(*os.File); isFile {
		defer CloseFile(file)
	}

	decoder := yaml.NewDecoder(reader)

	for {
		var document yaml.Node

		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", yamlFile.Path, err)
		}

		yamlFile.Nodes = append(yamlFile.Nodes, &document)
	}
}

// removeComments strips head, line and foot comments from every document of every file.
func (yamlFiles YamlFiles) removeComments() {
	for _, yamlFile := range yamlFiles {
		for _, node := range yamlFile.Nodes {
			removeCommentsFromNode(node)
		}
	}
}

// expandDirectories turns each relative path into an absolute one and replaces every
// directory entry with one entry per YAML file found beneath it, recursively. Each new
// entry gets its own copy of the directory entry's overlays, documents and outputPath.
//
// Relative paths are resolved against viper's instructionsDir key: the directory of the
// instructions file being decoded. Files found in directories are placed after all of the
// explicitly listed files, which affects which entry mergeDuplicates keeps first.
func (yamlFiles *YamlFiles) expandDirectories() error {
	listed := make(YamlFiles, 0, len(*yamlFiles))

	var fromDirectories YamlFiles

	for _, yamlFile := range *yamlFiles {
		if yamlFile.Path == stdInPath {
			listed = append(listed, yamlFile)

			continue
		}

		if !path.IsAbs(yamlFile.Path) {
			yamlFile.Path = path.Join(viper.GetString(instructionsDirKey), yamlFile.Path)
		}

		isDir, err := isDirectory(yamlFile.Path)
		if err != nil {
			return err
		}

		if !isDir {
			listed = append(listed, yamlFile)

			continue
		}

		expanded, err := yamlFile.expandDirectory()
		if err != nil {
			return err
		}

		fromDirectories = append(fromDirectories, expanded...)
	}

	listed = append(listed, fromDirectories...)
	*yamlFiles = listed

	return nil
}

// expandDirectory returns one entry per YAML file beneath this entry's directory, each
// carrying its own copy of this entry's overlays, documents and outputPath.
func (yamlFile *YamlFile) expandDirectory() (YamlFiles, error) {
	// Every file in the directory keeps its own name, so the output must be a directory too.
	if path.Ext(yamlFile.OutputPath) != "" {
		return nil, fmt.Errorf("%w, provide a directory for outputPath instead", ErrFoundDirectoryWithPathOutput)
	}

	fileNames, err := getFileNames(yamlFile.Path)
	if err != nil {
		return nil, err
	}

	expanded := make(YamlFiles, 0, len(fileNames))

	for _, fileName := range fileNames {
		expanded = append(expanded, &YamlFile{
			Name:       yamlFile.Name,
			Overlays:   slices.Clone(yamlFile.Overlays),
			Documents:  slices.Clone(yamlFile.Documents),
			Path:       fileName,
			OutputPath: yamlFile.OutputPath,
		})
	}

	return expanded, nil
}

// mergeDuplicates combines entries that share a path into the first such entry. Their
// overlays and documents are appended in order, and the last non-empty outputPath wins.
func (yamlFiles *YamlFiles) mergeDuplicates() {
	merged := make(YamlFiles, 0, len(*yamlFiles))
	byPath := make(map[string]*YamlFile, len(*yamlFiles))

	for _, yamlFile := range *yamlFiles {
		existing, found := byPath[yamlFile.Path]
		if !found {
			byPath[yamlFile.Path] = yamlFile
			merged = append(merged, yamlFile)

			continue
		}

		// slices.Concat always allocates, so entries never share a backing array.
		existing.Overlays = slices.Concat(existing.Overlays, yamlFile.Overlays)
		existing.Documents = slices.Concat(existing.Documents, yamlFile.Documents)

		if yamlFile.OutputPath != "" {
			existing.OutputPath = yamlFile.OutputPath
		}
	}

	*yamlFiles = merged
}

// removeCommentsFromNode walks the tree of YAML nodes and removes all comments.
func removeCommentsFromNode(node *yaml.Node) {
	if node.Content == nil {
		return
	}

	for _, child := range node.Content {
		child.HeadComment = ""
		child.LineComment = ""
		child.FootComment = ""

		removeCommentsFromNode(child)
	}
}
