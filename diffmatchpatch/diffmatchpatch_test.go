// Copyright (c) 2012 Sergi Mansilla <sergi.mansilla@gmail.com>
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's Diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

package diffmatchpatch

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func caller() string {
	_, _, line, ok := runtime.Caller(2)
	if !ok {
		return ""
	}

	return fmt.Sprintf("at line %d ", line)
}

func pretty(diffs []Diff) string {
	var w bytes.Buffer

	for i, diff := range diffs {
		_, _ = w.WriteString(fmt.Sprintf("%v. ", i))

		switch diff.Type {
		case DiffInsert:
			_, _ = w.WriteString("DiffIns")
		case DiffDelete:
			_, _ = w.WriteString("DiffDel")
		case DiffEqual:
			_, _ = w.WriteString("DiffEql")
		default:
			_, _ = w.WriteString("Unknown")
		}

		_, _ = w.WriteString(fmt.Sprintf(": %v\n", diff.Text))
	}

	return w.String()
}

func diffRebuildTexts(diffs []Diff) []string {
	texts := []string{"", ""}

	for _, d := range diffs {
		if d.Type != DiffInsert {
			texts[0] += d.Text
		}
		if d.Type != DiffDelete {
			texts[1] += d.Text
		}
	}

	return texts
}

func readFile(filepath string) string {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func speedtestTexts() (s1 string, s2 string) {
	s1 = readFile("../testdata/speedtest1.txt")
	s2 = readFile("../testdata/speedtest2.txt")

	return s1, s2
}

func TestRunesIndexOf(t *testing.T) {
	type TestCase struct {
		Pattern string
		Start   int

		Expected int
	}

	for i, tc := range []TestCase{
		{"abc", 0, 0},
		{"cde", 0, 2},
		{"e", 0, 4},
		{"cdef", 0, -1},
		{"abcdef", 0, -1},
		{"abc", 2, -1},
		{"cde", 2, 2},
		{"e", 2, 4},
		{"cdef", 2, -1},
		{"abcdef", 2, -1},
		{"e", 6, -1},
	} {
		actual := runesIndexOf([]rune("abcde"), []rune(tc.Pattern), tc.Start)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestIndexOf(t *testing.T) {
	type TestCase struct {
		String   string
		Pattern  string
		Position int

		Expected int
	}

	for i, tc := range []TestCase{
		{"hi world", "world", -1, 3},
		{"hi world", "world", 0, 3},
		{"hi world", "world", 1, 3},
		{"hi world", "world", 2, 3},
		{"hi world", "world", 3, 3},
		{"hi world", "world", 4, -1},
		{"abbc", "b", -1, 1},
		{"abbc", "b", 0, 1},
		{"abbc", "b", 1, 1},
		{"abbc", "b", 2, 2},
		{"abbc", "b", 3, -1},
		{"abbc", "b", 4, -1},
		// The greek letter beta is the two-byte sequence of "\u03b2".
		{"a\u03b2\u03b2c", "\u03b2", -1, 1},
		{"a\u03b2\u03b2c", "\u03b2", 0, 1},
		{"a\u03b2\u03b2c", "\u03b2", 1, 1},
		{"a\u03b2\u03b2c", "\u03b2", 3, 3},
		{"a\u03b2\u03b2c", "\u03b2", 5, -1},
		{"a\u03b2\u03b2c", "\u03b2", 6, -1},
	} {
		actual := indexOf(tc.String, tc.Pattern, tc.Position)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestLastIndexOf(t *testing.T) {
	type TestCase struct {
		String   string
		Pattern  string
		Position int

		Expected int
	}

	for i, tc := range []TestCase{
		{"hi world", "world", -1, -1},
		{"hi world", "world", 0, -1},
		{"hi world", "world", 1, -1},
		{"hi world", "world", 2, -1},
		{"hi world", "world", 3, -1},
		{"hi world", "world", 4, -1},
		{"hi world", "world", 5, -1},
		{"hi world", "world", 6, -1},
		{"hi world", "world", 7, 3},
		{"hi world", "world", 8, 3},
		{"abbc", "b", -1, -1},
		{"abbc", "b", 0, -1},
		{"abbc", "b", 1, 1},
		{"abbc", "b", 2, 2},
		{"abbc", "b", 3, 2},
		{"abbc", "b", 4, 2},
		// The greek letter beta is the two-byte sequence of "\u03b2".
		{"a\u03b2\u03b2c", "\u03b2", -1, -1},
		{"a\u03b2\u03b2c", "\u03b2", 0, -1},
		{"a\u03b2\u03b2c", "\u03b2", 1, 1},
		{"a\u03b2\u03b2c", "\u03b2", 3, 3},
		{"a\u03b2\u03b2c", "\u03b2", 5, 3},
		{"a\u03b2\u03b2c", "\u03b2", 6, 3},
	} {
		actual := lastIndexOf(tc.String, tc.Pattern, tc.Position)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}
