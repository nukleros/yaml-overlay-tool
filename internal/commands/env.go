// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func (rootCmd *Root) EnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:                   "env [settingName]",
		Short:                 envShort,
		Long:                  envLong,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				cfgKeyMap := envMap()
				cfgKeys := make([]string, len(cfgKeyMap))
				i := 0
				for cfgKey := range cfgKeyMap {
					cfgKeys[i] = cfgKey
					i++
				}

				return cfgKeys, cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		Run: rootCmd.env,
	}
}

func (rootCmd *Root) env(cmd *cobra.Command, args []string) {
	cfgKeyMap := envMap()
	if len(args) == 0 {
		for cfgKey, cfgValue := range cfgKeyMap {
			fmt.Fprintf(os.Stdout, "%s=\"%v\"\n", cfgKey, viper.Get(cfgValue))
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s=\"%v\"\n", args[0], viper.Get(cfgKeyMap[args[0]]))
	}
}

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
