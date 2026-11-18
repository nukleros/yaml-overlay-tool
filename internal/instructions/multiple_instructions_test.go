// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/instructions"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Fixtures for combining instructions files. Every fixture overlays the "team" label of
// appManifest, so the label's final value shows which overlay won.
const (
	labelsFile       = "testdata/multi/labels.yaml"
	labelsOverride   = "testdata/multi/labelsOverride.yaml"
	withYamlFiles    = "testdata/multi/withYamlFiles.yaml"
	nestedYamlFiles  = "testdata/multi/nested/alsoWithYamlFiles.yaml"
	appManifest      = "testdata/multi/manifests/app.yaml"
	appOutputFile    = "app.yaml"
	stdInOutputFile  = "stdin.yaml"
	missingTeamLabel = "<missing>"

	// Values the fixtures set the team label to.
	teamFromLabels   = "labels"
	teamFromOverride = "override"
	teamFromSpecific = "specific"
	teamFromNested   = "nested"
)

// The tests in this file do not call t.Parallel(): Execute reads and writes global viper
// state. See TestExecute.

// TestMultipleInstructionsFiles checks that instructions files are applied in the order
// given, with a later file winning over an earlier one. --path is relative to the working
// directory (the package directory under go test), not to the instructions files'
// directory, so these cases also prove that --path is resolved against the working
// directory.
func TestMultipleInstructionsFiles(t *testing.T) { //nolint:paralleltest // shares global viper state
	tests := []struct {
		name              string
		instructionsFiles []string
		path              string
		wantTeam          string
	}{
		{
			name:              "a later file's commonOverlays win",
			instructionsFiles: []string{labelsFile, labelsOverride},
			path:              appManifest,
			wantTeam:          teamFromOverride,
		},
		{
			name:              "reversing the order flips the winner",
			instructionsFiles: []string{labelsOverride, labelsFile},
			path:              appManifest,
			wantTeam:          teamFromLabels,
		},
		{
			name:              "a later file's commonOverlay beats an earlier file's file-specific overlay",
			instructionsFiles: []string{withYamlFiles, labelsOverride},
			wantTeam:          teamFromOverride,
		},
		{
			name:              "a later file's file-specific overlay beats an earlier file's commonOverlay",
			instructionsFiles: []string{labelsOverride, withYamlFiles},
			wantTeam:          teamFromSpecific,
		},
		{
			name:              "a single overlay-only file applies to --path",
			instructionsFiles: []string{labelsFile},
			path:              appManifest,
			wantTeam:          teamFromLabels,
		},
	}

	for _, test := range tests { //nolint:paralleltest // shares global viper state
		t.Run(test.name, func(t *testing.T) {
			outputDir := executeIntoTempDir(t, &instructions.Config{
				InstructionsFiles: test.instructionsFiles,
				Path:              test.path,
			})

			got := readTeamLabels(t, filepath.Join(outputDir, appOutputFile))
			if len(got) != 1 || got[0] != test.wantTeam {
				t.Errorf("team labels = %v, want [%s]", got, test.wantTeam)
			}
		})
	}
}

// TestSameYamlFileInSeveralInstructionsFiles checks that a YAML file named by two
// instructions files, each with a path relative to its own directory, is read and
// written once, with both files' overlays applied in order.
func TestSameYamlFileInSeveralInstructionsFiles(t *testing.T) { //nolint:paralleltest // shares global viper state
	outputDir := executeIntoTempDir(t, &instructions.Config{
		InstructionsFiles: []string{withYamlFiles, nestedYamlFiles},
	})

	got := readTeamLabels(t, filepath.Join(outputDir, appOutputFile))
	if len(got) != 1 || got[0] != teamFromNested {
		t.Errorf("team labels = %v, want exactly one document labeled [nested]", got)
	}
}

// TestPipedStdIn checks that YAML piped to stdin is processed when no instructions file
// lists yamlFiles and --path is not given.
func TestPipedStdIn(t *testing.T) { //nolint:paralleltest // replaces os.Stdin and shares global viper state
	manifest, err := os.ReadFile(appManifest)
	if err != nil {
		t.Fatalf("read %s: %v", appManifest, err)
	}

	replaceStdIn(t, manifest)

	outputDir := executeIntoTempDir(t, &instructions.Config{
		InstructionsFiles: []string{labelsFile},
		StdInPiped:        true,
	})

	got := readTeamLabels(t, filepath.Join(outputDir, stdInOutputFile))
	if len(got) != 1 || got[0] != teamFromLabels {
		t.Errorf("team labels = %v, want [labels]", got)
	}
}

// TestInputErrors checks the errors returned when there is nothing to process, or when
// stdin is given for an instructions or values file.
func TestInputErrors(t *testing.T) { //nolint:paralleltest // shares global viper state
	tests := []struct {
		name    string
		cfg     *instructions.Config
		wantErr error
	}{
		{
			name: "no yamlFiles, no --path and nothing piped",
			cfg: &instructions.Config{
				InstructionsFiles: []string{labelsFile},
				StdInPiped:        false,
			},
			wantErr: instructions.ErrNoYamlFiles,
		},
		{
			name:    "no instructions files and no input at all",
			cfg:     &instructions.Config{},
			wantErr: instructions.ErrNoYamlFiles,
		},
		{
			name: "instructions file from stdin",
			cfg: &instructions.Config{
				InstructionsFiles: []string{"-"},
			},
			wantErr: instructions.ErrStdInOnlyForPath,
		},
		{
			// Before stdin was limited to --path, the values file read stdin first and the
			// stdin fallback then picked the empty stream as the YAML to process.
			name: "values file from stdin with an overlay-only instructions file",
			cfg: &instructions.Config{
				InstructionsFiles: []string{labelsFile},
				ValueFiles:        []string{"-"},
				StdInPiped:        true,
			},
			wantErr: instructions.ErrStdInOnlyForPath,
		},
	}

	for _, test := range tests { //nolint:paralleltest // shares global viper state
		t.Run(test.name, func(t *testing.T) {
			viper.Set("outputDirectory", t.TempDir())

			if err := instructions.Execute(test.cfg); !errors.Is(err, test.wantErr) {
				t.Errorf("Execute() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

// executeIntoTempDir runs Execute with output written to a new temporary directory, and
// returns that directory.
func executeIntoTempDir(t *testing.T, cfg *instructions.Config) string {
	t.Helper()

	outputDir := t.TempDir()

	// openOutputFile reads the output directory from viper, not from cfg.OutputDir.
	viper.Set("outputDirectory", outputDir)
	viper.Set("stdout", false)

	if err := instructions.Execute(cfg); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	return outputDir
}

// readTeamLabels returns the metadata.labels.team value of every document in a YAML file.
func readTeamLabels(t *testing.T, fileName string) []string {
	t.Helper()

	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	var teams []string

	decoder := yaml.NewDecoder(bytes.NewReader(content))

	for {
		var document struct {
			Metadata struct {
				Labels map[string]string `yaml:"labels"`
			} `yaml:"metadata"`
		}

		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("decode %s: %v", fileName, err)
		}

		team, found := document.Metadata.Labels["team"]
		if !found {
			team = missingTeamLabel
		}

		teams = append(teams, team)
	}

	return teams
}

// replaceStdIn swaps os.Stdin for a pipe containing content, and restores it when the
// test finishes.
func replaceStdIn(t *testing.T, content []byte) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	originalStdIn := os.Stdin
	os.Stdin = reader

	t.Cleanup(func() {
		os.Stdin = originalStdIn

		reader.Close()
	})

	// The content is small enough to fit in the pipe buffer, so this does not block.
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}

	writer.Close()
}
