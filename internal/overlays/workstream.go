// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package overlays

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type workload struct {
	Overlay   *Overlay
	Node      *yaml.Node
	NodeIndex int
	Path      string
}

type WorkStream struct {
	stream chan *workload
}

func NewWorkStream() *WorkStream {
	return &WorkStream{}
}

func (workStream *WorkStream) StartStream() {
	workStream.stream = make(chan *workload)
}

func (workStream *WorkStream) AddWorkload(overlay *Overlay, yamlNode *yaml.Node, nodeIndex int, path string) {
	workStream.stream <- &workload{
		Overlay:   overlay,
		Node:      yamlNode,
		NodeIndex: nodeIndex,
		Path:      path,
	}
}

func (workStream *WorkStream) CloseStream() {
	close(workStream.stream)
}

func (workStream *WorkStream) StartHandler() error {
	for overlay := range workStream.stream {
		log.Noticef("Processing overlay [%q] in file %s on document %d", overlay.Overlay.Name, overlay.Path, overlay.NodeIndex)

		if err := overlay.Overlay.Apply(overlay.Node); err != nil {
			return fmt.Errorf("%w in file %s on Document %d", err, overlay.Path, overlay.NodeIndex)
		}
	}

	return nil
}
