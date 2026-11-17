// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Instructions is the decoded contents of one instructions file.
type Instructions struct {
	// CommonOverlays apply to every YAML file processed in this run, whichever
	// instructions file, --path or stdin it came from.
	CommonOverlays []*overlays.Overlay `yaml:"commonOverlays,omitempty"`
	// YamlFiles lists YAML files to process and the overlays specific to each one. It is
	// optional: YAML files can also come from another instructions file, --path or stdin.
	YamlFiles YamlFiles `yaml:"yamlFiles,omitempty"`
}

// GetInstructions loads every instructions file and the ad hoc flags, merges them into a
// single list of YAML files with their overlays in the order they must be applied, and
// reads the YAML documents of each file.
//
// The returned Instructions has no CommonOverlays because they have already been copied
// into the overlay layers of every YAML file (see mergeInstructionSets).
func (cfg *Config) GetInstructions() (*Instructions, error) {
	if err := cfg.validateStdInUsage(); err != nil {
		return nil, err
	}

	instructionSets, err := cfg.loadInstructionFiles()
	if err != nil {
		return nil, err
	}

	adHocOverlay, err := cfg.adHocOverlay()
	if err != nil {
		return nil, err
	}

	if adHocOverlay != nil {
		instructionSets = appendAdHocOverlay(instructionSets, adHocOverlay)
	}

	adHocFiles, err := cfg.adHocYamlFiles(instructionSets)
	if err != nil {
		return nil, err
	}

	if countYamlFiles(instructionSets)+len(adHocFiles) == 0 {
		return nil, ErrNoYamlFiles
	}

	merged := &Instructions{
		YamlFiles: mergeInstructionSets(instructionSets, adHocFiles),
	}

	// Files are read only after merging, so a file named by several instructions files
	// (or stdin) is read exactly once.
	if err := merged.YamlFiles.readYamlFiles(); err != nil {
		return nil, err
	}

	merged.setOutputPath()

	if viper.GetBool("removeComments") {
		merged.YamlFiles.removeComments()
	}

	return merged, nil
}

// validateStdInUsage makes sure stdin ("-") is requested at most once across the
// --instructions and --path flags, because stdin is empty once it has been read.
func (cfg *Config) validateStdInUsage() error {
	stdInUses := 0

	for _, instructionsFile := range cfg.InstructionsFiles {
		if instructionsFile == stdInPath {
			stdInUses++
		}
	}

	if cfg.Path == stdInPath {
		stdInUses++
	}

	if stdInUses > 1 {
		return ErrStdInUsedMoreThanOnce
	}

	return nil
}

// loadInstructionFiles reads every instructions file in the order given on the command
// line. Values files, if any, are read once and used to render each templated file.
func (cfg *Config) loadInstructionFiles() ([]*Instructions, error) {
	var values interface{}

	if cfg.ValueFiles != nil {
		var err error

		values, err = getValues(cfg.ValueFiles)
		if err != nil {
			return nil, err
		}
	}

	instructionSets := make([]*Instructions, 0, len(cfg.InstructionsFiles))

	for _, instructionsFile := range cfg.InstructionsFiles {
		instructionSet, err := cfg.readInstructionFile(instructionsFile, values)
		if err != nil {
			return nil, err
		}

		instructionSets = append(instructionSets, instructionSet)
	}

	return instructionSets, nil
}

// readInstructionFile reads and decodes one instructions file. When values files were
// given, the file is rendered as a Go template with those values first.
func (cfg *Config) readInstructionFile(instructionsFile string, values interface{}) (*Instructions, error) {
	log.Debugf("Instructions File: %s\n", instructionsFile)

	instructionsPath, err := filepath.Abs(instructionsFile)
	if err != nil {
		return nil, fmt.Errorf("cannot get absolute path of instructions file %s, %w", instructionsFile, err)
	}

	// Relative yamlFiles paths are resolved against the directory of the instructions file
	// they appear in. Decoding reads that directory from viper (see expandDirectories), so
	// it must be set right before each file is decoded, and files are decoded one at a time.
	viper.Set(instructionsDirKey, path.Dir(instructionsPath))

	var reader io.Reader

	if cfg.ValueFiles != nil {
		reader, err = renderInstructionsTemplate(instructionsFile, values)
	} else {
		reader, err = ReadStream(instructionsFile)
	}

	if err != nil {
		return nil, err
	}

	if file, isFile := reader.(*os.File); isFile {
		defer CloseFile(file)
	}

	var instructions Instructions

	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&instructions); err != nil {
		return nil, fmt.Errorf("unable to read instructions file %s: %w", instructionsFile, err)
	}

	return &instructions, nil
}

// appendAdHocOverlay adds the --query overlay as the last common overlay of the last
// instructions file. That is where it has always run when used with a single
// instructions file. With no instructions files, it becomes the only common overlay.
func appendAdHocOverlay(instructionSets []*Instructions, adHocOverlay *overlays.Overlay) []*Instructions {
	if len(instructionSets) == 0 {
		instructionSets = append(instructionSets, &Instructions{})
	}

	lastSet := instructionSets[len(instructionSets)-1]
	lastSet.CommonOverlays = append(lastSet.CommonOverlays, adHocOverlay)

	return instructionSets
}

// countYamlFiles returns the total number of YAML files listed across instructions files.
func countYamlFiles(instructionSets []*Instructions) int {
	total := 0

	for _, instructionSet := range instructionSets {
		total += len(instructionSet.YamlFiles)
	}

	return total
}

// setOutputPath fills in the output path of every YAML file. With no outputPath, a file
// keeps its path relative to the longest directory shared by all input files. An
// outputPath without a file extension is a directory, and the input file name is kept
// inside it.
func (instructions *Instructions) setOutputPath() {
	inputPaths := make([]string, 0, len(instructions.YamlFiles))

	for _, yamlFile := range instructions.YamlFiles {
		if yamlFile.Path != stdInPath {
			inputPaths = append(inputPaths, yamlFile.Path)
		}
	}

	pathPrefix := GetCommonPrefix(os.PathSeparator, inputPaths...)

	for _, yamlFile := range instructions.YamlFiles {
		if yamlFile.OutputPath == "" {
			yamlFile.OutputPath = strings.TrimPrefix(yamlFile.Path, pathPrefix)
		} else if path.Ext(yamlFile.OutputPath) == "" {
			yamlFile.OutputPath = path.Join(yamlFile.OutputPath, path.Base(yamlFile.Path))
		}
	}
}
