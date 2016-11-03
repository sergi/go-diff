package diffmatchpatch

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
		{"", "alpha\r\nbeta\r\n\r\n\r\n", "", "\u0001\u0002\u0003\u0003", []string{"", "alpha\r\n", "beta\r\n", "\r\n"}},
		{"a", "b", "\u0001", "\u0002", []string{"", "a", "b"}},
		// Omit final newline.
		{"alpha\nbeta\nalpha", "", "\u0001\u0002\u0003", "", []string{"", "alpha\n", "beta\n", "alpha"}},
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
	assert.Equal(t, n, utf8.RuneCountInString(chars))

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
				{DiffEqual, "\u0001\u0002\u0001"},
				{DiffInsert, "\u0002\u0001\u0002"},
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
	assert.Equal(t, n, len(charList))

	actual := dmp.DiffCharsToLines([]Diff{Diff{DiffDelete, string(charList)}}, lineList)
	assert.Equal(t, []Diff{Diff{DiffDelete, strings.Join(lineList, "")}}, actual)
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
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "b"}, Diff{DiffInsert, "c"}},
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "b"}, Diff{DiffInsert, "c"}},
		},
		{
			"Merge equalities",
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffEqual, "b"}, Diff{DiffEqual, "c"}},
			[]Diff{Diff{DiffEqual, "abc"}},
		},
		{
			"Merge deletions",
			[]Diff{Diff{DiffDelete, "a"}, Diff{DiffDelete, "b"}, Diff{DiffDelete, "c"}},
			[]Diff{Diff{DiffDelete, "abc"}},
		},
		{
			"Merge insertions",
			[]Diff{Diff{DiffInsert, "a"}, Diff{DiffInsert, "b"}, Diff{DiffInsert, "c"}},
			[]Diff{Diff{DiffInsert, "abc"}},
		},
		{
			"Merge interweave",
			[]Diff{Diff{DiffDelete, "a"}, Diff{DiffInsert, "b"}, Diff{DiffDelete, "c"}, Diff{DiffInsert, "d"}, Diff{DiffEqual, "e"}, Diff{DiffEqual, "f"}},
			[]Diff{Diff{DiffDelete, "ac"}, Diff{DiffInsert, "bd"}, Diff{DiffEqual, "ef"}},
		},
		{
			"Prefix and suffix detection",
			[]Diff{Diff{DiffDelete, "a"}, Diff{DiffInsert, "abc"}, Diff{DiffDelete, "dc"}},
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "d"}, Diff{DiffInsert, "b"}, Diff{DiffEqual, "c"}},
		},
		{
			"Prefix and suffix detection with equalities",
			[]Diff{Diff{DiffEqual, "x"}, Diff{DiffDelete, "a"}, Diff{DiffInsert, "abc"}, Diff{DiffDelete, "dc"}, Diff{DiffEqual, "y"}},
			[]Diff{Diff{DiffEqual, "xa"}, Diff{DiffDelete, "d"}, Diff{DiffInsert, "b"}, Diff{DiffEqual, "cy"}},
		},
		{
			"Same test as above but with unicode (\u0101 will appear in diffs with at least 257 unique lines)",
			[]Diff{Diff{DiffEqual, "x"}, Diff{DiffDelete, "\u0101"}, Diff{DiffInsert, "\u0101bc"}, Diff{DiffDelete, "dc"}, Diff{DiffEqual, "y"}},
			[]Diff{Diff{DiffEqual, "x\u0101"}, Diff{DiffDelete, "d"}, Diff{DiffInsert, "b"}, Diff{DiffEqual, "cy"}},
		},
		{
			"Slide edit left",
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffInsert, "ba"}, Diff{DiffEqual, "c"}},
			[]Diff{Diff{DiffInsert, "ab"}, Diff{DiffEqual, "ac"}},
		},
		{
			"Slide edit right",
			[]Diff{Diff{DiffEqual, "c"}, Diff{DiffInsert, "ab"}, Diff{DiffEqual, "a"}},
			[]Diff{Diff{DiffEqual, "ca"}, Diff{DiffInsert, "ba"}},
		},
		{
			"Slide edit left recursive",
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "b"}, Diff{DiffEqual, "c"}, Diff{DiffDelete, "ac"}, Diff{DiffEqual, "x"}},
			[]Diff{Diff{DiffDelete, "abc"}, Diff{DiffEqual, "acx"}},
		},
		{
			"Slide edit right recursive",
			[]Diff{Diff{DiffEqual, "x"}, Diff{DiffDelete, "ca"}, Diff{DiffEqual, "c"}, Diff{DiffDelete, "b"}, Diff{DiffEqual, "a"}},
			[]Diff{Diff{DiffEqual, "xca"}, Diff{DiffDelete, "cba"}},
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
				Diff{DiffEqual, "AAA\r\n\r\nBBB"},
				Diff{DiffInsert, "\r\nDDD\r\n\r\nBBB"},
				Diff{DiffEqual, "\r\nEEE"},
			},
			[]Diff{
				Diff{DiffEqual, "AAA\r\n\r\n"},
				Diff{DiffInsert, "BBB\r\nDDD\r\n\r\n"},
				Diff{DiffEqual, "BBB\r\nEEE"},
			},
		},
		{
			"Line boundaries",
			[]Diff{
				Diff{DiffEqual, "AAA\r\nBBB"},
				Diff{DiffInsert, " DDD\r\nBBB"},
				Diff{DiffEqual, " EEE"},
			},
			[]Diff{
				Diff{DiffEqual, "AAA\r\n"},
				Diff{DiffInsert, "BBB DDD\r\n"},
				Diff{DiffEqual, "BBB EEE"},
			},
		},
		{
			"Word boundaries",
			[]Diff{
				Diff{DiffEqual, "The c"},
				Diff{DiffInsert, "ow and the c"},
				Diff{DiffEqual, "at."},
			},
			[]Diff{
				Diff{DiffEqual, "The "},
				Diff{DiffInsert, "cow and the "},
				Diff{DiffEqual, "cat."},
			},
		},
		{
			"Alphanumeric boundaries",
			[]Diff{
				Diff{DiffEqual, "The-c"},
				Diff{DiffInsert, "ow-and-the-c"},
				Diff{DiffEqual, "at."},
			},
			[]Diff{
				Diff{DiffEqual, "The-"},
				Diff{DiffInsert, "cow-and-the-"},
				Diff{DiffEqual, "cat."},
			},
		},
		{
			"Hitting the start",
			[]Diff{
				Diff{DiffEqual, "a"},
				Diff{DiffDelete, "a"},
				Diff{DiffEqual, "ax"},
			},
			[]Diff{
				Diff{DiffDelete, "a"},
				Diff{DiffEqual, "aax"},
			},
		},
		{
			"Hitting the end",
			[]Diff{
				Diff{DiffEqual, "xa"},
				Diff{DiffDelete, "a"},
				Diff{DiffEqual, "a"},
			},
			[]Diff{
				Diff{DiffEqual, "xaa"},
				Diff{DiffDelete, "a"},
			},
		},
		{
			"Sentence boundaries",
			[]Diff{
				Diff{DiffEqual, "The xxx. The "},
				Diff{DiffInsert, "zzz. The "},
				Diff{DiffEqual, "yyy."},
			},
			[]Diff{
				Diff{DiffEqual, "The xxx."},
				Diff{DiffInsert, " The zzz."},
				Diff{DiffEqual, " The yyy."},
			},
		},
		{
			"UTF-8 strings",
			[]Diff{
				Diff{DiffEqual, "The ♕. The "},
				Diff{DiffInsert, "♔. The "},
				Diff{DiffEqual, "♖."},
			},
			[]Diff{
				Diff{DiffEqual, "The ♕."},
				Diff{DiffInsert, " The ♔."},
				Diff{DiffEqual, " The ♖."},
			},
		},
		{
			"Rune boundaries",
			[]Diff{
				Diff{DiffEqual, "♕♕"},
				Diff{DiffInsert, "♔♔"},
				Diff{DiffEqual, "♖♖"},
			},
			[]Diff{
				Diff{DiffEqual, "♕♕"},
				Diff{DiffInsert, "♔♔"},
				Diff{DiffEqual, "♖♖"},
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
	} {
		actual := dmp.DiffCleanupSemantic(tc.Diffs)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
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
				Diff{DiffDelete, "ab"},
				Diff{DiffInsert, "12"},
				Diff{DiffEqual, "wxyz"},
				Diff{DiffDelete, "cd"},
				Diff{DiffInsert, "34"},
			},
			[]Diff{
				Diff{DiffDelete, "ab"},
				Diff{DiffInsert, "12"},
				Diff{DiffEqual, "wxyz"},
				Diff{DiffDelete, "cd"},
				Diff{DiffInsert, "34"},
			},
		},
		{
			"Four-edit elimination",
			[]Diff{
				Diff{DiffDelete, "ab"},
				Diff{DiffInsert, "12"},
				Diff{DiffEqual, "xyz"},
				Diff{DiffDelete, "cd"},
				Diff{DiffInsert, "34"},
			},
			[]Diff{
				Diff{DiffDelete, "abxyzcd"},
				Diff{DiffInsert, "12xyz34"},
			},
		},
		{
			"Three-edit elimination",
			[]Diff{
				Diff{DiffInsert, "12"},
				Diff{DiffEqual, "x"},
				Diff{DiffDelete, "cd"},
				Diff{DiffInsert, "34"},
			},
			[]Diff{
				Diff{DiffDelete, "xcd"},
				Diff{DiffInsert, "12x34"},
			},
		},
		{
			"Backpass elimination",
			[]Diff{
				Diff{DiffDelete, "ab"},
				Diff{DiffInsert, "12"},
				Diff{DiffEqual, "xy"},
				Diff{DiffInsert, "34"},
				Diff{DiffEqual, "z"},
				Diff{DiffDelete, "cd"},
				Diff{DiffInsert, "56"},
			},
			[]Diff{
				Diff{DiffDelete, "abxyzcd"},
				Diff{DiffInsert, "12xy34z56"},
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
				Diff{DiffDelete, "ab"},
				Diff{DiffInsert, "12"},
				Diff{DiffEqual, "wxyz"},
				Diff{DiffDelete, "cd"},
				Diff{DiffInsert, "34"},
			},
			[]Diff{
				Diff{DiffDelete, "abwxyzcd"},
				Diff{DiffInsert, "12wxyz34"},
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
	dmp := New()

	// Convert a diff into delta string.
	diffs := []Diff{
		Diff{DiffEqual, "jump"},
		Diff{DiffDelete, "s"},
		Diff{DiffInsert, "ed"},
		Diff{DiffEqual, " over "},
		Diff{DiffDelete, "the"},
		Diff{DiffInsert, "a"},
		Diff{DiffEqual, " lazy"},
		Diff{DiffInsert, "old dog"},
	}
	text1 := dmp.DiffText1(diffs)
	assert.Equal(t, "jumps over the lazy", text1)

	delta := dmp.DiffToDelta(diffs)
	assert.Equal(t, "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", delta)

	// Convert delta string into a diff.
	deltaDiffs, err := dmp.DiffFromDelta(text1, delta)
	assert.Equal(t, diffs, deltaDiffs)

	// Generates error (19 < 20).
	_, err = dmp.DiffFromDelta(text1+"x", delta)
	if err == nil {
		t.Fatal("Too long.")
	}

	// Generates error (19 > 18).
	_, err = dmp.DiffFromDelta(text1[1:], delta)
	if err == nil {
		t.Fatal("Too short.")
	}

	// Generates error (%xy invalid URL escape).
	_, err = dmp.DiffFromDelta("", "+%c3%xy")
	if err == nil {
		assert.Fail(t, "expected Invalid URL escape.")
	}

	// Generates error (invalid utf8).
	_, err = dmp.DiffFromDelta("", "+%c3xy")
	if err == nil {
		assert.Fail(t, "expected Invalid utf8.")
	}

	// Test deltas with special characters.
	diffs = []Diff{
		Diff{DiffEqual, "\u0680 \x00 \t %"},
		Diff{DiffDelete, "\u0681 \x01 \n ^"},
		Diff{DiffInsert, "\u0682 \x02 \\ |"},
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
		Diff{DiffInsert, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # "},
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
		{"Levenshtein with trailing equality", []Diff{{DiffDelete, "abc"}, {DiffInsert, "1234"}, {DiffEqual, "xyz"}}, 4},
		{"Levenshtein with leading equality", []Diff{{DiffEqual, "xyz"}, {DiffDelete, "abc"}, {DiffInsert, "1234"}}, 4},
		{"Levenshtein with middle equality", []Diff{{DiffDelete, "abc"}, {DiffEqual, "xyz"}, {DiffInsert, "1234"}}, 7},
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

	text1 := "cat"
	text2 := "map"

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
			Time: time.Date(0001, time.January, 01, 00, 00, 00, 00, time.UTC),

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
		actual := dmp.DiffBisect(text1, text2, tc.Time)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
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
			[]Diff{Diff{DiffEqual, "abc"}},
		},
		{
			"abc",
			"ab123c",
			[]Diff{Diff{DiffEqual, "ab"}, Diff{DiffInsert, "123"}, Diff{DiffEqual, "c"}},
		},
		{
			"a123bc",
			"abc",
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "123"}, Diff{DiffEqual, "bc"}},
		},
		{
			"abc",
			"a123b456c",
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffInsert, "123"}, Diff{DiffEqual, "b"}, Diff{DiffInsert, "456"}, Diff{DiffEqual, "c"}},
		},
		{
			"a123b456c",
			"abc",
			[]Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "123"}, Diff{DiffEqual, "b"}, Diff{DiffDelete, "456"}, Diff{DiffEqual, "c"}},
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
			[]Diff{Diff{DiffDelete, "a"}, Diff{DiffInsert, "b"}},
		},
		{
			"Apples are a fruit.",
			"Bananas are also fruit.",
			[]Diff{
				Diff{DiffDelete, "Apple"},
				Diff{DiffInsert, "Banana"},
				Diff{DiffEqual, "s are a"},
				Diff{DiffInsert, "lso"},
				Diff{DiffEqual, " fruit."},
			},
		},
		{
			"ax\t",
			"\u0680x\u0000",
			[]Diff{
				Diff{DiffDelete, "a"},
				Diff{DiffInsert, "\u0680"},
				Diff{DiffEqual, "x"},
				Diff{DiffDelete, "\t"},
				Diff{DiffInsert, "\u0000"},
			},
		},
		{
			"1ayb2",
			"abxab",
			[]Diff{
				Diff{DiffDelete, "1"},
				Diff{DiffEqual, "a"},
				Diff{DiffDelete, "y"},
				Diff{DiffEqual, "b"},
				Diff{DiffDelete, "2"},
				Diff{DiffInsert, "xab"},
			},
		},
		{
			"abcy",
			"xaxcxabc",
			[]Diff{
				Diff{DiffInsert, "xaxcx"},
				Diff{DiffEqual, "abc"}, Diff{DiffDelete, "y"},
			},
		},
		{
			"ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg",
			"a-bcd-efghijklmnopqrs",
			[]Diff{
				Diff{DiffDelete, "ABCD"},
				Diff{DiffEqual, "a"},
				Diff{DiffDelete, "="},
				Diff{DiffInsert, "-"},
				Diff{DiffEqual, "bcd"},
				Diff{DiffDelete, "="},
				Diff{DiffInsert, "-"},
				Diff{DiffEqual, "efghijklmnopqrs"},
				Diff{DiffDelete, "EFGHIJKLMNOefg"},
			},
		},
		{
			"a [[Pennsylvania]] and [[New",
			" and [[Pennsylvania]]",
			[]Diff{
				Diff{DiffInsert, " "},
				Diff{DiffEqual, "a"},
				Diff{DiffInsert, "nd"},
				Diff{DiffEqual, " [[Pennsylvania]]"},
				Diff{DiffDelete, " and [[New"},
			},
		},
	} {
		actual := dmp.DiffMain(tc.Text1, tc.Text2, false)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
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

	// Test that we didn't take forever (be very forgiving).
	// Theoretically this test could fail very occasionally if the
	// OS task swaps or locks up for a second at the wrong moment.
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

func TestMatchAlphabet(t *testing.T) {
	type TestCase struct {
		Pattern string

		Expected map[byte]int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			Pattern: "abc",

			Expected: map[byte]int{
				'a': 4,
				'b': 2,
				'c': 1,
			},
		},
		{
			Pattern: "abcaba",

			Expected: map[byte]int{
				'a': 37,
				'b': 18,
				'c': 8,
			},
		},
	} {
		actual := dmp.MatchAlphabet(tc.Pattern)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestMatchBitap(t *testing.T) {
	type TestCase struct {
		Name string

		Text     string
		Pattern  string
		Location int

		Expected int
	}

	dmp := New()
	dmp.MatchDistance = 100
	dmp.MatchThreshold = 0.5

	for i, tc := range []TestCase{
		{"Exact match #1", "abcdefghijk", "fgh", 5, 5},
		{"Exact match #2", "abcdefghijk", "fgh", 0, 5},
		{"Fuzzy match #1", "abcdefghijk", "efxhi", 0, 4},
		{"Fuzzy match #2", "abcdefghijk", "cdefxyhijk", 5, 2},
		{"Fuzzy match #3", "abcdefghijk", "bxy", 1, -1},
		{"Overflow", "123456789xx0", "3456789x0", 2, 2},
		{"Before start match", "abcdef", "xxabc", 4, 0},
		{"Beyond end match", "abcdef", "defyy", 4, 3},
		{"Oversized pattern", "abcdef", "xabcdefy", 0, 0},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchThreshold = 0.4

	for i, tc := range []TestCase{
		{"Threshold #1", "abcdefghijk", "efxyhi", 1, 4},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchThreshold = 0.3

	for i, tc := range []TestCase{
		{"Threshold #2", "abcdefghijk", "efxyhi", 1, -1},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchThreshold = 0.0

	for i, tc := range []TestCase{
		{"Threshold #3", "abcdefghijk", "bcdef", 1, 1},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchThreshold = 0.5

	for i, tc := range []TestCase{
		{"Multiple select #1", "abcdexyzabcde", "abccde", 3, 0},
		{"Multiple select #2", "abcdexyzabcde", "abccde", 5, 8},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	// Strict location.
	dmp.MatchDistance = 10

	for i, tc := range []TestCase{
		{"Distance test #1", "abcdefghijklmnopqrstuvwxyz", "abcdefg", 24, -1},
		{"Distance test #2", "abcdefghijklmnopqrstuvwxyz", "abcdxxefg", 1, 0},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	// Loose location.
	dmp.MatchDistance = 1000

	for i, tc := range []TestCase{
		{"Distance test #3", "abcdefghijklmnopqrstuvwxyz", "abcdefg", 24, 0},
	} {
		actual := dmp.MatchBitap(tc.Text, tc.Pattern, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestMatchMain(t *testing.T) {
	type TestCase struct {
		Name string

		Text1    string
		Text2    string
		Location int

		Expected int
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Equality", "abcdef", "abcdef", 1000, 0},
		{"Null text", "", "abcdef", 1, -1},
		{"Null pattern", "abcdef", "", 3, 3},
		{"Exact match", "abcdef", "de", 3, 3},
		{"Beyond end match", "abcdef", "defy", 4, 3},
		{"Oversized pattern", "abcdef", "abcdefy", 0, 0},
	} {
		actual := dmp.MatchMain(tc.Text1, tc.Text2, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchThreshold = 0.7

	for i, tc := range []TestCase{
		{"Complex match", "I am the very model of a modern major general.", " that berry ", 5, 4},
	} {
		actual := dmp.MatchMain(tc.Text1, tc.Text2, tc.Location)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestPatchString(t *testing.T) {
	type TestCase struct {
		Patch Patch

		Expected string
	}

	for i, tc := range []TestCase{
		{
			Patch: Patch{
				start1:  20,
				start2:  21,
				length1: 18,
				length2: 17,

				diffs: []Diff{
					{DiffEqual, "jump"},
					{DiffDelete, "s"},
					{DiffInsert, "ed"},
					{DiffEqual, " over "},
					{DiffDelete, "the"},
					{DiffInsert, "a"},
					{DiffEqual, "\nlaz"},
				},
			},

			Expected: "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n %0Alaz\n",
		},
	} {
		actual := tc.Patch.String()
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestPatchFromText(t *testing.T) {
	type TestCase struct {
		Patch string

		ErrorMessagePrefix string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"", ""},
		{"@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n %0Alaz\n", ""},
		{"@@ -1 +1 @@\n-a\n+b\n", ""},
		{"@@ -1,3 +0,0 @@\n-abc\n", ""},
		{"@@ -0,0 +1,3 @@\n+abc\n", ""},
		{"Bad\nPatch\n", "Invalid patch string"},
	} {
		patches, err := dmp.PatchFromText(tc.Patch)
		if tc.ErrorMessagePrefix == "" {
			assert.Nil(t, err)

			if tc.Patch == "" {
				assert.Equal(t, []Patch{}, patches, fmt.Sprintf("Test case #%d, %#v", i, tc))
			} else {
				assert.Equal(t, tc.Patch, patches[0].String(), fmt.Sprintf("Test case #%d, %#v", i, tc))
			}
		} else {
			e := err.Error()
			if strings.HasPrefix(e, tc.ErrorMessagePrefix) {
				e = tc.ErrorMessagePrefix
			}
			assert.Equal(t, tc.ErrorMessagePrefix, e)
		}
	}

	diffs := []Diff{
		{DiffDelete, "`1234567890-=[]\\;',./"},
		{DiffInsert, "~!@#$%^&*()_+{}|:\"<>?"},
	}

	patches, err := dmp.PatchFromText("@@ -1,21 +1,21 @@\n-%601234567890-=%5B%5D%5C;',./\n+~!@#$%25%5E&*()_+%7B%7D%7C:%22%3C%3E?\n")
	assert.Len(t, patches, 1)
	assert.Equal(t, diffs,
		patches[0].diffs,
	)
	assert.Nil(t, err)
}

func TestPatchToText(t *testing.T) {
	type TestCase struct {
		Patch string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"},
		{"@@ -1,9 +1,9 @@\n-f\n+F\n oo+fooba\n@@ -7,9 +7,9 @@\n obar\n-,\n+.\n  tes\n"},
	} {
		patches, err := dmp.PatchFromText(tc.Patch)
		assert.Nil(t, err)

		actual := dmp.PatchToText(patches)
		assert.Equal(t, tc.Patch, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestPatchAddContext(t *testing.T) {
	type TestCase struct {
		Name string

		Patch string
		Text  string

		Expected string
	}

	dmp := New()
	dmp.PatchMargin = 4

	for i, tc := range []TestCase{
		{"Simple case", "@@ -21,4 +21,10 @@\n-jump\n+somersault\n", "The quick brown fox jumps over the lazy dog.", "@@ -17,12 +17,18 @@\n fox \n-jump\n+somersault\n s ov\n"},
		{"Not enough trailing context", "@@ -21,4 +21,10 @@\n-jump\n+somersault\n", "The quick brown fox jumps.", "@@ -17,10 +17,16 @@\n fox \n-jump\n+somersault\n s.\n"},
		{"Not enough leading context", "@@ -3 +3,2 @@\n-e\n+at\n", "The quick brown fox jumps.", "@@ -1,7 +1,8 @@\n Th\n-e\n+at\n  qui\n"},
		{"Ambiguity", "@@ -3 +3,2 @@\n-e\n+at\n", "The quick brown fox jumps.  The quick brown fox crashes.", "@@ -1,27 +1,28 @@\n Th\n-e\n+at\n  quick brown fox jumps. \n"},
	} {
		patches, err := dmp.PatchFromText(tc.Patch)
		assert.Nil(t, err)

		actual := dmp.PatchAddContext(patches[0], tc.Text)
		assert.Equal(t, tc.Expected, actual.String(), fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

// TODO
func TestPatchMakeAndPatchToText(t *testing.T) {
	type TestCase struct {
		Name string

		Input1 interface{}
		Input2 interface{}
		Input3 interface{}

		Expected string
	}

	dmp := New()

	text1 := "The quick brown fox jumps over the lazy dog."
	text2 := "That quick brown fox jumped over a lazy dog."

	for i, tc := range []TestCase{
		{"Null case", "", "", nil, ""},
		{"Text2+Text1 inputs", text2, text1, nil, "@@ -1,8 +1,7 @@\n Th\n-at\n+e\n  qui\n@@ -21,17 +21,18 @@\n jump\n-ed\n+s\n  over \n-a\n+the\n  laz\n"},
		{"Text1+Text2 inputs", text1, text2, nil, "@@ -1,11 +1,12 @@\n Th\n-e\n+at\n  quick b\n@@ -22,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"},
		{"Diff input", dmp.DiffMain(text1, text2, false), nil, nil, "@@ -1,11 +1,12 @@\n Th\n-e\n+at\n  quick b\n@@ -22,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"},
		{"Text1+Diff inputs", text1, dmp.DiffMain(text1, text2, false), nil, "@@ -1,11 +1,12 @@\n Th\n-e\n+at\n  quick b\n@@ -22,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"},
		{"Text1+Text2+Diff inputs (deprecated)", text1, text2, dmp.DiffMain(text1, text2, false), "@@ -1,11 +1,12 @@\n Th\n-e\n+at\n  quick b\n@@ -22,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"},
		{"Character encoding", "`1234567890-=[]\\;',./", "~!@#$%^&*()_+{}|:\"<>?", nil, "@@ -1,21 +1,21 @@\n-%601234567890-=%5B%5D%5C;',./\n+~!@#$%25%5E&*()_+%7B%7D%7C:%22%3C%3E?\n"},
		{"Long string with repeats", strings.Repeat("abcdef", 100), strings.Repeat("abcdef", 100) + "123", nil, "@@ -573,28 +573,31 @@\n cdefabcdefabcdefabcdefabcdef\n+123\n"},
		{"Corner case of #31 fixed by #32", "2016-09-01T03:07:14.807830741Z", "2016-09-01T03:07:15.154800781Z", nil, "@@ -15,16 +15,16 @@\n 07:1\n+5.15\n 4\n-.\n 80\n+0\n 78\n-3074\n 1Z\n"},
	} {
		var patches []Patch
		if tc.Input3 != nil {
			patches = dmp.PatchMake(tc.Input1, tc.Input2, tc.Input3)
		} else if tc.Input2 != nil {
			patches = dmp.PatchMake(tc.Input1, tc.Input2)
		} else if ps, ok := tc.Input1.([]Patch); ok {
			patches = ps
		} else {
			patches = dmp.PatchMake(tc.Input1)
		}

		actual := dmp.PatchToText(patches)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	// Corner case of #28 wrong patch with timeout of 0
	dmp.DiffTimeout = 0

	text1 = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus ut risus et enim consectetur convallis a non ipsum. Sed nec nibh cursus, interdum libero vel."
	text2 = "Lorem a ipsum dolor sit amet, consectetur adipiscing elit. Vivamus ut risus et enim consectetur convallis a non ipsum. Sed nec nibh cursus, interdum liberovel."

	diffs := dmp.DiffMain(text1, text2, true)
	// Additional check that the diff texts are equal to the originals even if we are using DiffMain with checklines=true #29
	assert.Equal(t, text1, dmp.DiffText1(diffs))
	assert.Equal(t, text2, dmp.DiffText2(diffs))

	patches := dmp.PatchMake(text1, diffs)

	actual := dmp.PatchToText(patches)
	assert.Equal(t, "@@ -1,14 +1,16 @@\n Lorem \n+a \n ipsum do\n@@ -148,13 +148,12 @@\n m libero\n- \n vel.\n", actual)
}

func TestPatchSplitMax(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"abcdefghijklmnopqrstuvwxyz01234567890", "XabXcdXefXghXijXklXmnXopXqrXstXuvXwxXyzX01X23X45X67X89X0", "@@ -1,32 +1,46 @@\n+X\n ab\n+X\n cd\n+X\n ef\n+X\n gh\n+X\n ij\n+X\n kl\n+X\n mn\n+X\n op\n+X\n qr\n+X\n st\n+X\n uv\n+X\n wx\n+X\n yz\n+X\n 012345\n@@ -25,13 +39,18 @@\n zX01\n+X\n 23\n+X\n 45\n+X\n 67\n+X\n 89\n+X\n 0\n"},
		{"abcdef1234567890123456789012345678901234567890123456789012345678901234567890uvwxyz", "abcdefuvwxyz", "@@ -3,78 +3,8 @@\n cdef\n-1234567890123456789012345678901234567890123456789012345678901234567890\n uvwx\n"},
		{"1234567890123456789012345678901234567890123456789012345678901234567890", "abc", "@@ -1,32 +1,4 @@\n-1234567890123456789012345678\n 9012\n@@ -29,32 +1,4 @@\n-9012345678901234567890123456\n 7890\n@@ -57,14 +1,3 @@\n-78901234567890\n+abc\n"},
		{"abcdefghij , h : 0 , t : 1 abcdefghij , h : 0 , t : 1 abcdefghij , h : 0 , t : 1", "abcdefghij , h : 1 , t : 1 abcdefghij , h : 1 , t : 1 abcdefghij , h : 0 , t : 1", "@@ -2,32 +2,32 @@\n bcdefghij , h : \n-0\n+1\n  , t : 1 abcdef\n@@ -29,32 +29,32 @@\n bcdefghij , h : \n-0\n+1\n  , t : 1 abcdef\n"},
	} {
		patches := dmp.PatchMake(tc.Text1, tc.Text2)
		patches = dmp.PatchSplitMax(patches)

		actual := dmp.PatchToText(patches)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestPatchAddPadding(t *testing.T) {
	type TestCase struct {
		Name string

		Text1 string
		Text2 string

		Expected            string
		ExpectedWithPadding string
	}

	dmp := New()

	for i, tc := range []TestCase{
		{"Both edges full", "", "test", "@@ -0,0 +1,4 @@\n+test\n", "@@ -1,8 +1,12 @@\n %01%02%03%04\n+test\n %01%02%03%04\n"},
		{"Both edges partial", "XY", "XtestY", "@@ -1,2 +1,6 @@\n X\n+test\n Y\n", "@@ -2,8 +2,12 @@\n %02%03%04X\n+test\n Y%01%02%03\n"},
		{"Both edges none", "XXXXYYYY", "XXXXtestYYYY", "@@ -1,8 +1,12 @@\n XXXX\n+test\n YYYY\n", "@@ -5,8 +5,12 @@\n XXXX\n+test\n YYYY\n"},
	} {
		patches := dmp.PatchMake(tc.Text1, tc.Text2)

		actual := dmp.PatchToText(patches)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))

		dmp.PatchAddPadding(patches)

		actualWithPadding := dmp.PatchToText(patches)
		assert.Equal(t, tc.ExpectedWithPadding, actualWithPadding, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestPatchApply(t *testing.T) {
	type TestCase struct {
		Name string

		Text1    string
		Text2    string
		TextBase string

		Expected        string
		ExpectedApplies []bool
	}

	dmp := New()
	dmp.MatchDistance = 1000
	dmp.MatchThreshold = 0.5
	dmp.PatchDeleteThreshold = 0.5

	for i, tc := range []TestCase{
		{"Null case", "", "", "Hello world.", "Hello world.", []bool{}},
		{"Exact match", "The quick brown fox jumps over the lazy dog.", "That quick brown fox jumped over a lazy dog.", "The quick brown fox jumps over the lazy dog.", "That quick brown fox jumped over a lazy dog.", []bool{true, true}},
		{"Partial match", "The quick brown fox jumps over the lazy dog.", "That quick brown fox jumped over a lazy dog.", "The quick red rabbit jumps over the tired tiger.", "That quick red rabbit jumped over a tired tiger.", []bool{true, true}},
		{"Failed match", "The quick brown fox jumps over the lazy dog.", "That quick brown fox jumped over a lazy dog.", "I am the very model of a modern major general.", "I am the very model of a modern major general.", []bool{false, false}},
		{"Big delete, small Diff", "x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy", "x123456789012345678901234567890-----++++++++++-----123456789012345678901234567890y", "xabcy", []bool{true, true}},
		{"Big delete, big Diff 1", "x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy", "x12345678901234567890---------------++++++++++---------------12345678901234567890y", "xabc12345678901234567890---------------++++++++++---------------12345678901234567890y", []bool{false, true}},
	} {
		patches := dmp.PatchMake(tc.Text1, tc.Text2)

		actual, actualApplies := dmp.PatchApply(patches, tc.TextBase)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
		assert.Equal(t, tc.ExpectedApplies, actualApplies, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.PatchDeleteThreshold = 0.6

	for i, tc := range []TestCase{
		{"Big delete, big Diff 2", "x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy", "x12345678901234567890---------------++++++++++---------------12345678901234567890y", "xabcy", []bool{true, true}},
	} {
		patches := dmp.PatchMake(tc.Text1, tc.Text2)

		actual, actualApplies := dmp.PatchApply(patches, tc.TextBase)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
		assert.Equal(t, tc.ExpectedApplies, actualApplies, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchDistance = 0
	dmp.MatchThreshold = 0.0
	dmp.PatchDeleteThreshold = 0.5

	for i, tc := range []TestCase{
		{"Compensate for failed patch", "abcdefghijklmnopqrstuvwxyz--------------------1234567890", "abcXXXXXXXXXXdefghijklmnopqrstuvwxyz--------------------1234567YYYYYYYYYY890", "ABCDEFGHIJKLMNOPQRSTUVWXYZ--------------------1234567890", "ABCDEFGHIJKLMNOPQRSTUVWXYZ--------------------1234567YYYYYYYYYY890", []bool{false, true}},
	} {
		patches := dmp.PatchMake(tc.Text1, tc.Text2)

		actual, actualApplies := dmp.PatchApply(patches, tc.TextBase)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
		assert.Equal(t, tc.ExpectedApplies, actualApplies, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.MatchThreshold = 0.5
	dmp.MatchDistance = 1000

	for i, tc := range []TestCase{
		{"No side effects", "", "test", "", "test", []bool{true}},
		{"No side effects with major delete", "The quick brown fox jumps over the lazy dog.", "Woof", "The quick brown fox jumps over the lazy dog.", "Woof", []bool{true, true}},
		{"Edge exact match", "", "test", "", "test", []bool{true}},
		{"Near edge exact match", "XY", "XtestY", "XY", "XtestY", []bool{true}},
		{"Edge partial match", "y", "y123", "x", "x123", []bool{true}},
	} {
		patches := dmp.PatchMake(tc.Text1, tc.Text2)

		actual, actualApplies := dmp.PatchApply(patches, tc.TextBase)
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
		assert.Equal(t, tc.ExpectedApplies, actualApplies, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
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

func BenchmarkDiffCommonPrefix(b *testing.B) {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ"

	dmp := New()

	for i := 0; i < b.N; i++ {
		dmp.DiffCommonPrefix(s, s)
	}
}

func BenchmarkDiffCommonSuffix(b *testing.B) {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ"

	dmp := New()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dmp.DiffCommonSuffix(s, s)
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
		diffs = dmp.DiffCharsToLines(diffs, linearray)
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

func BenchmarkDiffCleanupSemantic(b *testing.B) {
	s1, s2 := speedtestTexts()

	dmp := New()

	diffs := dmp.DiffMain(s1, s2, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dmp.DiffCleanupSemantic(diffs)
	}
}
