// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	// stdInPath is the special path that means "read from stdin instead of a file".
	stdInPath = "-"

	// stdInOutputPath is the output file name used for YAML that was read from stdin.
	stdInOutputPath = "stdin.yaml"

	// instructionsDirKey is the viper key holding the directory of the instructions file
	// currently being decoded. Relative yamlFiles paths are resolved against it.
	instructionsDirKey = "instructionsDir"
)

var (
	// ErrNoYamlFiles occurs when no instructions file lists any yamlFiles, --path was not
	// given and nothing was piped to stdin, which leaves nothing to apply overlays to.
	ErrNoYamlFiles = errors.New(
		"no YAML files to process: add yamlFiles to an instructions file, pass -p/--path, or pipe YAML on stdin",
	)

	// ErrStdInUsedMoreThanOnce occurs when "-" (stdin) is given more than once across the
	// --instructions and --path flags. Stdin can only be read once.
	ErrStdInUsedMoreThanOnce = errors.New(`stdin ("-") can only be used once across --instructions and --path`)
)

// Config contains the options, gathered from flags, environment variables and the config
// file, that control how instructions are loaded and how output is written.
type Config struct {
	// LogLevel is the minimum level of log messages to display.
	LogLevel logging.Level
	// InstructionsFiles are the instructions files to apply, in the order given on the
	// command line. When a later file conflicts with an earlier one, the later file wins.
	InstructionsFiles []string
	// OutputDir is the directory that overlaid YAML files are written to.
	OutputDir string
	// StdOut writes overlaid YAML to stdout instead of to OutputDir.
	StdOut bool
	// StdInPiped reports whether stdin is a pipe or a redirected file rather than a
	// terminal. When it is true and nothing else provides YAML files, YAML is read
	// from stdin.
	StdInPiped bool
	// RemoveComments strips all comments from the source YAML before overlays are applied.
	RemoveComments bool
	// Indent is the number of spaces used to indent YAML output.
	Indent int
	// Styles are the YAML output styles applied to the final documents.
	Styles actions.Styles
	// ValueFiles are values files used to render templated instructions files. Each file
	// is merged over the previous one.
	ValueFiles []string
	// Overlay is the ad hoc overlay built from the --query and --action flags.
	Overlay overlays.Overlay
	// Value is the raw --value flag for the ad hoc overlay, parsed as YAML.
	Value string
	// Path is the --path flag: a YAML file, a directory of YAML files, or "-" for stdin.
	Path string
	// DefaultOnMissingAction is the onMissing action used by overlays that do not set one.
	DefaultOnMissingAction actions.OnMissingAction
}

// doPostProcessing renders a YAML file and writes it to stdout or to its output file.
// writtenPaths tracks which output files have already been written in this run so that
// later YAML files targeting the same output path are appended rather than overwritten.
func (cfg *Config) doPostProcessing(yamlFile *YamlFile, writtenPaths map[string]bool) error {
	output, err := cfg.encodeNodes(yamlFile.Nodes)
	if err != nil {
		return fmt.Errorf("unable to marshal final document %s, error: %w", yamlFile.Path, err)
	}

	// Every document in the file was deleted by overlays, so there is nothing to write.
	if output.Len() == 0 {
		log.Debugf("File %s was omitted from output", yamlFile.Path)

		return nil
	}

	destination := os.Stdout

	if !viper.GetBool("stdout") {
		log.Debugf("Final: >>>\n%s\n", output)

		destination, err = cfg.openOutputFile(yamlFile, writtenPaths)
		if err != nil {
			return err
		}

		defer CloseFile(destination)
	}

	if _, err = output.WriteTo(destination); err != nil {
		return fmt.Errorf("failed to %w", err)
	}

	output.Reset()

	return nil
}

// openOutputFile opens or creates the output file for a YAML file.
// The first write to a given path truncates it (fresh start). Later writes in the same run
// append, so that multiple YAML files sharing one outputPath are all preserved.
func (cfg *Config) openOutputFile(yamlFile *YamlFile, writtenPaths map[string]bool) (*os.File, error) {
	defaultDirMode := 0755
	defaultFileMode := 0644

	// A relative outputPath is placed inside the output directory.
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

// encodeNodes encodes the given YAML documents into a single buffer for output.
// Documents emptied by delete overlays are skipped.
func (cfg *Config) encodeNodes(nodes []*yaml.Node) (*bytes.Buffer, error) {
	var anyDocumentEncoded bool

	output := new(bytes.Buffer)

	encoder := yaml.NewEncoder(output)

	defer func() {
		if anyDocumentEncoded {
			if err := encoder.Close(); err != nil {
				log.Criticalf("error closing encoder, %s", err)
			}
		}
	}()

	encoder.SetIndent(viper.GetInt("indentLevel"))

	// The encoder only writes "---" between documents, so the separator before the first
	// written document is added by hand. It is tracked separately from the loop index
	// because leading documents may have been emptied and skipped.
	var firstDocumentWritten bool

	for _, node := range nodes {
		if len(node.Content) == 0 {
			continue
		}

		if !firstDocumentWritten {
			output.WriteString("---\n")

			firstDocumentWritten = true
		}

		style := actions.GetStyleFromConfig(viper.GetStringSlice("outputStyle")...)
		if len(style) > 0 {
			actions.SetStyle(style, node)
		}

		if err := encoder.Encode(node); err != nil {
			return nil, fmt.Errorf("%w", err)
		}

		anyDocumentEncoded = true
	}

	return output, nil
}

// adHocYamlFiles returns the YAML files to process that did not come from an instructions
// file: the --path flag if it was given, otherwise piped stdin when nothing else provides
// YAML files. Falling back to stdin lets `cat app.yaml | yot -i overlays.yaml` work
// without "-p -".
func (cfg *Config) adHocYamlFiles(instructionSets []*Instructions) (YamlFiles, error) {
	if cfg.Path != "" {
		return yamlFilesFromPath(cfg.Path)
	}

	readStdIn := cfg.StdInPiped &&
		!cfg.instructionsReadFromStdIn() &&
		countYamlFiles(instructionSets) == 0

	if readStdIn {
		log.Debugf("No YAML files were given, reading YAML from stdin")

		return yamlFilesFromPath(stdInPath)
	}

	return nil, nil
}

// instructionsReadFromStdIn reports whether one of the instructions files is stdin, in
// which case stdin has already been consumed.
func (cfg *Config) instructionsReadFromStdIn() bool {
	for _, instructionsFile := range cfg.InstructionsFiles {
		if instructionsFile == stdInPath {
			return true
		}
	}

	return false
}

// yamlFilesFromPath builds YAML file entries for a path given on the command line.
// A relative path is resolved against the current working directory, and a directory is
// expanded into the YAML files beneath it. The files are not read here.
func yamlFilesFromPath(inputPath string) (YamlFiles, error) {
	resolvedPath := inputPath

	// Resolve against the working directory now. Otherwise expandDirectories would join a
	// relative path with the directory of whichever instructions file was read last.
	if inputPath != stdInPath {
		absolutePath, err := filepath.Abs(inputPath)
		if err != nil {
			return nil, fmt.Errorf("cannot get absolute path of %s, %w", inputPath, err)
		}

		resolvedPath = absolutePath
	}

	yamlFiles := YamlFiles{
		&YamlFile{
			Name:       "--path flag",
			Path:       resolvedPath,
			OutputPath: determineOutputPath(inputPath),
		},
	}

	if err := yamlFiles.expandDirectories(); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return yamlFiles, nil
}

// adHocOverlay builds the overlay given with the --query, --action and --value flags.
// It returns nil when --query was not given.
func (cfg *Config) adHocOverlay() (*overlays.Overlay, error) {
	if cfg.Overlay.Query == nil {
		return nil, nil
	}

	if err := yaml.Unmarshal([]byte(cfg.Value), &cfg.Overlay.Value); err != nil {
		return nil, fmt.Errorf("unable to read overlay value from flag, %w", err)
	}

	// yaml.Unmarshal wraps the value in a document node; overlays need the value itself.
	if cfg.Overlay.Value.Kind == yaml.DocumentNode && cfg.Overlay.Value.Content != nil {
		cfg.Overlay.Value = *cfg.Overlay.Value.Content[0]
	}

	return &cfg.Overlay, nil
}

// determineOutputPath returns the output path for a path given on the command line.
// YAML read from stdin has no file name, so it is written to stdin.yaml. Other paths
// return "" so that setOutputPath derives the output path from the input path.
func determineOutputPath(inputPath string) string {
	if inputPath == stdInPath {
		return stdInOutputPath
	}

	return ""
}
