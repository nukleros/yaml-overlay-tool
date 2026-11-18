// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

package instructions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/instructions"
)

// Paths used by more than one TestGetCommonPrefix case.
const (
	prefixRootFile  = "/x.yaml"
	prefixHomeDir   = "/home/u/"
	prefixManifestA = "/home/u/m/a.yaml"
)

// TestGetCommonPrefix checks the directory that derived output paths are made relative to.
// Paths are written with forward slashes and converted, so the cases also hold on Windows.
func TestGetCommonPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{
			// A lone file in the root used to give "//", so its output path stayed absolute
			// and the file was written over itself.
			name:  "single file in the root directory",
			paths: []string{prefixRootFile},
			want:  "/",
		},
		{
			name:  "single file",
			paths: []string{"/home/u/app.yaml"},
			want:  prefixHomeDir,
		},
		{
			name:  "files in the same directory",
			paths: []string{prefixManifestA, "/home/u/m/b.yaml"},
			want:  "/home/u/m/",
		},
		{
			name:  "files in different directories",
			paths: []string{prefixManifestA, "/home/u/n/b.yaml"},
			want:  prefixHomeDir,
		},
		{
			name:  "files in different top-level directories",
			paths: []string{"/a/x.yaml", "/b/y.yaml"},
			want:  "/",
		},
		{
			name:  "files in the root directory",
			paths: []string{prefixRootFile, "/y.yaml"},
			want:  "/",
		},
		{
			name:  "directory names that share a partial prefix",
			paths: []string{"/a/foo/x.yaml", "/a/fob/y.yaml"},
			want:  "/a/",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			paths := make([]string, len(test.paths))
			for index, path := range test.paths {
				paths[index] = filepath.FromSlash(path)
			}

			got := instructions.GetCommonPrefix(os.PathSeparator, paths...)
			if want := filepath.FromSlash(test.want); got != want {
				t.Errorf("GetCommonPrefix(%q) = %q, want %q", paths, got, want)
			}
		})
	}
}
