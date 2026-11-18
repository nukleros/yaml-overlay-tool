// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// EnvCommand returns the subcommand that prints the settings yot is using.
func (root *Root) EnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:                   "env [settingName]",
		Short:                 envShort,
		Long:                  envLong,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				settings := envMap()
				envVars := make([]string, 0, len(settings))

				for envVar := range settings {
					envVars = append(envVars, envVar)
				}

				return envVars, cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		Run: root.env,
	}
}

// env prints every setting, or only the one named in args, as ENV_VAR="value".
func (root *Root) env(cmd *cobra.Command, args []string) {
	settings := envMap()
	if len(args) == 0 {
		for envVar, settingKey := range settings {
			fmt.Fprintf(os.Stdout, "%s=\"%v\"\n", envVar, viper.Get(settingKey))
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s=\"%v\"\n", args[0], viper.Get(settings[args[0]]))
	}
}

// envMap maps each environment variable yot reads to the setting it controls.
func envMap() map[string]string {
	return map[string]string{
		"YOT_CONFIG_FILE":               "configFile",
		"YOT_DEFAULT_ON_MISSING_ACTION": "defaultOnMissingAction",
		"YOT_INDENT_LEVEL":              "indentLevel",
		"YOT_LOG_LEVEL":                 "logLevel",
		"YOT_OUTPUT_DIRECTORY":          "outputDirectory",
		"YOT_OUTPUT_STYLE":              "outputStyle",
		"YOT_REMOVE_COMMENTS":           "removeComments",
		"YOT_STDOUT":                    "stdout",
	}
}
