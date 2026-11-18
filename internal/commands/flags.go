// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"github.com/nukleros/yaml-overlay-tool/internal/actions"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/thediveo/enumflag"
)

// logLevelDebug is the name used for the debug log level, both as a valid
// value for the "log-level" flag and as its no-arg default.
const logLevelDebug = "debug"

// initializeCommonFlags registers the --config and --log-level flags.
func (root *Root) initializeCommonFlags() {
	root.Command.Flags().StringVar(&root.configFile, "config", root.configFile, "config file (default is $HOME/.yot)")

	if err := viper.BindPFlag("configFile", root.Command.Flags().Lookup("config")); err != nil {
		root.Log.Fatal(err)
	}

	logMap := map[logging.Level][]string{
		logging.CRITICAL: {"critical", "crit", "c"},
		logging.ERROR:    {"error", "err", "e"},
		logging.WARNING:  {"warning", "warn", "w"},
		logging.NOTICE:   {"notice", "note", "n"},
		logging.INFO:     {"info", "i"},
		logging.DEBUG:    {logLevelDebug, "d", "verbose", "v"},
	}

	root.Command.Flags().VarP(
		enumflag.New(&root.Options.LogLevel, "logLevel", logMap, enumflag.EnumCaseInsensitive),
		"log-level",
		"v",
		helpLogLevel,
	)

	root.Command.Flags().Lookup("log-level").NoOptDefVal = logLevelDebug

	if err := viper.BindPFlag("logLevel", root.Command.Flags().Lookup("log-level")); err != nil {
		root.Log.Fatal(err)
	}
}

// initializeOutputFlags registers the --output-directory and --stdout flags.
func (root *Root) initializeOutputFlags() {
	root.Command.Flags().StringVarP(
		&root.Options.OutputDir,
		"output-directory",
		"o",
		"./output",
		helpOutputDirectory,
	)

	if err := root.Command.MarkFlagDirname("output-directory"); err != nil {
		root.Log.Fatal(err)
	}

	root.Command.Flags().BoolVarP(
		&root.Options.StdOut,
		"stdout",
		"s",
		false,
		helpRenderStdOut,
	)

	if err := viper.BindPFlag("outputDirectory", root.Command.Flags().Lookup("output-directory")); err != nil {
		root.Log.Fatal(err)
	}

	if err := viper.BindPFlag("stdout", root.Command.Flags().Lookup("stdout")); err != nil {
		root.Log.Fatal(err)
	}

	viper.SetDefault("outputDirectory", "./output")
}

// initializeFormatFlags registers the flags that control how output YAML is formatted.
func (root *Root) initializeFormatFlags() {
	defaultIndentLevel := 2

	root.Command.Flags().IntVarP(
		&root.Options.Indent,
		"indent-level",
		"I",
		defaultIndentLevel,
		helpIndentLevel,
	)

	root.Command.Flags().VarP(
		enumflag.NewSlice(&root.Options.Styles, "style", actions.Styles{}.FlagMap(), enumflag.EnumCaseInsensitive),
		"output-style",
		"S",
		helpOutputStyle,
	)

	root.Command.Flags().BoolVarP(
		&root.Options.RemoveComments,
		"remove-comments",
		"",
		false,
		helpRemoveComments,
	)

	if err := viper.BindPFlag("removeComments", root.Command.Flags().Lookup("remove-comments")); err != nil {
		root.Log.Fatal(err)
	}

	if err := viper.BindPFlag("indentLevel", root.Command.Flags().Lookup("indent-level")); err != nil {
		root.Log.Fatal(err)
	}

	if err := viper.BindPFlag("outputStyle", root.Command.Flags().Lookup("output-style")); err != nil {
		root.Log.Fatal(err)
	}
}

// initializeTemplateFlags registers the repeatable --values-file flag.
func (root *Root) initializeTemplateFlags() {
	root.Command.Flags().StringArrayVarP(
		&root.Options.ValueFiles,
		"values-file",
		"f",
		nil,
		helpValueFile,
	)
}

// initializeInstructionFlags registers the repeatable --instructions flag and the
// --default-on-missing-action flag.
func (root *Root) initializeInstructionFlags() {
	// A string array (not a string slice) so that commas in file names are not split.
	root.Command.Flags().StringArrayVarP(
		&root.Options.InstructionsFiles,
		"instructions",
		"i",
		nil,
		helpInstructionsFile,
	)

	if err := root.Command.MarkFlagFilename("instructions"); err != nil {
		root.Log.Error(err)
	}

	root.Command.Flags().Var(
		&root.Options.DefaultOnMissingAction,
		"default-on-missing-action",
		helpDefaultOnMissingAction,
	)

	if err := viper.BindPFlag("defaultOnMissingAction", root.Command.Flags().Lookup("default-on-missing-action")); err != nil {
		root.Log.Fatal(err)
	}
}

// initializeStdInFlags registers the flags for one-off overlays from the command line:
// --query, --action, --value and --path.
func (root *Root) initializeStdInFlags() {
	root.Command.Flags().VarP(
		&root.Options.Overlay.Query,
		"query",
		"q",
		helpQuery,
	)

	root.Command.Flags().VarP(
		&root.Options.Overlay.Action,
		"action",
		"a",
		helpAction,
	)

	root.Command.Flags().StringVarP(
		&root.Options.Value,
		"value",
		"x",
		"",
		helpValue,
	)
	root.Command.Flags().StringVarP(
		&root.Options.Path,
		"path",
		"p",
		"",
		helpPath,
	)
}
