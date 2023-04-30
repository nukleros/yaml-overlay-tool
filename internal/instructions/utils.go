// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// ReadStream reads a yaml file from stdin.
func ReadStream(fileName string) (io.Reader, error) {
	if fileName == "-" {
		return bufio.NewReader(os.Stdin), nil
	}

	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to %w", err)
	}

	return file, nil
}

// CloseFile safely closes a file handle.
func CloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Errorf("error closing file!: %s", err)
	}
}

// GetCommonPrefix determines the longest common prefix in a set of paths.
func GetCommonPrefix(separator byte, paths ...string) string {
	switch len(paths) {
	case 0:
		return ""
	case 1:
		return path.Dir(path.Clean(paths[0])) + string(os.PathSeparator)
	}

	cleanPath := path.Clean(paths[0]) + string(separator)

	for _, v := range paths[1:] {
		v = path.Clean(v) + string(separator)

		if len(v) < len(cleanPath) {
			cleanPath = cleanPath[:len(v)]
		}

		for i := 0; i < len(cleanPath); i++ {
			if v[i] != cleanPath[i] {
				cleanPath = cleanPath[:i]

				break
			}
		}
	}

	for i := len(cleanPath) - 1; i >= 0; i-- {
		if cleanPath[i] == separator {
			cleanPath = cleanPath[:i]

			break
		}
	}

	return cleanPath + string(os.PathSeparator)
}

// isDirectory determines if the given path is a directory.
func isDirectory(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("could not get file info for %s, %w", filePath, err)
	}

	return fileInfo.IsDir(), err
}

// getFileNames recursively walks a path and outputs all '.yaml' and '.yml' filenames found.
func getFileNames(filePath string) ([]string, error) {
	var results []string

	err := filepath.Walk(filePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
				results = append(results, path)
			}

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("could not walk directory %s, %w", filePath, err)
	}

	return results, nil
}
