// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"os"

	"github.com/spf13/cobra"
)

// Names of the shells supported by the completion command.
const (
	ShellBash       = "bash"
	ShellZsh        = "zsh"
	ShellFish       = "fish"
	ShellPowerShell = "powershell"
)

func (r *Root) CompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:                   completionUse,
		Short:                 completionShort,
		Long:                  completionLong,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{ShellBash, ShellZsh, ShellFish, ShellPowerShell},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run:                   r.Completions,
	}
}

func (r *Root) Completions(cmd *cobra.Command, args []string) {
	if len(args) > 1 {
		panic("Too Many Args")
	}

	switch args[0] {
	case ShellBash:
		if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
			r.Log.Fatal(err)
		}
	case ShellZsh:
		if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
			r.Log.Fatal(err)
		}
	case ShellFish:
		if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
			r.Log.Fatal(err)
		}
	case ShellPowerShell:
		if err := cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout); err != nil {
			r.Log.Fatal(err)
		}
	}
}
