// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's Diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

// Package diffmatchpatch offers robust algorithms to perform the
// operations required for synchronizing plain text.
package diffmatchpatch

import (
	"strings"
	"time"
	"unicode/utf8"
)

// unescaper unescapes selected chars for compatibility with JavaScript's encodeURI.
// In speed critical applications this could be dropped since the
// receiving application will certainly decode these fine.
// Note that this function is case-sensitive.  Thus "%3F" would not be
// unescaped.  But this is ok because it is only called with the output of
// HttpUtility.UrlEncode which returns lowercase hex.
//
// Example: "%3f" -> "?", "%24" -> "$", etc.
var unescaper = strings.NewReplacer(
	"%21", "!", "%7E", "~", "%27", "'",
	"%28", "(", "%29", ")", "%3B", ";",
	"%2F", "/", "%3F", "?", "%3A", ":",
	"%40", "@", "%26", "&", "%3D", "=",
	"%2B", "+", "%24", "$", "%2C", ",", "%23", "#", "%2A", "*")

// indexOf returns the first index of pattern in str, starting at str[i].
func indexOf(str string, pattern string, i int) int {
	if i > len(str)-1 {
		return -1
	}
	if i <= 0 {
		return strings.Index(str, pattern)
	}
	ind := strings.Index(str[i:], pattern)
	if ind == -1 {
		return -1
	}
	return ind + i
}

// lastIndexOf returns the last index of pattern in str, starting at str[i].
func lastIndexOf(str string, pattern string, i int) int {
	if i < 0 {
		return -1
	}
	if i >= len(str) {
		return strings.LastIndex(str, pattern)
	}
	_, size := utf8.DecodeRuneInString(str[i:])
	return strings.LastIndex(str[:i+size], pattern)
}

// Return the index of pattern in target, starting at target[i].
func runesIndexOf(target, pattern []rune, i int) int {
	if i > len(target)-1 {
		return -1
	}
	if i <= 0 {
		return runesIndex(target, pattern)
	}
	ind := runesIndex(target[i:], pattern)
	if ind == -1 {
		return -1
	}
	return ind + i
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func runesEqual(r1, r2 []rune) bool {
	if len(r1) != len(r2) {
		return false
	}
	for i, c := range r1 {
		if c != r2[i] {
			return false
		}
	}
	return true
}

// The equivalent of strings.Index for rune slices.
func runesIndex(r1, r2 []rune) int {
	last := len(r1) - len(r2)
	for i := 0; i <= last; i++ {
		if runesEqual(r1[i:i+len(r2)], r2) {
			return i
		}
	}
	return -1
}

// DiffMatchPatch holds the configuration for diff-match-patch operations.
type DiffMatchPatch struct {
	// Number of seconds to map a diff before giving up (0 for infinity).
	DiffTimeout time.Duration
	// Cost of an empty edit operation in terms of edit characters.
	DiffEditCost int
	// How far to search for a match (0 = exact location, 1000+ = broad match).
	// A match this many characters away from the expected location will add
	// 1.0 to the score (0.0 is a perfect match).
	MatchDistance int
	// When deleting a large block of text (over ~64 characters), how close do
	// the contents have to be to match the expected contents. (0.0 = perfection,
	// 1.0 = very loose).  Note that MatchThreshold controls how closely the
	// end points of a delete need to match.
	PatchDeleteThreshold float64
	// Chunk size for context length.
	PatchMargin int
	// The number of bits in an int.
	MatchMaxBits int
	// At what point is no match declared (0.0 = perfection, 1.0 = very loose).
	MatchThreshold float64
}

// New creates a new DiffMatchPatch object with default parameters.
func New() *DiffMatchPatch {
	// Defaults.
	return &DiffMatchPatch{
		DiffTimeout:          time.Second,
		DiffEditCost:         4,
		MatchThreshold:       0.5,
		MatchDistance:        1000,
		PatchDeleteThreshold: 0.5,
		PatchMargin:          4,
		MatchMaxBits:         32,
	}
}
