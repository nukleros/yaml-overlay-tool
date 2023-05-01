// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package commands

import (
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"github.com/thediveo/enumflag"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
)

func (rootCmd *Root) initializeCommonFlags() {
	rootCmd.Command.Flags().StringVar(&rootCmd.configFile, "config", rootCmd.configFile, "config file (default is $HOME/.yot)")

	if err := viper.BindPFlag("configFile", rootCmd.Command.Flags().Lookup("config")); err != nil {
		rootCmd.Log.Fatal(err)
	}

	logMap := map[logging.Level][]string{
		logging.CRITICAL: {"critical", "crit", "c"},
		logging.ERROR:    {"error", "err", "e"},
		logging.WARNING:  {"warning", "warn", "w"},
		logging.NOTICE:   {"notice", "note", "n"},
		logging.INFO:     {"info", "i"},
		logging.DEBUG:    {"debug", "d", "verbose", "v"},
	}

	rootCmd.Command.Flags().VarP(
		enumflag.New(&rootCmd.Options.LogLevel, "logLevel", logMap, enumflag.EnumCaseInsensitive),
		"log-level",
		"v",
		helpLogLevel,
	)

	rootCmd.Command.Flags().Lookup("log-level").NoOptDefVal = "debug"

	if err := viper.BindPFlag("logLevel", rootCmd.Command.Flags().Lookup("log-level")); err != nil {
		rootCmd.Log.Fatal(err)
	}
}

func (rootCmd *Root) initializeOutputFlags() {
	rootCmd.Command.Flags().StringVarP(
		&rootCmd.Options.OutputDir,
		"output-directory",
		"o",
		"./output",
		helpOutputDirectory,
	)

	if err := rootCmd.Command.MarkFlagDirname("output-directory"); err != nil {
		rootCmd.Log.Fatal(err)
	}

	rootCmd.Command.Flags().BoolVarP(
		&rootCmd.Options.StdOut,
		"stdout",
		"s",
		false,
		helpRenderStdOut,
	)

	if err := viper.BindPFlag("outputDirectory", rootCmd.Command.Flags().Lookup("output-directory")); err != nil {
		rootCmd.Log.Fatal(err)
	}

	if err := viper.BindPFlag("stdout", rootCmd.Command.Flags().Lookup("stdout")); err != nil {
		rootCmd.Log.Fatal(err)
	}

	viper.SetDefault("outputDirectory", "./output")
}

func (rootCmd *Root) initializeFormatFlags() {
	defaultIndentLevel := 2

	rootCmd.Command.Flags().IntVarP(
		&rootCmd.Options.Indent,
		"indent-level",
		"I",
		defaultIndentLevel,
		helpIndentLevel,
	)

	rootCmd.Command.Flags().VarP(
		enumflag.NewSlice(&rootCmd.Options.Styles, "style", actions.Styles{}.FlagMap(), enumflag.EnumCaseInsensitive),
		"output-style",
		"S",
		helpOutputStyle,
	)

	rootCmd.Command.Flags().BoolVarP(
		&rootCmd.Options.RemoveComments,
		"remove-comments",
		"",
		false,
		helpRemoveComments,
	)

	if err := viper.BindPFlag("removeComments", rootCmd.Command.Flags().Lookup("remove-comments")); err != nil {
		rootCmd.Log.Fatal(err)
	}

	if err := viper.BindPFlag("indentLevel", rootCmd.Command.Flags().Lookup("indent-level")); err != nil {
		rootCmd.Log.Fatal(err)
	}

	if err := viper.BindPFlag("outputStyle", rootCmd.Command.Flags().Lookup("output-style")); err != nil {
		rootCmd.Log.Fatal(err)
	}
}

func (rootCmd *Root) initializeTemplateFlags() {
	rootCmd.Command.Flags().StringArrayVarP(
		&rootCmd.Options.ValueFiles,
		"values-file",
		"f",
		nil,
		helpValueFile,
	)
}

func (rootCmd *Root) initializeInstructionFlags() {
	rootCmd.Command.Flags().StringVarP(
		&rootCmd.Options.InstructionsFile,
		"instructions",
		"i",
		"",
		helpInstructionsFile,
	)

	if err := rootCmd.Command.MarkFlagFilename("instructions"); err != nil {
		rootCmd.Log.Error(err)
	}

	rootCmd.Command.Flags().Var(
		&rootCmd.Options.DefaultOnMissingAction,
		"default-on-missing-action",
		helpDefaultOnMissingAction,
	)

	if err := viper.BindPFlag("defaultOnMissingAction", rootCmd.Command.Flags().Lookup("default-on-missing-action")); err != nil {
		rootCmd.Log.Fatal(err)
	}
}

func (rootCmd *Root) initializeStdInFlags() {
	rootCmd.Command.Flags().VarP(
		&rootCmd.Options.Overlay.Query,
		"query",
		"q",
		helpQuery,
	)

	rootCmd.Command.Flags().VarP(
		&rootCmd.Options.Overlay.Action,
		"action",
		"a",
		helpAction,
	)

	rootCmd.Command.Flags().StringVarP(
		&rootCmd.Options.Value,
		"value",
		"x",
		"",
		helpValue,
	)
	rootCmd.Command.Flags().StringVarP(
		&rootCmd.Options.Path,
		"path",
		"p",
		"",
		helpPath,
	)
}
