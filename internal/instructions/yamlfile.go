// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
)

var ErrFoundDirectoryWithPathOutput = errors.New("found directory as input path with file as output path")

// YamlFile is used to define which files should be manipulated and overlays specific to that file.
type YamlFile struct {
	// Optional Name to define for organization purposes.
	Name string `yaml:"name,omitempty"`
	// List of Overlays specific to this yamlFile.
	Overlays []*overlays.Overlay `yaml:"overlays,omitempty"`
	// a list of more specific entries and overlays for a specific document within the yamlFile.
	Documents []*Document `yaml:"documents,omitempty"`

	Nodes []*yaml.Node

	Path string `yaml:"path,omitempty"`

	OutputPath string `yaml:"outputPath,omitempty"`
}

type YamlFiles []*YamlFile

// queueOverlays sends all overlay jobs to the workstream for processing.
func (yamlFile *YamlFile) queueOverlays(stream *overlays.WorkStream) {
	for nodeIndex, yamlNode := range yamlFile.Nodes {
		for _, overlay := range yamlFile.Overlays {
			if ok := overlay.CheckDocumentIndex(nodeIndex); ok {
				stream.AddWorkload(overlay, yamlNode, nodeIndex, yamlFile.Path)
			}
		}

		for _, yamlDocument := range yamlFile.Documents {
			if ok := yamlDocument.checkDocumentIndex(nodeIndex); ok {
				for _, overlay := range yamlDocument.Overlays {
					stream.AddWorkload(overlay, yamlNode, nodeIndex, yamlFile.Path)
				}
			}
		}
	}

	stream.CloseStream()
}

func (yamlFiles *YamlFiles) UnmarshalYAML(yamlNode *yaml.Node) error {
	var yamlFilesStruct []*YamlFile

	if err := yamlNode.Decode(&yamlFilesStruct); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, yamlNode.Line, yamlNode.Column)
	}

	parsedYamlFiles := YamlFiles(yamlFilesStruct)
	if err := parsedYamlFiles.expandDirectories(); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, yamlNode.Line, yamlNode.Column)
	}

	parsedYamlFiles.mergeDuplicates()

	for _, yamlFile := range parsedYamlFiles {
		if err := yamlFile.readYamlFile(); err != nil {
			return err
		}
	}

	*yamlFiles = parsedYamlFiles

	return nil
}

func (yamlFile *YamlFile) readYamlFile() error {
	reader, err := ReadStream(yamlFile.Path)
	if err != nil {
		return err
	}

	yamlDecoder := yaml.NewDecoder(reader)

	for {
		var yamlNode yaml.Node

		if err := yamlDecoder.Decode(&yamlNode); errors.Is(err, io.EOF) {
			if reader, ok := reader.(*os.File); ok {
				CloseFile(reader)
			}

			break
		} else if err != nil {
			return fmt.Errorf("failed to read file %s: %w", yamlFile.Path, err)
		}

		yamlFile.Nodes = append(yamlFile.Nodes, &yamlNode)
	}

	return nil
}

func (yamlFiles *YamlFiles) expandDirectories() error {
	updatedYamlFiles := []*YamlFile(*yamlFiles)

	var yamlFilesInDir []string

	for i := 0; i <= len(updatedYamlFiles)-1; i++ {
		if updatedYamlFiles[i].Path == "-" {
			continue
		}

		if !path.IsAbs(updatedYamlFiles[i].Path) {
			updatedYamlFiles[i].Path = path.Join(viper.GetString("instructionsDir"), updatedYamlFiles[i].Path)
		}

		if ok, err := isDirectory(updatedYamlFiles[i].Path); err != nil {
			return err
		} else if ok {
			if path.Ext(updatedYamlFiles[i].OutputPath) != "" {
				return fmt.Errorf("%w, provide a directory for outputPath instead", ErrFoundDirectoryWithPathOutput)
			}
			yamlFilesInDir, err = getFileNames(updatedYamlFiles[i].Path)
			if err != nil {
				return err
			}

			for _, foundYamlFilePath := range yamlFilesInDir {
				addSpecificYamlPath := &YamlFile{
					Name:       updatedYamlFiles[i].Name,
					Overlays:   updatedYamlFiles[i].Overlays,
					Documents:  updatedYamlFiles[i].Documents,
					Path:       foundYamlFilePath,
					OutputPath: updatedYamlFiles[i].OutputPath,
				}

				updatedYamlFiles = append(updatedYamlFiles, addSpecificYamlPath)
			}

			updatedYamlFiles = append(updatedYamlFiles[:i], updatedYamlFiles[i+1:]...)
			i--
		}
	}

	*yamlFiles = updatedYamlFiles

	return nil
}

func (yamlFiles *YamlFiles) mergeDuplicates() {
	updatedYamlFiles := []*YamlFile(*yamlFiles)

	search := make(map[string]*YamlFile, len(*yamlFiles))

	for i := 0; i <= len(updatedYamlFiles)-1; i++ {
		if search[updatedYamlFiles[i].Path] == nil {
			search[updatedYamlFiles[i].Path] = updatedYamlFiles[i]
		} else {
			search[updatedYamlFiles[i].Path].Overlays = append(search[updatedYamlFiles[i].Path].Overlays, updatedYamlFiles[i].Overlays...)
			search[updatedYamlFiles[i].Path].Documents = append(search[updatedYamlFiles[i].Path].Documents, updatedYamlFiles[i].Documents...)
			if updatedYamlFiles[i].OutputPath != "" {
				search[updatedYamlFiles[i].Path].OutputPath = updatedYamlFiles[i].OutputPath
			}

			updatedYamlFiles = append(updatedYamlFiles[:i], updatedYamlFiles[i+1:]...)
			i--
		}
	}

	*yamlFiles = updatedYamlFiles
}

// removeCommentsFromNode will traverse the tree of yaml nodes and remove all comments.
func removeCommentsFromNode(yamlNode *yaml.Node) {
	if yamlNode.Content == nil {
		return
	}

	for _, child := range yamlNode.Content {
		child.HeadComment = ""
		child.LineComment = ""
		child.FootComment = ""

		removeCommentsFromNode(child)
	}
}
