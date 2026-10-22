// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions_test

import (
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/instructions"
	"github.com/spf13/viper"
)

// TestExecute does not call t.Parallel(): Execute() reads and writes the global
// viper config (e.g. ReadInstructionFile sets "instructionsDir", encodeNodes reads
// "indentLevel"/"outputStyle"/"stdout"), which races with any other test in this
// package that also touches viper while running concurrently. See TestDocumentSeparators.
func TestExecute(t *testing.T) { //nolint:paralleltest // must not run concurrently; see comment above
	const testOutputDir = "output/"

	type args struct {
		cfg *instructions.Config
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test simple execution",
			args: args{
				cfg: &instructions.Config{
					InstructionsFile: "testdata/instructions.yaml",
					OutputDir:        testOutputDir,
					Indent:           2,
				},
			},
			wantErr: false,
		},
		{
			name: "test directory execution",
			args: args{
				cfg: &instructions.Config{
					InstructionsFile: "testdata/testInstructions.yaml",
					OutputDir:        testOutputDir,
					Indent:           2,
				},
			},
			wantErr: false,
		},
	}

	// openOutputFile resolves relative output paths via this global viper key
	// (not cfg.OutputDir); set it once since every subtest needs the same value.
	viper.Set("outputDirectory", "./output")

	// Subtests run sequentially, not in parallel: Execute() itself is not safe for
	// concurrent invocation -- ReadInstructionFile writes the "instructionsDir" of
	// the file it's currently processing to the same global viper config, which two
	// concurrent Execute() calls would race on. Real CLI usage only ever calls
	// Execute() once per process, so this mirrors how it's actually used.
	for _, tt := range tests { //nolint:paralleltest // subtests share global viper state and must not run concurrently
		t.Run(tt.name, func(t *testing.T) {
			if err := instructions.Execute(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("Execute(%v) error = %v, wantErr %v", tt.args.cfg, err, tt.wantErr)
			}
		})
	}
}
