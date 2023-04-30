// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
)

// Config contains configuration options used with instruction files.
type Config struct {
	LogLevel               logging.Level
	InstructionsFile       string
	OutputDir              string
	StdOut                 bool
	RemoveComments         bool
	Indent                 int
	Styles                 actions.Styles
	ValueFiles             []string
	Overlay                overlays.Overlay
	Value                  string
	Path                   string
	DefaultOnMissingAction actions.OnMissingAction
}

// doPostProcessing renders a document and outputs it to the location specified in config.
func (cfg *Config) doPostProcessing(yamlFile *YamlFile) error {
	var outputPath *os.File

	var err error

	output, err := cfg.encodeNodes(yamlFile.Nodes)
	if err != nil {
		return fmt.Errorf("unable to marshal final document %s, error: %w", yamlFile.Path, err)
	}

	if output.Len() == 0 {
		log.Debugf("File %s was omitted from output", yamlFile.Path)

		return nil
	}

	if viper.GetBool("stdout") {
		outputPath = os.Stdout
	} else {
		log.Debugf("Final: >>>\n%s\n", output)
		outputPath, err = cfg.openOutputFile(yamlFile)
		if err != nil {
			return err
		}

		defer CloseFile(outputPath)
	}

	if _, err = output.WriteTo(outputPath); err != nil {
		return fmt.Errorf("failed to %w", err)
	}

	output.Reset()

	return nil
}

// openOutputFile opens or creates a file for outputing results.
func (cfg *Config) openOutputFile(yamlFile *YamlFile) (*os.File, error) {
	defaultDirMode := 0755
	defaultFileMode := 0644

	fileName := yamlFile.OutputPath
	if !path.IsAbs(yamlFile.OutputPath) {
		fileName = path.Join(viper.GetString("outputDirectory"), yamlFile.OutputPath)
	}

	dirName := path.Dir(fileName)

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if err := os.MkdirAll(dirName, fs.FileMode(defaultDirMode)); err != nil {
			return nil, fmt.Errorf("failed to create output directory %s, %w", dirName, err)
		}
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fs.FileMode(defaultFileMode))
	if err != nil {
		return nil, fmt.Errorf("failed to create/open file %s: %w", fileName, err)
	}

	os.Stdout.WriteString(fileName + "\n")

	return file, nil
}

// encodeNodes processes each node given and encodes them into a single buffer for output.
func (cfg *Config) encodeNodes(nodes []*yaml.Node) (*bytes.Buffer, error) {
	var fileWritten bool

	output := new(bytes.Buffer)

	yamlEncoder := yaml.NewEncoder(output)

	defer func() {
		if fileWritten {
			if err := yamlEncoder.Close(); err != nil {
				log.Criticalf("error closing encoder, %s", err)
			}
		}
	}()

	yamlEncoder.SetIndent(viper.GetInt("indentLevel"))

	for i, node := range nodes {
		if len(node.Content) == 0 {
			continue
		}

		if i == 0 {
			output.WriteString("---\n")
		}

		style := actions.GetStyleFromConfig(viper.GetStringSlice("outputStyle")...)
		actions.SetStyle(style, node)

		err := yamlEncoder.Encode(node)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}

		fileWritten = true
	}

	return output, nil
}

func (cfg *Config) ReadAdHocPaths(instructions *Instructions) error {
	if cfg.Path != "" {
		yamlFiles := YamlFiles{
			&YamlFile{
				Name:       "StdIn",
				Path:       cfg.Path,
				OutputPath: determinePath(cfg.Path),
			},
		}

		if err := yamlFiles.expandDirectories(); err != nil {
			return fmt.Errorf("%w", err)
		}

		for _, yamlFile := range yamlFiles {
			if err := yamlFile.readYamlFile(); err != nil {
				return err
			}
		}

		instructions.YamlFiles = append(instructions.YamlFiles, yamlFiles...)

		instructions.YamlFiles.mergeDuplicates()
	}

	return nil
}

func (cfg *Config) ReadAdHocOverlays(instructions *Instructions) error {
	if cfg.Overlay.Query != nil {
		if err := yaml.Unmarshal([]byte(cfg.Value), &cfg.Overlay.Value); err != nil {
			return fmt.Errorf("unable to read overlay value from flag, %w", err)
		}

		if cfg.Overlay.Value.Kind == yaml.DocumentNode && cfg.Overlay.Value.Content != nil {
			cfg.Overlay.Value = *cfg.Overlay.Value.Content[0]
		}

		instructions.CommonOverlays = append(instructions.CommonOverlays, &cfg.Overlay)
	}

	return nil
}

func determinePath(filePath string) string {
	if filePath == "-" {
		return "stdin.yaml"
	}

	return ""
}
