// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's Diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

package diffmatchpatch

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

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

func TestDiffCommonPrefix(t *testing.T) {
	type TestCase struct {
		Name string

		Text1 string
		Text2 string

		Expected int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Null", "abc", "xyz", 0},
		{"Non-null", "1234abcdef", "1234xyz", 4},
		{"Whole", "1234", "1234xyz", 4},
	} {
		actual := dmp.DiffCommonPrefix(tc.Text1, tc.Text2)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func BenchmarkDiffCommonPrefix(b *testing.B) {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ"

	dmp := New()

	for i := 0; i < b.N; i++ {
		dmp.DiffCommonPrefix(s, s)
	}
}

func TestCommonPrefixLength(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected int
	}

	for i, tc := range []TestCase{
		{"abc", "xyz", 0},
		{"1234abcdef", "1234xyz", 4},
		{"1234", "1234xyz", 4},
	} {
		actual := commonPrefixLength([]rune(tc.Text1), []rune(tc.Text2))
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestDiffCommonSuffix(t *testing.T) {
	type TestCase struct {
		Name string

		Text1 string
		Text2 string

		Expected int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Null", "abc", "xyz", 0},
		{"Non-null", "abcdef1234", "xyz1234", 4},
		{"Whole", "1234", "xyz1234", 4},
	} {
		actual := dmp.DiffCommonSuffix(tc.Text1, tc.Text2)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

var SinkInt int // exported sink var to avoid compiler optimizations in benchmarks

func BenchmarkDiffCommonSuffix(b *testing.B) {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ"

	dmp := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SinkInt = dmp.DiffCommonSuffix(s, s)
	}
}

func BenchmarkCommonLength(b *testing.B) {
	data := []struct {
		name string
		x, y []rune
	}{
		{name: "empty", x: nil, y: []rune{}},
		{name: "short", x: []rune("AABCC"), y: []rune("AA-CC")},
		{
			name: "long",
			x:    []rune(strings.Repeat("A", 1000) + "B" + strings.Repeat("C", 1000)),
			y:    []rune(strings.Repeat("A", 1000) + "-" + strings.Repeat("C", 1000)),
		},
	}
	b.Run("prefix", func(b *testing.B) {
		for _, d := range data {
			b.Run(d.name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					SinkInt = commonPrefixLength(d.x, d.y)
				}
			})
		}
	})
	b.Run("suffix", func(b *testing.B) {
		for _, d := range data {
			b.Run(d.name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					SinkInt = commonSuffixLength(d.x, d.y)
				}
			})
		}
	})
}

func TestCommonSuffixLength(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected int
	}

	for i, tc := range []TestCase{
		{"abc", "xyz", 0},
		{"abcdef1234", "xyz1234", 4},
		{"1234", "xyz1234", 4},
		{"123", "a3", 1},
	} {
		actual := commonSuffixLength([]rune(tc.Text1), []rune(tc.Text2))
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestDiffCommonOverlap(t *testing.T) {
	type TestCase struct {
		Name string

		Text1 string
		Text2 string

		Expected int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Null", "", "abcd", 0},
		{"Whole", "abc", "abcd", 3},
		{"Null", "123456", "abcd", 0},
		{"Null", "123456xxx", "xxxabcd", 3},
		// Some overly clever languages (C#) may treat ligatures as equal to their component letters, e.g. U+FB01 == 'fi'
		{"Unicode", "fi", "\ufb01i", 0},
	} {
		actual := dmp.DiffCommonOverlap(tc.Text1, tc.Text2)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffHalfMatch(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected []string
	}

	dmp := New()
	dmp.DiffTimeout = 1

	for i, tc := range []TestCase{
		// No match
		{"1234567890", "abcdef", nil},
		{"12345", "23", nil},

		// Single Match
		{"1234567890", "a345678z", []string{"12", "90", "a", "z", "345678"}},
		{"a345678z", "1234567890", []string{"a", "z", "12", "90", "345678"}},
		{"abc56789z", "1234567890", []string{"abc", "z", "1234", "0", "56789"}},
		{"a23456xyz", "1234567890", []string{"a", "xyz", "1", "7890", "23456"}},

		// Multiple Matches
		{"121231234123451234123121", "a1234123451234z", []string{"12123", "123121", "a", "z", "1234123451234"}},
		{"x-=-=-=-=-=-=-=-=-=-=-=-=", "xx-=-=-=-=-=-=-=", []string{"", "-=-=-=-=-=", "x", "", "x-=-=-=-=-=-=-="}},
		{"-=-=-=-=-=-=-=-=-=-=-=-=y", "-=-=-=-=-=-=-=yy", []string{"-=-=-=-=-=", "", "", "y", "-=-=-=-=-=-=-=y"}},

		// Non-optimal halfmatch, ptimal diff would be -q+x=H-i+e=lloHe+Hu=llo-Hew+y not -qHillo+x=HelloHe-w+Hulloy
		{"qHilloHelloHew", "xHelloHeHulloy", []string{"qHillo", "w", "x", "Hulloy", "HelloHe"}},
	} {
		actual := dmp.DiffHalfMatch(tc.Text1, tc.Text2)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	dmp.DiffTimeout = 0

	for i, tc := range []TestCase{
		// Optimal no halfmatch
		{"qHilloHelloHew", "xHelloHeHulloy", nil},
	} {
		actual := dmp.DiffHalfMatch(tc.Text1, tc.Text2)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func BenchmarkDiffHalfMatch(b *testing.B) {
	s1, s2 := speedtestTexts()

	dmp := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dmp.DiffHalfMatch(s1, s2)
	}
}

func TestDiffBisectSplit(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string
	}

	dmp := New()

	for _, tc := range []TestCase{
		{"STUV\x05WX\x05YZ\x05[", "WĺĻļ\x05YZ\x05ĽľĿŀZ"},
	} {
		diffs := dmp.diffBisectSplit([]rune(tc.Text1),
			[]rune(tc.Text2), 7, 6, time.Now().Add(time.Hour))

		for _, d := range diffs {
			assert.True(t, utf8.ValidString(d.Text))
		}

		// TODO define the expected outcome
	}
}

func TestDiffLinesToChars(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		ExpectedChars1 string
		ExpectedChars2 string
		ExpectedLines  []string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"", "alpha\r\nbeta\r\n\r\n\r\n", "", "\x01\x02\x03\x03", []string{"", "alpha\r\n", "beta\r\n", "\r\n"}},
		{"a", "b", "\x01", "\x02", []string{"", "a", "b"}},
		// Omit final newline.
		{"alpha\nbeta\nalpha", "", "\x01\x02\x03", "", []string{"", "alpha\n", "beta\n", "alpha"}},
		// Same lines in Text1 and Text2
		{"abc\ndefg\n12345\n", "abc\ndef\n12345\n678", "\x01\x02\x03", "\x01\x04\x03\x05", []string{"", "abc\n", "defg\n", "12345\n", "def\n", "678"}},
	} {
		actualChars1, actualChars2, actualLines := dmp.DiffLinesToChars(tc.Text1, tc.Text2)
		assert.Equal(t, tc.ExpectedChars1, actualChars1, fmt.Sprintf("Test case #%d, %#v", i, tc))
		assert.Equal(t, tc.ExpectedChars2, actualChars2, fmt.Sprintf("Test case #%d, %#v", i, tc))
		assert.Equal(t, tc.ExpectedLines, actualLines, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// More than 256 to reveal any 8-bit limitations.
	n := 300
	lineList := []string{
		"", // Account for the initial empty element of the lines array.
	}
	var charList []rune
	for x := 1; x < n+1; x++ {
		lineList = append(lineList, strconv.Itoa(x)+"\n")
		charList = append(charList, rune(x))
	}
	lines := strings.Join(lineList, "")
	chars := string(charList)

	actualChars1, actualChars2, actualLines := dmp.DiffLinesToChars(lines, "")
	assert.Equal(t, chars, actualChars1)
	assert.Equal(t, "", actualChars2)
	assert.Equal(t, lineList, actualLines)
}

func TestDiffCharsToLines(t *testing.T) {
	type TestCase struct {
		Diffs []Diff
		Lines []string

		Expected []Diff
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			Diffs: []Diff{
				{DiffEqual, "\x01\x02\x01"},
				{DiffInsert, "\x02\x01\x02"},
			},
			Lines: []string{"", "alpha\n", "beta\n"},

			Expected: []Diff{
				{DiffEqual, "alpha\nbeta\nalpha\n"},
				{DiffInsert, "beta\nalpha\nbeta\n"},
			},
		},
	} {
		actual := dmp.DiffCharsToLines(tc.Diffs, tc.Lines)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// More than 256 to reveal any 8-bit limitations.
	n := 300
	lineList := []string{
		"", // Account for the initial empty element of the lines array.
	}
	charList := []rune{}
	for x := 1; x <= n; x++ {
		lineList = append(lineList, strconv.Itoa(x)+"\n")
		charList = append(charList, rune(x))
	}
	chars := string(charList)

	actual := dmp.DiffCharsToLines([]Diff{{DiffDelete, chars}}, lineList)
	assert.Equal(t, []Diff{{DiffDelete, strings.Join(lineList, "")}}, actual)
}

func TestDiffCleanupMerge(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []Diff

		Expected []Diff
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			"Null case",
			[]Diff{},
			[]Diff{},
		},
		{
			"No Diff case",
			[]Diff{{DiffEqual, "a"}, {DiffDelete, "b"}, {DiffInsert, "c"}},
			[]Diff{{DiffEqual, "a"}, {DiffDelete, "b"}, {DiffInsert, "c"}},
		},
		{
			"Merge equalities",
			[]Diff{{DiffEqual, "a"}, {DiffEqual, "b"}, {DiffEqual, "c"}},
			[]Diff{{DiffEqual, "abc"}},
		},
		{
			"Merge deletions",
			[]Diff{{DiffDelete, "a"}, {DiffDelete, "b"}, {DiffDelete, "c"}},
			[]Diff{{DiffDelete, "abc"}},
		},
		{
			"Merge insertions",
			[]Diff{{DiffInsert, "a"}, {DiffInsert, "b"}, {DiffInsert, "c"}},
			[]Diff{{DiffInsert, "abc"}},
		},
		{
			"Merge interweave",
			[]Diff{{DiffDelete, "a"}, {DiffInsert, "b"}, {DiffDelete, "c"}, {DiffInsert, "d"}, {DiffEqual, "e"}, {DiffEqual, "f"}},
			[]Diff{{DiffDelete, "ac"}, {DiffInsert, "bd"}, {DiffEqual, "ef"}},
		},
		{
			"Prefix and suffix detection",
			[]Diff{{DiffDelete, "a"}, {DiffInsert, "abc"}, {DiffDelete, "dc"}},
			[]Diff{{DiffEqual, "a"}, {DiffDelete, "d"}, {DiffInsert, "b"}, {DiffEqual, "c"}},
		},
		{
			"Prefix and suffix detection with equalities",
			[]Diff{{DiffEqual, "x"}, {DiffDelete, "a"}, {DiffInsert, "abc"}, {DiffDelete, "dc"}, {DiffEqual, "y"}},
			[]Diff{{DiffEqual, "xa"}, {DiffDelete, "d"}, {DiffInsert, "b"}, {DiffEqual, "cy"}},
		},
		{
			"Same test as above but with unicode (\u0101 will appear in diffs with at least 257 unique lines)",
			[]Diff{{DiffEqual, "x"}, {DiffDelete, "\u0101"}, {DiffInsert, "\u0101bc"}, {DiffDelete, "dc"}, {DiffEqual, "y"}},
			[]Diff{{DiffEqual, "x\u0101"}, {DiffDelete, "d"}, {DiffInsert, "b"}, {DiffEqual, "cy"}},
		},
		{
			"Slide edit left",
			[]Diff{{DiffEqual, "a"}, {DiffInsert, "ba"}, {DiffEqual, "c"}},
			[]Diff{{DiffInsert, "ab"}, {DiffEqual, "ac"}},
		},
		{
			"Slide edit right",
			[]Diff{{DiffEqual, "c"}, {DiffInsert, "ab"}, {DiffEqual, "a"}},
			[]Diff{{DiffEqual, "ca"}, {DiffInsert, "ba"}},
		},
		{
			"Slide edit left recursive",
			[]Diff{{DiffEqual, "a"}, {DiffDelete, "b"}, {DiffEqual, "c"}, {DiffDelete, "ac"}, {DiffEqual, "x"}},
			[]Diff{{DiffDelete, "abc"}, {DiffEqual, "acx"}},
		},
		{
			"Slide edit right recursive",
			[]Diff{{DiffEqual, "x"}, {DiffDelete, "ca"}, {DiffEqual, "c"}, {DiffDelete, "b"}, {DiffEqual, "a"}},
			[]Diff{{DiffEqual, "xca"}, {DiffDelete, "cba"}},
		},
	} {
		actual := dmp.DiffCleanupMerge(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffCleanupSemanticLossless(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []Diff

		Expected []Diff
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			"Null case",
			[]Diff{},
			[]Diff{},
		},
		{
			"Blank lines",
			[]Diff{
				{DiffEqual, "AAA\r\n\r\nBBB"},
				{DiffInsert, "\r\nDDD\r\n\r\nBBB"},
				{DiffEqual, "\r\nEEE"},
			},
			[]Diff{
				{DiffEqual, "AAA\r\n\r\n"},
				{DiffInsert, "BBB\r\nDDD\r\n\r\n"},
				{DiffEqual, "BBB\r\nEEE"},
			},
		},
		{
			"Line boundaries",
			[]Diff{
				{DiffEqual, "AAA\r\nBBB"},
				{DiffInsert, " DDD\r\nBBB"},
				{DiffEqual, " EEE"},
			},
			[]Diff{
				{DiffEqual, "AAA\r\n"},
				{DiffInsert, "BBB DDD\r\n"},
				{DiffEqual, "BBB EEE"},
			},
		},
		{
			"Word boundaries",
			[]Diff{
				{DiffEqual, "The c"},
				{DiffInsert, "ow and the c"},
				{DiffEqual, "at."},
			},
			[]Diff{
				{DiffEqual, "The "},
				{DiffInsert, "cow and the "},
				{DiffEqual, "cat."},
			},
		},
		{
			"Alphanumeric boundaries",
			[]Diff{
				{DiffEqual, "The-c"},
				{DiffInsert, "ow-and-the-c"},
				{DiffEqual, "at."},
			},
			[]Diff{
				{DiffEqual, "The-"},
				{DiffInsert, "cow-and-the-"},
				{DiffEqual, "cat."},
			},
		},
		{
			"Hitting the start",
			[]Diff{
				{DiffEqual, "a"},
				{DiffDelete, "a"},
				{DiffEqual, "ax"},
			},
			[]Diff{
				{DiffDelete, "a"},
				{DiffEqual, "aax"},
			},
		},
		{
			"Hitting the end",
			[]Diff{
				{DiffEqual, "xa"},
				{DiffDelete, "a"},
				{DiffEqual, "a"},
			},
			[]Diff{
				{DiffEqual, "xaa"},
				{DiffDelete, "a"},
			},
		},
		{
			"Sentence boundaries",
			[]Diff{
				{DiffEqual, "The xxx. The "},
				{DiffInsert, "zzz. The "},
				{DiffEqual, "yyy."},
			},
			[]Diff{
				{DiffEqual, "The xxx."},
				{DiffInsert, " The zzz."},
				{DiffEqual, " The yyy."},
			},
		},
		{
			"UTF-8 strings",
			[]Diff{
				{DiffEqual, "The ♕. The "},
				{DiffInsert, "♔. The "},
				{DiffEqual, "♖."},
			},
			[]Diff{
				{DiffEqual, "The ♕."},
				{DiffInsert, " The ♔."},
				{DiffEqual, " The ♖."},
			},
		},
		{
			"Rune boundaries",
			[]Diff{
				{DiffEqual, "♕♕"},
				{DiffInsert, "♔♔"},
				{DiffEqual, "♖♖"},
			},
			[]Diff{
				{DiffEqual, "♕♕"},
				{DiffInsert, "♔♔"},
				{DiffEqual, "♖♖"},
			},
		},
	} {
		actual := dmp.DiffCleanupSemanticLossless(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffCleanupSemantic(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []Diff

		Expected []Diff
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			"Null case",
			[]Diff{},
			[]Diff{},
		},
		{
			"No elimination #1",
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "cd"},
				{DiffEqual, "12"},
				{DiffDelete, "e"},
			},
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "cd"},
				{DiffEqual, "12"},
				{DiffDelete, "e"},
			},
		},
		{
			"No elimination #2",
			[]Diff{
				{DiffDelete, "abc"},
				{DiffInsert, "ABC"},
				{DiffEqual, "1234"},
				{DiffDelete, "wxyz"},
			},
			[]Diff{
				{DiffDelete, "abc"},
				{DiffInsert, "ABC"},
				{DiffEqual, "1234"},
				{DiffDelete, "wxyz"},
			},
		},
		{
			"No elimination #3",
			[]Diff{
				{DiffEqual, "2016-09-01T03:07:1"},
				{DiffInsert, "5.15"},
				{DiffEqual, "4"},
				{DiffDelete, "."},
				{DiffEqual, "80"},
				{DiffInsert, "0"},
				{DiffEqual, "78"},
				{DiffDelete, "3074"},
				{DiffEqual, "1Z"},
			},
			[]Diff{
				{DiffEqual, "2016-09-01T03:07:1"},
				{DiffInsert, "5.15"},
				{DiffEqual, "4"},
				{DiffDelete, "."},
				{DiffEqual, "80"},
				{DiffInsert, "0"},
				{DiffEqual, "78"},
				{DiffDelete, "3074"},
				{DiffEqual, "1Z"},
			},
		},
		{
			"Simple elimination",
			[]Diff{
				{DiffDelete, "a"},
				{DiffEqual, "b"},
				{DiffDelete, "c"},
			},
			[]Diff{
				{DiffDelete, "abc"},
				{DiffInsert, "b"},
			},
		},
		{
			"Backpass elimination",
			[]Diff{
				{DiffDelete, "ab"},
				{DiffEqual, "cd"},
				{DiffDelete, "e"},
				{DiffEqual, "f"},
				{DiffInsert, "g"},
			},
			[]Diff{
				{DiffDelete, "abcdef"},
				{DiffInsert, "cdfg"},
			},
		},
		{
			"Multiple eliminations",
			[]Diff{
				{DiffInsert, "1"},
				{DiffEqual, "A"},
				{DiffDelete, "B"},
				{DiffInsert, "2"},
				{DiffEqual, "_"},
				{DiffInsert, "1"},
				{DiffEqual, "A"},
				{DiffDelete, "B"},
				{DiffInsert, "2"},
			},
			[]Diff{
				{DiffDelete, "AB_AB"},
				{DiffInsert, "1A2_1A2"},
			},
		},
		{
			"Word boundaries",
			[]Diff{
				{DiffEqual, "The c"},
				{DiffDelete, "ow and the c"},
				{DiffEqual, "at."},
			},
			[]Diff{
				{DiffEqual, "The "},
				{DiffDelete, "cow and the "},
				{DiffEqual, "cat."},
			},
		},
		{
			"No overlap elimination",
			[]Diff{
				{DiffDelete, "abcxx"},
				{DiffInsert, "xxdef"},
			},
			[]Diff{
				{DiffDelete, "abcxx"},
				{DiffInsert, "xxdef"},
			},
		},
		{
			"Overlap elimination",
			[]Diff{
				{DiffDelete, "abcxxx"},
				{DiffInsert, "xxxdef"},
			},
			[]Diff{
				{DiffDelete, "abc"},
				{DiffEqual, "xxx"},
				{DiffInsert, "def"},
			},
		},
		{
			"Reverse overlap elimination",
			[]Diff{
				{DiffDelete, "xxxabc"},
				{DiffInsert, "defxxx"},
			},
			[]Diff{
				{DiffInsert, "def"},
				{DiffEqual, "xxx"},
				{DiffDelete, "abc"},
			},
		},
		{
			"Two overlap eliminations",
			[]Diff{
				{DiffDelete, "abcd1212"},
				{DiffInsert, "1212efghi"},
				{DiffEqual, "----"},
				{DiffDelete, "A3"},
				{DiffInsert, "3BC"},
			},
			[]Diff{
				{DiffDelete, "abcd"},
				{DiffEqual, "1212"},
				{DiffInsert, "efghi"},
				{DiffEqual, "----"},
				{DiffDelete, "A"},
				{DiffEqual, "3"},
				{DiffInsert, "BC"},
			},
		},
		{
			"Test case for adapting DiffCleanupSemantic to be equal to the Python version #19",
			[]Diff{
				{DiffEqual, "James McCarthy "},
				{DiffDelete, "close to "},
				{DiffEqual, "sign"},
				{DiffDelete, "ing"},
				{DiffInsert, "s"},
				{DiffEqual, " new "},
				{DiffDelete, "E"},
				{DiffInsert, "fi"},
				{DiffEqual, "ve"},
				{DiffInsert, "-yea"},
				{DiffEqual, "r"},
				{DiffDelete, "ton"},
				{DiffEqual, " deal"},
				{DiffInsert, " at Everton"},
			},
			[]Diff{
				{DiffEqual, "James McCarthy "},
				{DiffDelete, "close to "},
				{DiffEqual, "sign"},
				{DiffDelete, "ing"},
				{DiffInsert, "s"},
				{DiffEqual, " new "},
				{DiffInsert, "five-year deal at "},
				{DiffEqual, "Everton"},
				{DiffDelete, " deal"},
			},
		},
		{
			"Taken from python / CPP library",
			[]Diff{
				{DiffInsert, "星球大戰：新的希望 "},
				{DiffEqual, "star wars: "},
				{DiffDelete, "episodio iv - un"},
				{DiffEqual, "a n"},
				{DiffDelete, "u"},
				{DiffEqual, "e"},
				{DiffDelete, "va"},
				{DiffInsert, "w"},
				{DiffEqual, " "},
				{DiffDelete, "es"},
				{DiffInsert, "ho"},
				{DiffEqual, "pe"},
				{DiffDelete, "ranza"},
			},
			[]Diff{
				{DiffInsert, "星球大戰：新的希望 "},
				{DiffEqual, "star wars: "},
				{DiffDelete, "episodio iv - una nueva esperanza"},
				{DiffInsert, "a new hope"},
			},
		},
		{
			"panic",
			[]Diff{
				{DiffInsert, "킬러 인 "},
				{DiffEqual, "리커버리"},
				{DiffDelete, " 보이즈"},
			},
			[]Diff{
				{DiffInsert, "킬러 인 "},
				{DiffEqual, "리커버리"},
				{DiffDelete, " 보이즈"},
			},
		},
	} {
		actual := dmp.DiffCleanupSemantic(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func BenchmarkDiffCleanupSemantic(b *testing.B) {
	s1, s2 := speedtestTexts()

	dmp := New()

	diffs := dmp.DiffMain(s1, s2, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dmp.DiffCleanupSemantic(diffs)
	}
}

func TestDiffCleanupEfficiency(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []Diff

		Expected []Diff
	}

	dmp := New()
	dmp.DiffEditCost = 4

	for i, tc := range []TestCase{
		{
			"Null case",
			[]Diff{},
			[]Diff{},
		},
		{
			"No elimination",
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "wxyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "wxyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
		},
		{
			"Four-edit elimination",
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "xyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]Diff{
				{DiffDelete, "abxyzcd"},
				{DiffInsert, "12xyz34"},
			},
		},
		{
			"Three-edit elimination",
			[]Diff{
				{DiffInsert, "12"},
				{DiffEqual, "x"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]Diff{
				{DiffDelete, "xcd"},
				{DiffInsert, "12x34"},
			},
		},
		{
			"Backpass elimination",
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "xy"},
				{DiffInsert, "34"},
				{DiffEqual, "z"},
				{DiffDelete, "cd"},
				{DiffInsert, "56"},
			},
			[]Diff{
				{DiffDelete, "abxyzcd"},
				{DiffInsert, "12xy34z56"},
			},
		},
	} {
		actual := dmp.DiffCleanupEfficiency(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.DiffEditCost = 5

	for i, tc := range []TestCase{
		{
			"High cost elimination",
			[]Diff{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "wxyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]Diff{
				{DiffDelete, "abwxyzcd"},
				{DiffInsert, "12wxyz34"},
			},
		},
	} {
		actual := dmp.DiffCleanupEfficiency(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffPrettyHtml(t *testing.T) {
	type TestCase struct {
		Diffs []Diff

		Expected string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			Diffs: []Diff{
				{DiffEqual, "a\n"},
				{DiffDelete, "<B>b</B>"},
				{DiffInsert, "c&d"},
			},

			Expected: "<span>a&para;<br></span><del style=\"background:#ffe6e6;\">&lt;B&gt;b&lt;/B&gt;</del><ins style=\"background:#e6ffe6;\">c&amp;d</ins>",
		},
	} {
		actual := dmp.DiffPrettyHtml(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestDiffPrettyText(t *testing.T) {
	type TestCase struct {
		Diffs []Diff

		Expected string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			Diffs: []Diff{
				{DiffEqual, "a\n"},
				{DiffDelete, "<B>b</B>"},
				{DiffInsert, "c&d"},
			},

			Expected: "a\n\x1b[31m<B>b</B>\x1b[0m\x1b[32mc&d\x1b[0m",
		},
	} {
		actual := dmp.DiffPrettyText(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestDiffText(t *testing.T) {
	type TestCase struct {
		Diffs []Diff

		ExpectedText1 string
		ExpectedText2 string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			Diffs: []Diff{
				{DiffEqual, "jump"},
				{DiffDelete, "s"},
				{DiffInsert, "ed"},
				{DiffEqual, " over "},
				{DiffDelete, "the"},
				{DiffInsert, "a"},
				{DiffEqual, " lazy"},
			},

			ExpectedText1: "jumps over the lazy",
			ExpectedText2: "jumped over a lazy",
		},
	} {
		actualText1 := dmp.DiffText1(tc.Diffs)
		assert.Equal(t, tc.ExpectedText1, actualText1, fmt.Sprintf("Test case #%d, %#v", i, tc))

		actualText2 := dmp.DiffText2(tc.Diffs)
		assert.Equal(t, tc.ExpectedText2, actualText2, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestDiffDelta(t *testing.T) {
	type TestCase struct {
		Name string

		Text  string
		Delta string

		ErrorMessagePrefix string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"delta shorter than text", "jumps over the lazyx", "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", "delta length (19) is different from source text length (20)"},
		{"delta longer than text", "umps over the lazy", "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", "delta length (19) is different from source text length (18)"},
		{"invalid URL escaping", "", "+%c3%xy", "invalid URL escape \"%xy\""},
		{"invalid UTF-8 sequence", "", "+%c3xy", "invalid UTF-8 token: \"\\xc3xy\""},
		{"invalid diff operation", "", "a", "invalid diff operation in DiffFromDelta: a"},
		{"invalid diff syntax", "", "-", "strconv.ParseInt: parsing \"\": invalid syntax"},
		{"negative number in delta", "", "--1", "negative number in DiffFromDelta: -1"},
		{"empty case", "", "", ""},
	} {
		diffs, err := dmp.DiffFromDelta(tc.Text, tc.Delta)
		msg := fmt.Sprintf("Test case #%d, %s", i, tc.Name)
		if tc.ErrorMessagePrefix == "" {
			assert.Nil(t, err, msg)
			assert.Nil(t, diffs, msg)
		} else {
			e := err.Error()
			if strings.HasPrefix(e, tc.ErrorMessagePrefix) {
				e = tc.ErrorMessagePrefix
			}
			assert.Nil(t, diffs, msg)
			assert.Equal(t, tc.ErrorMessagePrefix, e, msg)
		}
	}

	// Convert a diff into delta string.
	diffs := []Diff{
		{DiffEqual, "jump"},
		{DiffDelete, "s"},
		{DiffInsert, "ed"},
		{DiffEqual, " over "},
		{DiffDelete, "the"},
		{DiffInsert, "a"},
		{DiffEqual, " lazy"},
		{DiffInsert, "old dog"},
	}
	text1 := dmp.DiffText1(diffs)
	assert.Equal(t, "jumps over the lazy", text1)

	delta := dmp.DiffToDelta(diffs)
	assert.Equal(t, "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", delta)

	// Convert delta string into a diff.
	deltaDiffs, err := dmp.DiffFromDelta(text1, delta)
	assert.NoError(t, err)
	assert.Equal(t, diffs, deltaDiffs)

	// Test deltas with special characters.
	diffs = []Diff{
		{DiffEqual, "\u0680 \x00 \t %"},
		{DiffDelete, "\u0681 \x01 \n ^"},
		{DiffInsert, "\u0682 \x02 \\ |"},
	}
	text1 = dmp.DiffText1(diffs)
	assert.Equal(t, "\u0680 \x00 \t %\u0681 \x01 \n ^", text1)

	// Lowercase, due to UrlEncode uses lower.
	delta = dmp.DiffToDelta(diffs)
	assert.Equal(t, "=7\t-7\t+%DA%82 %02 %5C %7C", delta)

	deltaDiffs, err = dmp.DiffFromDelta(text1, delta)
	assert.Equal(t, diffs, deltaDiffs)
	assert.Nil(t, err)

	// Verify pool of unchanged characters.
	diffs = []Diff{
		{DiffInsert, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # "},
	}

	delta = dmp.DiffToDelta(diffs)
	assert.Equal(t, "+A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # ", delta, "Unchanged characters.")

	// Convert delta string into a diff.
	deltaDiffs, err = dmp.DiffFromDelta("", delta)
	assert.Equal(t, diffs, deltaDiffs)
	assert.Nil(t, err)
}

func TestDiffXIndex(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs    []Diff
		Location int

		Expected int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Translation on equality", []Diff{{DiffDelete, "a"}, {DiffInsert, "1234"}, {DiffEqual, "xyz"}}, 2, 5},
		{"Translation on deletion", []Diff{{DiffEqual, "a"}, {DiffDelete, "1234"}, {DiffEqual, "xyz"}}, 3, 1},
	} {
		actual := dmp.DiffXIndex(tc.Diffs, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffLevenshtein(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []Diff

		Expected int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Levenshtein with trailing equality", []Diff{{DiffDelete, "абв"}, {DiffInsert, "1234"}, {DiffEqual, "эюя"}}, 4},
		{"Levenshtein with leading equality", []Diff{{DiffEqual, "эюя"}, {DiffDelete, "абв"}, {DiffInsert, "1234"}}, 4},
		{"Levenshtein with middle equality", []Diff{{DiffDelete, "абв"}, {DiffEqual, "эюя"}, {DiffInsert, "1234"}}, 7},
	} {
		actual := dmp.DiffLevenshtein(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffBisect(t *testing.T) {
	type TestCase struct {
		Name string

		Time time.Time

		Expected []Diff
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			Name: "normal",
			Time: time.Date(9999, time.December, 31, 23, 59, 59, 59, time.UTC),

			Expected: []Diff{
				{DiffDelete, "c"},
				{DiffInsert, "m"},
				{DiffEqual, "a"},
				{DiffDelete, "t"},
				{DiffInsert, "p"},
			},
		},
		{
			Name: "Negative deadlines count as having infinite time",
			Time: time.Date(0o001, time.January, 0o1, 0o0, 0o0, 0o0, 0o0, time.UTC),

			Expected: []Diff{
				{DiffDelete, "c"},
				{DiffInsert, "m"},
				{DiffEqual, "a"},
				{DiffDelete, "t"},
				{DiffInsert, "p"},
			},
		},
		{
			Name: "Timeout",
			Time: time.Now().Add(time.Nanosecond),

			Expected: []Diff{
				{DiffDelete, "cat"},
				{DiffInsert, "map"},
			},
		},
	} {
		actual := dmp.DiffBisect("cat", "map", tc.Time)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	// Test for invalid UTF-8 sequences
	assert.Equal(t, []Diff{
		{DiffEqual, "��"},
	}, dmp.DiffBisect("\xe0\xe5", "\xe0\xe5", time.Now().Add(time.Minute)))
}

func TestDiffMain(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected []Diff
	}

	dmp := New()

	// Perform a trivial diff.
	for i, tc := range []TestCase{
		{
			"",
			"",
			nil,
		},
		{
			"abc",
			"abc",
			[]Diff{{DiffEqual, "abc"}},
		},
		{
			"abc",
			"ab123c",
			[]Diff{{DiffEqual, "ab"}, {DiffInsert, "123"}, {DiffEqual, "c"}},
		},
		{
			"a123bc",
			"abc",
			[]Diff{{DiffEqual, "a"}, {DiffDelete, "123"}, {DiffEqual, "bc"}},
		},
		{
			"abc",
			"a123b456c",
			[]Diff{{DiffEqual, "a"}, {DiffInsert, "123"}, {DiffEqual, "b"}, {DiffInsert, "456"}, {DiffEqual, "c"}},
		},
		{
			"a123b456c",
			"abc",
			[]Diff{{DiffEqual, "a"}, {DiffDelete, "123"}, {DiffEqual, "b"}, {DiffDelete, "456"}, {DiffEqual, "c"}},
		},
	} {
		actual := dmp.DiffMain(tc.Text1, tc.Text2, false)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// Perform a real diff and switch off the timeout.
	dmp.DiffTimeout = 0

	for i, tc := range []TestCase{
		{
			"a",
			"b",
			[]Diff{{DiffDelete, "a"}, {DiffInsert, "b"}},
		},
		{
			"Apples are a fruit.",
			"Bananas are also fruit.",
			[]Diff{
				{DiffDelete, "Apple"},
				{DiffInsert, "Banana"},
				{DiffEqual, "s are a"},
				{DiffInsert, "lso"},
				{DiffEqual, " fruit."},
			},
		},
		{
			"ax\t",
			"\u0680x\u0000",
			[]Diff{
				{DiffDelete, "a"},
				{DiffInsert, "\u0680"},
				{DiffEqual, "x"},
				{DiffDelete, "\t"},
				{DiffInsert, "\u0000"},
			},
		},
		{
			"1ayb2",
			"abxab",
			[]Diff{
				{DiffDelete, "1"},
				{DiffEqual, "a"},
				{DiffDelete, "y"},
				{DiffEqual, "b"},
				{DiffDelete, "2"},
				{DiffInsert, "xab"},
			},
		},
		{
			"abcy",
			"xaxcxabc",
			[]Diff{
				{DiffInsert, "xaxcx"},
				{DiffEqual, "abc"},
				{DiffDelete, "y"},
			},
		},
		{
			"ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg",
			"a-bcd-efghijklmnopqrs",
			[]Diff{
				{DiffDelete, "ABCD"},
				{DiffEqual, "a"},
				{DiffDelete, "="},
				{DiffInsert, "-"},
				{DiffEqual, "bcd"},
				{DiffDelete, "="},
				{DiffInsert, "-"},
				{DiffEqual, "efghijklmnopqrs"},
				{DiffDelete, "EFGHIJKLMNOefg"},
			},
		},
		{
			"a [[Pennsylvania]] and [[New",
			" and [[Pennsylvania]]",
			[]Diff{
				{DiffInsert, " "},
				{DiffEqual, "a"},
				{DiffInsert, "nd"},
				{DiffEqual, " [[Pennsylvania]]"},
				{DiffDelete, " and [[New"},
			},
		},
	} {
		actual := dmp.DiffMain(tc.Text1, tc.Text2, false)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// Test for invalid UTF-8 sequences
	assert.Equal(t, []Diff{
		{DiffDelete, "��"},
	}, dmp.DiffMain("\xe0\xe5", "", false))
}

func TestDiffMainWithTimeout(t *testing.T) {
	dmp := New()
	dmp.DiffTimeout = 200 * time.Millisecond

	a := "`Twas brillig, and the slithy toves\nDid gyre and gimble in the wabe:\nAll mimsy were the borogoves,\nAnd the mome raths outgrabe.\n"
	b := "I am the very model of a modern major general,\nI've information vegetable, animal, and mineral,\nI know the kings of England, and I quote the fights historical,\nFrom Marathon to Waterloo, in order categorical.\n"
	// Increase the text lengths by 1024 times to ensure a timeout.
	for x := 0; x < 13; x++ {
		a = a + a
		b = b + b
	}

	startTime := time.Now()
	dmp.DiffMain(a, b, true)
	endTime := time.Now()

	delta := endTime.Sub(startTime)

	// Test that we took at least the timeout period.
	assert.True(t, delta >= dmp.DiffTimeout, fmt.Sprintf("%v !>= %v", delta, dmp.DiffTimeout))

	// Test that we didn't take forever (be very forgiving). Theoretically this test could fail very occasionally if the OS task swaps or locks up for a second at the wrong moment.
	assert.True(t, delta < (dmp.DiffTimeout*100), fmt.Sprintf("%v !< %v", delta, dmp.DiffTimeout*100))
}

func TestDiffMainWithCheckLines(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string
	}

	dmp := New()
	dmp.DiffTimeout = 0

	// Test cases must be at least 100 chars long to pass the cutoff.
	for i, tc := range []TestCase{
		{
			"1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n",
			"abcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\n",
		},
		{
			"1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			"abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij",
		},
		{
			"1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n",
			"abcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n",
		},
	} {
		resultWithoutCheckLines := dmp.DiffMain(tc.Text1, tc.Text2, false)
		resultWithCheckLines := dmp.DiffMain(tc.Text1, tc.Text2, true)

		// TODO this fails for the third test case, why?
		if i != 2 {
			assert.Equal(t, resultWithoutCheckLines, resultWithCheckLines, fmt.Sprintf("Test case #%d, %#v", i, tc))
		}
		assert.Equal(t, diffRebuildTexts(resultWithoutCheckLines), diffRebuildTexts(resultWithCheckLines), fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestMassiveRuneDiffConversion(t *testing.T) {
	sNew, err := os.ReadFile("../testdata/fixture.go")
	if err != nil {
		panic(err)
	}

	dmp := New()
	t1, t2, tt := dmp.DiffLinesToChars("", string(sNew))
	diffs := dmp.DiffMain(t1, t2, false)
	diffs = dmp.DiffCharsToLines(diffs, tt)
	assert.NotEmpty(t, diffs)
}

func TestDiffPartialLineIndex(t *testing.T) {
	dmp := New()
	t1, t2, tt := dmp.DiffLinesToChars(
		`line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10 text1`,
		`line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10 text2`)
	diffs := dmp.DiffMain(t1, t2, false)
	diffs = dmp.DiffCharsToLines(diffs, tt)
	assert.Equal(t, []Diff{
		{DiffEqual, "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7\nline 8\nline 9\n"},
		{DiffDelete, "line 10 text1"},
		{DiffInsert, "line 10 text2"},
	}, diffs)
}

func BenchmarkDiffMain(bench *testing.B) {
	s1 := "`Twas brillig, and the slithy toves\nDid gyre and gimble in the wabe:\nAll mimsy were the borogoves,\nAnd the mome raths outgrabe.\n"
	s2 := "I am the very model of a modern major general,\nI've information vegetable, animal, and mineral,\nI know the kings of England, and I quote the fights historical,\nFrom Marathon to Waterloo, in order categorical.\n"

	// Increase the text lengths by 1024 times to ensure a timeout.
	for x := 0; x < 10; x++ {
		s1 = s1 + s1
		s2 = s2 + s2
	}

	dmp := New()
	dmp.DiffTimeout = time.Second

	bench.ResetTimer()

	for i := 0; i < bench.N; i++ {
		dmp.DiffMain(s1, s2, true)
	}
}

func BenchmarkDiffMainLarge(b *testing.B) {
	s1, s2 := speedtestTexts()

	dmp := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dmp.DiffMain(s1, s2, true)
	}
}

func BenchmarkDiffMainRunesLargeLines(b *testing.B) {
	s1, s2 := speedtestTexts()

	dmp := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		text1, text2, linearray := dmp.DiffLinesToRunes(s1, s2)

		diffs := dmp.DiffMainRunes(text1, text2, false)
		dmp.DiffCharsToLines(diffs, linearray)
	}
}

func BenchmarkDiffMainRunesLargeDiffLines(b *testing.B) {
	fp, _ := os.Open("../testdata/diff10klinestest.txt")
	defer fp.Close()
	data, _ := io.ReadAll(fp)

	dmp := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		text1, text2, linearray := dmp.DiffLinesToRunes(string(data), "")

		diffs := dmp.DiffMainRunes(text1, text2, false)
		dmp.DiffCharsToLines(diffs, linearray)
	}
}
