// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// renderInstructionsTemplate renders an instructions file as a Go template with the given
// values, and returns the rendered YAML. Missing values render as their zero value.
func renderInstructionsTemplate(fileName string, values interface{}) (io.Reader, error) {
	instructionsTemplate, err := template.ParseFiles(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to process template for instructions file %s, %w", fileName, err)
	}

	instructionsTemplate = instructionsTemplate.Funcs(sprig.FuncMap()).Option("missingkey=zero")

	var rendered bytes.Buffer

	if err := instructionsTemplate.Execute(&rendered, values); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &rendered, nil
}
