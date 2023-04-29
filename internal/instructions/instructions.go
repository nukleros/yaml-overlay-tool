// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
)

// Instructions is a struct used for decoding an instructions file.
type Instructions struct {
	// Common Overlays that will apply to all files specified.
	CommonOverlays []*overlays.Overlay `yaml:"commonOverlays,omitempty"`
	// List of YamlFiles and overlays to apply.
	YamlFiles YamlFiles `yaml:"yamlFiles,omitempty"`
}

func (cfg *Config) GetInstructions() (*Instructions, error) {
	instructions := new(Instructions)

	var err error

	if cfg.InstructionsFile != "" {
		instructions, err = cfg.ReadInstructionFile()
		if err != nil {
			return nil, err
		}
	} else {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not determine working directory, %w", err)
		}

		viper.Set("instructionDir", workingDirectory)
	}

	if err := cfg.ReadAdHocPaths(instructions); err != nil {
		return nil, err
	}

	if err := cfg.ReadAdHocOverlays(instructions); err != nil {
		return nil, err
	}

	instructions.setOutputPath()

	instructions.addCommonOverlays()

	// remove the comments if requested
	if viper.GetBool("removeComments") {
		for _, yamlFile := range instructions.YamlFiles {
			for _, node := range yamlFile.Nodes {
				removeCommentsFromNode(node)
			}
		}
	}

	return instructions, nil
}

// ReadInstructionFile reads a file and decodes it into an Instructions struct.
func (cfg *Config) ReadInstructionFile() (*Instructions, error) {
	var reader io.Reader

	var values interface{}

	log.Debugf("Instructions File: %s\n", cfg.InstructionsFile)

	var instructions Instructions

	instructionsPath, err := filepath.Abs(cfg.InstructionsFile)
	if err != nil {
		return nil, fmt.Errorf("cannot get absolute path of instructions file %s, %w", cfg.InstructionsFile, err)
	}

	viper.Set("instructionsDir", path.Dir(instructionsPath))

	if cfg.ValueFiles != nil {
		values, err = getValues(cfg.ValueFiles)
		if err != nil {
			return nil, err
		}

		reader, err = renderInstructionsTemplate(cfg.InstructionsFile, values)
		if err != nil {
			return nil, err
		}
	} else {
		reader, err = ReadStream(cfg.InstructionsFile)
		if err != nil {
			return nil, err
		}
	}

	yamlDecoder := yaml.NewDecoder(reader)
	if err := yamlDecoder.Decode(&instructions); err != nil {
		return nil, fmt.Errorf("unable to read instructions file %s: %w", cfg.InstructionsFile, err)
	}

	return &instructions, nil
}

// addCommonOverlays, takes all common overlays and adds them to each yamlFile entry.
func (instructions *Instructions) addCommonOverlays() {
	for _, yamlFile := range instructions.YamlFiles {
		yamlFile.Overlays = append(instructions.CommonOverlays, yamlFile.Overlays...)
	}
}

func (instructions *Instructions) setOutputPath() {
	outputPath := make([]string, 0, len(instructions.YamlFiles))

	for _, yamlFile := range instructions.YamlFiles {
		if yamlFile.Path != "-" {
			outputPath = append(outputPath, yamlFile.Path)
		}
	}

	pathPrefix := GetCommonPrefix(os.PathSeparator, outputPath...)

	for _, yamlFile := range instructions.YamlFiles {
		if yamlFile.OutputPath == "" {
			yamlFile.OutputPath = strings.TrimPrefix(yamlFile.Path, pathPrefix)
		} else if path.Ext(yamlFile.OutputPath) == "" {
			yamlFile.OutputPath = path.Join(yamlFile.OutputPath, path.Base(yamlFile.Path))
		}
	}
}
