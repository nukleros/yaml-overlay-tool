// Copyright 2021 VMware, Inc.
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
	"github.com/vmware-tanzu-labs/yaml-overlay-tool/internal/actions"
	"github.com/vmware-tanzu-labs/yaml-overlay-tool/internal/overlays"
	"gopkg.in/yaml.v3"
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
// writtenPaths tracks which output files have already been written in this run so that
// subsequent YamlFiles targeting the same path are appended rather than overwritten.
func (cfg *Config) doPostProcessing(yf *YamlFile, writtenPaths map[string]bool) error {
	var o *os.File

	var err error

	output, err := cfg.encodeNodes(yf.Nodes)
	if err != nil {
		return fmt.Errorf("unable to marshal final document %s, error: %w", yf.Path, err)
	}

	if output.Len() == 0 {
		log.Debugf("File %s was omitted from output", yf.Path)

		return nil
	}

	if viper.GetBool("stdout") {
		o = os.Stdout
	} else {
		log.Debugf("Final: >>>\n%s\n", output)
		o, err = cfg.openOutputFile(yf, writtenPaths)
		if err != nil {
			return err
		}

		defer CloseFile(o)
	}

	if _, err = output.WriteTo(o); err != nil {
		return fmt.Errorf("failed to %w", err)
	}

	output.Reset()

	return nil
}

// openOutputFile opens or creates a file for outputing results.
// The first write to a given path truncates (fresh start); subsequent writes in the same run
// append so that multiple YamlFiles sharing one outputPath are all preserved.
func (cfg *Config) openOutputFile(yf *YamlFile, writtenPaths map[string]bool) (*os.File, error) {
	defaultDirMode := 0755
	defaultFileMode := 0644

	fileName := yf.OutputPath
	if !path.IsAbs(yf.OutputPath) {
		fileName = path.Join(viper.GetString("outputDirectory"), yf.OutputPath)
	}

	dirName := path.Dir(fileName)

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if err := os.MkdirAll(dirName, fs.FileMode(defaultDirMode)); err != nil {
			return nil, fmt.Errorf("failed to create output directory %s, %w", dirName, err)
		}
	}

	flags := os.O_CREATE | os.O_RDWR | os.O_TRUNC
	if writtenPaths[fileName] {
		flags = os.O_CREATE | os.O_RDWR | os.O_APPEND
	}

	writtenPaths[fileName] = true

	file, err := os.OpenFile(fileName, flags, fs.FileMode(defaultFileMode))
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

	ye := yaml.NewEncoder(output)

	defer func() {
		if fileWritten {
			if err := ye.Close(); err != nil {
				log.Criticalf("error closing encoder, %s", err)
			}
		}
	}()

	ye.SetIndent(viper.GetInt("indentLevel"))

	var firstDocWritten bool

	for _, node := range nodes {
		if len(node.Content) == 0 {
			continue
		}

		if !firstDocWritten {
			output.WriteString("---\n")
			firstDocWritten = true
		}

		style := actions.GetStyleFromConfig(viper.GetStringSlice("outputStyle")...)
		if len(style) > 0 {
			actions.SetStyle(style, node)
		}

		err := ye.Encode(node)
		if err != nil {
			return nil, fmt.Errorf("%w", err)
		}

		fileWritten = true
	}

	return output, nil
}

func (cfg *Config) ReadAdHocPaths(i *Instructions) error {
	if cfg.Path != "" {
		yfs := YamlFiles{
			&YamlFile{
				Name:       "StdIn",
				Path:       cfg.Path,
				OutputPath: determinePath(cfg.Path),
			},
		}

		if err := yfs.expandDirectories(); err != nil {
			return fmt.Errorf("%w", err)
		}

		for _, yf := range yfs {
			if err := yf.readYamlFile(); err != nil {
				return err
			}
		}

		i.YamlFiles = append(i.YamlFiles, yfs...)

		i.YamlFiles.mergeDuplicates()
	}

	return nil
}

func (cfg *Config) ReadAdHocOverlays(i *Instructions) error {
	if cfg.Overlay.Query != nil {
		if err := yaml.Unmarshal([]byte(cfg.Value), &cfg.Overlay.Value); err != nil {
			return fmt.Errorf("unable to read overlay value from flag, %w", err)
		}

		if cfg.Overlay.Value.Kind == yaml.DocumentNode && cfg.Overlay.Value.Content != nil {
			cfg.Overlay.Value = *cfg.Overlay.Value.Content[0]
		}

		i.CommonOverlays = append(i.CommonOverlays, &cfg.Overlay)
	}

	return nil
}

func determinePath(p string) string {
	if p == "-" {
		return "stdin.yaml"
	}

	return ""
}
