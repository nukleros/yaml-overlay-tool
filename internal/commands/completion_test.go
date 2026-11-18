// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands_test

import (
	"fmt"
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/commands"
	"github.com/spf13/cobra"
)

func TestCompletions(t *testing.T) {
	t.Parallel()

	// commands.New registers process-wide state (cobra.OnInitialize and the global viper
	// instance), so it must not be called from the parallel subtests below. Create the
	// root once and share it: Completions only reads root.Log. Each case still gets its
	// own command from CompletionCommand, so the completion generators never share one.
	root := commands.New()

	type args struct {
		cmd  *cobra.Command
		args []string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Without any args",
			args: args{
				cmd:  root.CompletionCommand(),
				args: nil,
			},
			wantErr: true,
		},
		{
			name: "With bash arg",
			args: args{
				cmd:  root.CompletionCommand(),
				args: []string{commands.ShellBash},
			},
			wantErr: false,
		},
		{
			name: "With Zsh arg",
			args: args{
				cmd:  root.CompletionCommand(),
				args: []string{commands.ShellZsh},
			},
			wantErr: false,
		},
		{
			name: "With fish arg",
			args: args{
				cmd:  root.CompletionCommand(),
				args: []string{commands.ShellFish},
			},
			wantErr: false,
		},
		{
			name: "With powershell arg",
			args: args{
				cmd:  root.CompletionCommand(),
				args: []string{commands.ShellPowerShell},
			},
			wantErr: false,
		},
		{
			name: "Without too many args",
			args: args{
				cmd:  root.CompletionCommand(),
				args: []string{commands.ShellBash, commands.ShellZsh},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// Completions panics on bad arguments, so a recovered panic counts as an error.
			defer func() {
				if recovered := recover(); recovered != nil {
					recoveredErr := fmt.Errorf("%v", recovered) //nolint

					if !test.wantErr && recoveredErr != nil {
						t.Errorf("Completions() error = %v, wantErr %v", recovered, test.wantErr)
					}
				}
			}()

			root.Completions(test.args.cmd, test.args.args)

			if test.wantErr {
				t.Errorf("Completions() error = %v, wantErr %v", nil, test.wantErr)
			}
		})
	}
}
