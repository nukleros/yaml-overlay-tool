// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package main

import (
	command "github.com/nukleros/yaml-overlay-tool/internal/commands"
)

func main() {
	yot := command.New()

	yot.Run()
}
