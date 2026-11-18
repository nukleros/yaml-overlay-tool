// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"context"
	"fmt"

	"github.com/nukleros/yaml-overlay-tool/internal/overlays"
	"github.com/op/go-logging"
	"golang.org/x/sync/errgroup"
)

var log = logging.MustGetLogger("instructions") //nolint:gochecknoglobals

// Execute loads the instructions described by cfg, applies every overlay, and writes the
// results. Each YAML file is overlaid in its own goroutine. Results are written one at a
// time, in the order files finish, by PostProcessHandler.
func Execute(cfg *Config) error {
	group, ctx := errgroup.WithContext(context.Background())

	instructions, err := cfg.GetInstructions()
	if err != nil {
		return err
	}

	processedFiles := make(chan *YamlFile, len(instructions.YamlFiles))

	for _, yamlFile := range instructions.YamlFiles {
		group.Go(
			func() error {
				// Overlays for one file are applied in order by a single handler, so a later
				// overlay always sees the result of an earlier one.
				stream := overlays.NewWorkStream()

				stream.StartStream()

				go yamlFile.queueOverlays(stream)

				if err := stream.StartHandler(); err != nil {
					return fmt.Errorf("%w", err)
				}

				select {
				case processedFiles <- yamlFile:
					return nil
				case <-ctx.Done():
					if err := ctx.Err(); err != nil {
						return fmt.Errorf("%w", err)
					}

					return nil
				}
			},
		)
	}

	go func() {
		group.Wait() //nolint:errcheck // we use this to gracefully close the channel and receive the err later
		close(processedFiles)
	}()

	if err := PostProcessHandler(cfg, processedFiles); err != nil {
		return err
	}

	if err := group.Wait(); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// PostProcessHandler receives overlaid YAML files and performs post overlay tasks:
// rendering, then saving to a file or writing to stdout.
func PostProcessHandler(cfg *Config, processedFiles <-chan *YamlFile) error {
	writtenPaths := make(map[string]bool)

	for yamlFile := range processedFiles {
		if err := cfg.doPostProcessing(yamlFile, writtenPaths); err != nil {
			return err
		}
	}

	return nil
}
