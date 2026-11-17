// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"github.com/nukleros/yaml-overlay-tool/internal/instructions"
	"github.com/op/go-logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ErrMissingRequired occurs when a required flag is not passed.
var ErrMissingRequired = fmt.Errorf("missing required arguments")

// Root is the yot root command along with the options its flags populate.
type Root struct {
	configFile string
	Log        *logging.Logger
	Options    *instructions.Config
	Command    *cobra.Command
}

// New builds the yot root command with its flags and subcommands.
func New() *Root {
	root := &Root{
		Log: logging.MustGetLogger("cmd"),
		Options: &instructions.Config{
			LogLevel:               logging.WARNING,
			Styles:                 actions.Styles{actions.NormalStyle},
			DefaultOnMissingAction: actions.Ignore,
		},
		configFile: os.Getenv("YOT_CONFIG_FILE"),
	}

	cobra.OnInitialize(root.initConfig)

	root.Command = root.NewCommand()
	root.AddFlags()
	root.AddCommands()

	return root
}

// initConfig binds environment variables and reads the yot config file, if one exists.
func (root *Root) initConfig() {
	for envVar, settingKey := range envMap() {
		if err := viper.BindEnv(envVar, settingKey); err != nil {
			root.Log.Fatal(err)
		}
	}

	viper.SetConfigType("yaml")
	viper.SetConfigName("yot.config")

	if root.configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(root.configFile)
	} else {
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.yot/") // call multiple times to add many search paths
		viper.AddConfigPath("/etc/yot/")   // path to look for the config file in
	}

	if err := viper.ReadInConfig(); err != nil {
		if ok := errors.As(err, &viper.ConfigFileNotFoundError{}); !ok {
			root.Log.Fatal(err)
		}

		return
	}

	viper.Set("configFile", viper.ConfigFileUsed())
}

// NewCommand returns the cobra command for yot, which runs when no subcommand is given.
func (root Root) NewCommand() *cobra.Command {
	rootCmd := &cobra.Command{
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
		PersistentPreRun:           root.SetupLogging,
		PersistentPreRunE:          nil,
		PreRun:                     nil,
		PreRunE:                    nil,
		RunE:                       root.Execute,
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

	return rootCmd
}

// AddFlags registers every flag of the root command.
func (root *Root) AddFlags() {
	root.initializeCommonFlags()
	root.initializeInstructionFlags()
	root.initializeOutputFlags()
	root.initializeFormatFlags()
	root.initializeTemplateFlags()
	root.initializeStdInFlags()
}

// AddCommands registers the subcommands of the root command.
func (root *Root) AddCommands() {
	root.Command.AddCommand(root.CompletionCommand())
	root.Command.AddCommand(root.EnvCommand())
}

// SetupLogging configures the log format and level before any command runs.
func (root *Root) SetupLogging(cmd *cobra.Command, args []string) {
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
		root.Log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}
}

// Execute runs yot: it loads the instructions, applies the overlays and writes the output.
func (root *Root) Execute(cmd *cobra.Command, args []string) error {
	root.Options.StdInPiped = isStdInPiped()

	if err := instructions.Execute(root.Options); err != nil {
		cmd.SilenceUsage = true

		return fmt.Errorf("%w", err)
	}

	return nil
}

// Run executes the root command and exits with a non-zero status on error.
func (root *Root) Run() {
	if err := root.Command.Execute(); err != nil {
		root.Log.Error(err)
		os.Exit(1)
	}
}

// isStdInPiped reports whether stdin is a pipe or a redirected file. A terminal or a
// character device such as /dev/null is not, so yot will not wait on it for input.
func isStdInPiped() bool {
	stdInInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return stdInInfo.Mode()&os.ModeCharDevice == 0
}
