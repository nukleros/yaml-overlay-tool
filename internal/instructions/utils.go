// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReadStream opens fileName for reading, or returns a reader for stdin when fileName is
// "-". The caller is responsible for closing the returned reader when it is an *os.File.
func ReadStream(fileName string) (io.Reader, error) {
	if fileName == stdInPath {
		return bufio.NewReader(os.Stdin), nil
	}

	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to %w", err)
	}

	return file, nil
}

// CloseFile closes a file handle and logs any error instead of returning it.
func CloseFile(file *os.File) {
	if err := file.Close(); err != nil {
		log.Errorf("error closing file!: %s", err)
	}
}

// GetCommonPrefix returns the longest directory prefix shared by all paths, ending in a
// path separator. For a single path, that is the directory containing it.
func GetCommonPrefix(separator byte, paths ...string) string {
	switch len(paths) {
	case 0:
		return ""
	case 1:
		directory := filepath.Dir(filepath.Clean(paths[0]))

		// Only a root directory (/ or C:\) already ends in a separator. Adding another gives
		// //, which no path starts with, so nothing is trimmed and the file overwrites itself.
		if os.IsPathSeparator(directory[len(directory)-1]) {
			return directory
		}

		return directory + string(os.PathSeparator)
	}

	// Start with the first path and shorten the prefix to whatever each other path shares.
	prefix := filepath.Clean(paths[0]) + string(separator)

	for _, candidate := range paths[1:] {
		candidate = filepath.Clean(candidate) + string(separator)

		if len(candidate) < len(prefix) {
			prefix = prefix[:len(candidate)]
		}

		for index := range len(prefix) {
			if candidate[index] != prefix[index] {
				prefix = prefix[:index]

				break
			}
		}
	}

	// The shared prefix may end partway through a name ("/a/foo" and "/a/fob" share "/a/fo"),
	// so cut it back to the last separator.
	for index := len(prefix) - 1; index >= 0; index-- {
		if prefix[index] == separator {
			prefix = prefix[:index]

			break
		}
	}

	return prefix + string(os.PathSeparator)
}

// isDirectory reports whether the given path is a directory.
func isDirectory(filePath string) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("could not get file info for %s, %w", filePath, err)
	}

	return fileInfo.IsDir(), nil
}

// getFileNames recursively walks a directory and returns every .yaml and .yml file in it.
func getFileNames(directory string) ([]string, error) {
	var results []string

	err := filepath.Walk(directory,
		func(walkedPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Ext(walkedPath) == ".yaml" || filepath.Ext(walkedPath) == ".yml" {
				results = append(results, walkedPath)
			}

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("could not walk directory %s, %w", directory, err)
	}

	return results, nil
}
