// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package actions

import (
	"strings"
)

const (
	// ValueMarker is a format marker for the previous value.
	ValueMarker = `%v`
	// LineCommentMarker is a format marker for the previous line comment.
	LineCommentMarker = `%l`
	// HeadCommentMarker is a format marker for the previous head comment.
	HeadCommentMarker = `%h`
	// FootCommentMarker is a format marker for the previous foot comment.
	FootCommentMarker = `%f`
	// KeyMarker is a format marker for the previous key name.
	KeyMarker = `%k`
)

func checkForMarkers(decodedYaml string) bool {
	for _, marker := range []string{
		ValueMarker,
		LineCommentMarker,
		HeadCommentMarker,
		FootCommentMarker,
		KeyMarker,
	} {
		if strings.Contains(decodedYaml, marker) {
			return true
		}
	}

	return false
}
