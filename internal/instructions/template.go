// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func renderInstructionsTemplate(fileName string, values interface{}) (io.Reader, error) {
	instrTemplate, err := template.ParseFiles(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to process template for instructions file %s, %w", fileName, err)
	}

	instrTemplate = instrTemplate.Funcs(template.FuncMap(sprig.FuncMap())).Option("missingkey=zero")

	var byteBuffer bytes.Buffer

	if err := instrTemplate.Execute(&byteBuffer, values); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &byteBuffer, nil
}
