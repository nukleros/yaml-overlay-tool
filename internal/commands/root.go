// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/op/go-logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"github.com/nukleros/yaml-overlay-tool/internal/instructions"
)

// ErrMissingRequired occurs when a required flag is not passed.
var ErrMissingRequired = fmt.Errorf("missing required arguments")

type Root struct {
	configFile string
	Log        *logging.Logger
	Options    *instructions.Config
	Command    *cobra.Command
}

func New() *Root {
	rootCmd := &Root{
		Log: logging.MustGetLogger("cmd"),
		Options: &instructions.Config{
			LogLevel:               logging.WARNING,
			Styles:                 actions.Styles{actions.NormalStyle},
			DefaultOnMissingAction: actions.Ignore,
		},
		configFile: os.Getenv("YOT_CONFIG_FILE"),
	}

	cobra.OnInitialize(rootCmd.initConfig)

	rootCmd.Command = rootCmd.NewCommand()
	rootCmd.AddFlags()
	rootCmd.AddCommands()

	return rootCmd
}

func (rootCmd *Root) initConfig() {
	for k, v := range envMap() {
		if err := viper.BindEnv(k, v); err != nil {
			rootCmd.Log.Fatal(err)
		}
	}

	viper.SetConfigType("yaml")
	viper.SetConfigName("yot.config")

	if rootCmd.configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(rootCmd.configFile)
	} else {
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.yot/") // call multiple times to add many search paths
		viper.AddConfigPath("/etc/yot/")   // path to look for the config file in
	}

	if err := viper.ReadInConfig(); err != nil {
		if ok := errors.As(err, &viper.ConfigFileNotFoundError{}); !ok {
			rootCmd.Log.Fatal(err)
		}

		return
	}

	viper.Set("configFile", viper.ConfigFileUsed())
}

func (rootCmd Root) NewCommand() *cobra.Command {
	// cmd represents the base command when called without any subcommands.
	cmd := &cobra.Command{
		Use:                        YotUse,
		Aliases:                    []string{},
		SuggestFor:                 []string{},
		Short:                      YotShort,
		Long:                       YotLong,
		Example:                    helpUsageExample,
		ValidArgs:                  []string{},
		Args:                       nil,
		ArgAliases:                 []string{},
		BashCompletionFunction:     "",
		Deprecated:                 "",
		Hidden:                     false,
		Annotations:                map[string]string{},
		Version:                    version,
		PersistentPreRun:           rootCmd.SetupLogging,
		PersistentPreRunE:          nil,
		PreRun:                     nil,
		PreRunE:                    nil,
		RunE:                       rootCmd.Execute,
		PostRun:                    nil,
		PostRunE:                   nil,
		PersistentPostRun:          nil,
		PersistentPostRunE:         nil,
		SilenceErrors:              false,
		SilenceUsage:               false,
		DisableFlagParsing:         false,
		DisableAutoGenTag:          true,
		DisableFlagsInUseLine:      false,
		DisableSuggestions:         false,
		SuggestionsMinimumDistance: 0,
		TraverseChildren:           false,
		FParseErrWhitelist:         cobra.FParseErrWhitelist{},
	}

	return cmd
}

func (rootCmd *Root) AddFlags() {
	rootCmd.initializeCommonFlags()
	rootCmd.initializeInstructionFlags()
	rootCmd.initializeOutputFlags()
	rootCmd.initializeFormatFlags()
	rootCmd.initializeTemplateFlags()
	rootCmd.initializeStdInFlags()
}

func (rootCmd *Root) AddCommands() {
	rootCmd.Command.AddCommand(rootCmd.CompletionCommand())
	rootCmd.Command.AddCommand(rootCmd.EnvCommand())
}

func (rootCmd *Root) SetupLogging(cmd *cobra.Command, args []string) {
	cmd.SetOut(cmd.OutOrStdout())

	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05} [%{level}]%{color:reset} %{message}`,
	)
	backend := logging.AddModuleLevel(
		logging.NewBackendFormatter(logging.NewLogBackend(os.Stderr, "", 0), format),
	)

	logLevel, err := logging.LogLevel((viper.GetString("logLevel")))
	if err != nil {
		panic(err)
	}

	backend.SetLevel(logLevel, "")

	logging.SetBackend(backend)

	if viper.ConfigFileUsed() != "" {
		rootCmd.Log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}
}

func (rootCmd *Root) Execute(cmd *cobra.Command, args []string) error {
	if err := instructions.Execute(rootCmd.Options); err != nil {
		cmd.SilenceUsage = true

		return fmt.Errorf("%w", err)
	}

	return nil
}

func (rootCmd *Root) Run() {
	if err := rootCmd.Command.Execute(); err != nil {
		rootCmd.Log.Error(err)
		os.Exit(1)
	}
}
