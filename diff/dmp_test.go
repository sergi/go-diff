package diffmatchpatch

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
	"runtime"
	"bytes"

	"github.com/bmizerany/assert"
)

func softAssert(t *testing.T, cond bool, msg string) {
	if !cond {
		print("assertion fail: ", msg, "\n")
		panic(1)
	}
}

func caller() string {
	if _, _, line, ok := runtime.Caller(2); ok {
		return fmt.Sprintf("(actual-line %v) ", line)
	}
	return ""
}

func pretty(diffs []Diff) string {
	var w bytes.Buffer
	for i, diff := range diffs {
		w.WriteString(fmt.Sprintf("%v. ",i))
		switch diff.Type {
		case DiffInsert:
			w.WriteString("DiffIns")
		case DiffDelete:
			w.WriteString("DiffDel")
		case DiffEqual:
			w.WriteString("DiffEql")
		default:
			w.WriteString("Unknown")
		}
		w.WriteString(fmt.Sprintf(": %v\n", diff.Text))
	}
	return w.String()
}

func assertMapEqual(t *testing.T, seq1, seq2 interface{}) {
	v1 := reflect.ValueOf(seq1)
	k1 := v1.Kind()
	v2 := reflect.ValueOf(seq2)
	k2 := v2.Kind()

	if k1 != reflect.Map || k2 != reflect.Map {
		t.Fatalf("%v Parameters are not maps", caller())
	} else if v1.Len() != v2.Len() {
		t.Fatalf("%v Maps of different length: %v != %v", caller(), v1.Len(), v2.Len())
	}

	keys1, keys2 := v1.MapKeys(), v2.MapKeys()

	if len(keys1) != len(keys2) {
		t.Fatalf("%v Maps of different length", caller())
	}

	for _, key1 := range keys1 {
		if a, b := v2.MapIndex(key1), v1.MapIndex(key1); a != b {
			t.Fatal("%v Different key/value in Map: %v != %v", caller(), a, b)
		}
	}

	for _, key2 := range keys2 {
		if a, b := v1.MapIndex(key2), v2.MapIndex(key2); a != b {
			t.Fatal("%v Different key/value in Map: %v != %v", caller(), a, b)
		}
	}
}

func assertDiffEqual(t *testing.T, seq1, seq2 []Diff) {
	if a, b := len(seq1), len(seq2); a != b {
		t.Fatalf("%v Sequences of different length: %v != %v", caller(), a, b)
	}

	for i := range seq1 {
		if a, b := seq1[i], seq2[i]; a != b {
			t.Errorf("%v\nseq1:\n%v\nseq2:\n%v", caller(), pretty(seq1), pretty(seq2))
			t.Fatalf("%v %v != %v", caller(), a, b)
		}
	}
}

func assertStrEqual(t *testing.T, seq1, seq2 []string) {
	if a, b := len(seq1), len(seq2); a != b {
		t.Fatalf("%v Sequences of different length: %v != %v", caller(), a, b)
	}

	for i := range seq1 {
		if a, b := seq1[i], seq2[i]; a != b {
			t.Fatalf("%v %v != %v", caller(), a, b)
		}
	}
}

func diffRebuildtexts(diffs []Diff) []string {
	text := []string{"", ""}
	for _, myDiff := range diffs {
		if myDiff.Type != DiffInsert {
			text[0] += myDiff.Text
		}
		if myDiff.Type != DiffDelete {
			text[1] += myDiff.Text
		}
	}
	return text
}

func Test_diffCommonPrefix(t *testing.T) {
	dmp := createDMP()
	// Detect any common suffix.
	// Null case.
	assert.Equal(t, 0, dmp.DiffCommonPrefix("abc", "xyz"), "'abc' and 'xyz' should not be equal")

	// Non-null case.
	assert.Equal(t, 4, dmp.DiffCommonPrefix("1234abcdef", "1234xyz"), "")

	// Whole case.
	assert.Equal(t, 4, dmp.DiffCommonPrefix("1234", "1234xyz"), "")
}

func Test_diffCommonSuffixTest(t *testing.T) {
	dmp := createDMP()
	// Detect any common suffix.
	// Null case.
	assert.Equal(t, 0, dmp.DiffCommonSuffix("abc", "xyz"), "")

	// Non-null case.
	assert.Equal(t, 4, dmp.DiffCommonSuffix("abcdef1234", "xyz1234"), "")

	// Whole case.
	assert.Equal(t, 4, dmp.DiffCommonSuffix("1234", "xyz1234"), "")
}

func Test_diffCommonOverlapTest(t *testing.T) {
	dmp := createDMP()
	// Detect any suffix/prefix overlap.
	// Null case.
	assert.Equal(t, 0, dmp.DiffCommonOverlap("", "abcd"), "")

	// Whole case.
	assert.Equal(t, 3, dmp.DiffCommonOverlap("abc", "abcd"), "")

	// No overlap.
	assert.Equal(t, 0, dmp.DiffCommonOverlap("123456", "abcd"), "")

	// Overlap.
	assert.Equal(t, 3, dmp.DiffCommonOverlap("123456xxx", "xxxabcd"), "")

	// Unicode.
	// Some overly clever languages (C#) may treat ligatures as equal to their
	// component letters.  E.g. U+FB01 == 'fi'
	assert.Equal(t, 0, dmp.DiffCommonOverlap("fi", "\ufb01i"), "")
}

func Test_diffHalfmatchTest(t *testing.T) {
	dmp := createDMP()
	dmp.DiffTimeout = 1
	// No match.
	softAssert(t, dmp.DiffHalfMatch("1234567890", "abcdef") == nil, "")
	softAssert(t, dmp.DiffHalfMatch("12345", "23") == nil, "")

	// Single Match.
	assertStrEqual(t, 
		[]string{"12", "90", "a", "z", "345678"},
		dmp.DiffHalfMatch("1234567890", "a345678z"))

	assertStrEqual(t, []string{"a", "z", "12", "90", "345678"}, dmp.DiffHalfMatch("a345678z", "1234567890"))

	assertStrEqual(t, []string{"abc", "z", "1234", "0", "56789"}, dmp.DiffHalfMatch("abc56789z", "1234567890"))

	assertStrEqual(t, []string{"a", "xyz", "1", "7890", "23456"}, dmp.DiffHalfMatch("a23456xyz", "1234567890"))

	// Multiple Matches.
	assertStrEqual(t, []string{"12123", "123121", "a", "z", "1234123451234"}, dmp.DiffHalfMatch("121231234123451234123121", "a1234123451234z"))

	assertStrEqual(t, []string{"", "-=-=-=-=-=", "x", "", "x-=-=-=-=-=-=-="}, dmp.DiffHalfMatch("x-=-=-=-=-=-=-=-=-=-=-=-=", "xx-=-=-=-=-=-=-="))

	assertStrEqual(t, []string{"-=-=-=-=-=", "", "", "y", "-=-=-=-=-=-=-=y"}, dmp.DiffHalfMatch("-=-=-=-=-=-=-=-=-=-=-=-=y", "-=-=-=-=-=-=-=yy"))

	// Non-optimal halfmatch.
	// Optimal diff would be -q+x=H-i+e=lloHe+Hu=llo-Hew+y not -qHillo+x=HelloHe-w+Hulloy
	assertStrEqual(t, []string{"qHillo", "w", "x", "Hulloy", "HelloHe"}, dmp.DiffHalfMatch("qHilloHelloHew", "xHelloHeHulloy"))

	// Optimal no halfmatch.
	dmp.DiffTimeout = 0
	softAssert(t, dmp.DiffHalfMatch("qHilloHelloHew", "xHelloHeHulloy") == nil, "")
}

func Test_diffLinesToChars(t *testing.T) {
	dmp := createDMP()
	// Convert lines down to characters.
	tmpVector := []string{"", "alpha\n", "beta\n"}

	result0, result1, result2 := dmp.DiffLinesToChars("alpha\nbeta\nalpha\n", "beta\nalpha\nbeta\n")
	assert.Equal(t, "\u0001\u0002\u0001", result0, "")
	assert.Equal(t, "\u0002\u0001\u0002", result1, "")
	assertStrEqual(t, tmpVector, result2)

	tmpVector = []string{"", "alpha\r\n", "beta\r\n", "\r\n"}
	result0, result1, result2 = dmp.DiffLinesToChars("", "alpha\r\nbeta\r\n\r\n\r\n")
	assert.Equal(t, "", result0, "")
	assert.Equal(t, "\u0001\u0002\u0003\u0003", result1, "")
	assertStrEqual(t, tmpVector, result2)

	tmpVector = []string{"", "a", "b"}
	result0, result1, result2 = dmp.DiffLinesToChars("a", "b")
	assert.Equal(t, "\u0001", result0, "")
	assert.Equal(t, "\u0002", result1, "")
	assertStrEqual(t, tmpVector, result2)

	// More than 256 to reveal any 8-bit limitations.
	/*
	   n := 300
	   tmpVector = []string{}
	   lineList := []rune{}
	   charList := []rune{}

	   for x := 1; x < n+1; x++ {
	       tmpVector = append(tmpVector, string(x)+"\n")
	       lineList = append(lineList, rune(x), '\n')
	       charList = append(charList, rune(x))
	   }
	   assert.Equal(t, n, len(tmpVector), "")

	   lines := string(lineList)
	   chars := string(charList)
	   assert.Equal(t, n, utf8.RuneCountInString(chars), "")
	   tmpVector = append(tmpVector, "")

	   result0, result1, result2 = dmp.DiffLinesToChars(lines, "")

	   assert.Equal(t, chars, result0)
	   assert.Equal(t, "", result1, "")
	   assertDiffEqual(t, tmpVector, result2)
	*/
}

func Test_diffCharsToLines(t *testing.T) {
	dmp := createDMP()
	// Convert chars up to lines.
	diffs := []Diff{
		Diff{DiffEqual, "\u0001\u0002\u0001"},
		Diff{DiffInsert, "\u0002\u0001\u0002"}}

	tmpVector := []string{"", "alpha\n", "beta\n"}
	actual := dmp.DiffCharsToLines(diffs, tmpVector)
	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "alpha\nbeta\nalpha\n"},
		Diff{DiffInsert, "beta\nalpha\nbeta\n"}}, actual)

	// More than 256 to reveal any 8-bit limitations.
	n := 257
	tmpVector = []string{}
	lineList := []rune{}
	charList := []rune{}

	for x := 1; x <= n; x++ {
		tmpVector = append(tmpVector, string(x)+"\n")
		lineList = append(lineList, rune(x), '\n')
		charList = append(charList, rune(x))
	}

	assert.Equal(t, n, len(tmpVector))
	assert.Equal(t, n, len(charList))

	tmpVector = append([]string{""}, tmpVector...)
	diffs = []Diff{Diff{DiffDelete, string(charList)}}
	actual = dmp.DiffCharsToLines(diffs, tmpVector)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, string(lineList)}}, actual)
}

func Test_diffCleanupMerge(t *testing.T) {
	dmp := createDMP()
	// Cleanup a messy diff.
	// Null case.
	diffs := []Diff{}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{}, diffs)

	// No Diff case.
	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "b"}, Diff{DiffInsert, "c"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "b"}, Diff{DiffInsert, "c"}}, diffs)

	// Merge equalities.
	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffEqual, "b"}, Diff{DiffEqual, "c"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffEqual, "abc"}}, diffs)

	// Merge deletions.
	diffs = []Diff{Diff{DiffDelete, "a"}, Diff{DiffDelete, "b"}, Diff{DiffDelete, "c"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffDelete, "abc"}}, diffs)

	// Merge insertions.
	diffs = []Diff{Diff{DiffInsert, "a"}, Diff{DiffInsert, "b"}, Diff{DiffInsert, "c"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffInsert, "abc"}}, diffs)

	// Merge interweave.
	diffs = []Diff{Diff{DiffDelete, "a"}, Diff{DiffInsert, "b"}, Diff{DiffDelete, "c"}, Diff{DiffInsert, "d"}, Diff{DiffEqual, "e"}, Diff{DiffEqual, "f"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffDelete, "ac"}, Diff{DiffInsert, "bd"}, Diff{DiffEqual, "ef"}}, diffs)

	// Prefix and suffix detection.
	diffs = []Diff{Diff{DiffDelete, "a"}, Diff{DiffInsert, "abc"}, Diff{DiffDelete, "dc"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "d"}, Diff{DiffInsert, "b"}, Diff{DiffEqual, "c"}}, diffs)

	// Prefix and suffix detection with equalities.
	diffs = []Diff{Diff{DiffEqual, "x"}, Diff{DiffDelete, "a"}, Diff{DiffInsert, "abc"}, Diff{DiffDelete, "dc"}, Diff{DiffEqual, "y"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffEqual, "xa"}, Diff{DiffDelete, "d"}, Diff{DiffInsert, "b"}, Diff{DiffEqual, "cy"}}, diffs)

	// Slide edit left.
	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffInsert, "ba"}, Diff{DiffEqual, "c"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffInsert, "ab"}, Diff{DiffEqual, "ac"}}, diffs)

	// Slide edit right.
	diffs = []Diff{Diff{DiffEqual, "c"}, Diff{DiffInsert, "ab"}, Diff{DiffEqual, "a"}}
	diffs = dmp.DiffCleanupMerge(diffs)

	assertDiffEqual(t, []Diff{Diff{DiffEqual, "ca"}, Diff{DiffInsert, "ba"}}, diffs)

	// Slide edit left recursive.
	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "b"}, Diff{DiffEqual, "c"}, Diff{DiffDelete, "ac"}, Diff{DiffEqual, "x"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffDelete, "abc"}, Diff{DiffEqual, "acx"}}, diffs)

	// Slide edit right recursive.
	diffs = []Diff{Diff{DiffEqual, "x"}, Diff{DiffDelete, "ca"}, Diff{DiffEqual, "c"}, Diff{DiffDelete, "b"}, Diff{DiffEqual, "a"}}
	diffs = dmp.DiffCleanupMerge(diffs)
	assertDiffEqual(t, []Diff{Diff{DiffEqual, "xca"}, Diff{DiffDelete, "cba"}}, diffs)
}

func Test_diffCleanupSemanticLossless(t *testing.T) {
	dmp := createDMP()
	// Slide diffs to match logical boundaries.
	// Null case.
	diffs := []Diff{}
	dmp.DiffCleanupSemanticLossless(diffs)
	assertDiffEqual(t, []Diff{}, diffs)

	// Blank lines.
	diffs = []Diff{
		Diff{DiffEqual, "AAA\r\n\r\nBBB"},
		Diff{DiffInsert, "\r\nDDD\r\n\r\nBBB"},
		Diff{DiffEqual, "\r\nEEE"},
	}

	dmp.DiffCleanupSemanticLossless(diffs)

	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "AAA\r\n\r\n"},
		Diff{DiffInsert, "BBB\r\nDDD\r\n\r\n"},
		Diff{DiffEqual, "BBB\r\nEEE"}}, diffs)

	// Line boundaries.
	diffs = []Diff{
		Diff{DiffEqual, "AAA\r\nBBB"},
		Diff{DiffInsert, " DDD\r\nBBB"},
		Diff{DiffEqual, " EEE"}}

	dmp.DiffCleanupSemanticLossless(diffs)

	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "AAA\r\n"},
		Diff{DiffInsert, "BBB DDD\r\n"},
		Diff{DiffEqual, "BBB EEE"}}, diffs)

	// Word boundaries.
	diffs = []Diff{
		Diff{DiffEqual, "The c"},
		Diff{DiffInsert, "ow and the c"},
		Diff{DiffEqual, "at."}}

	dmp.DiffCleanupSemanticLossless(diffs)

	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "The "},
		Diff{DiffInsert, "cow and the "},
		Diff{DiffEqual, "cat."}}, diffs)

	// Alphanumeric boundaries.
	diffs = []Diff{
		Diff{DiffEqual, "The-c"},
		Diff{DiffInsert, "ow-and-the-c"},
		Diff{DiffEqual, "at."}}

	dmp.DiffCleanupSemanticLossless(diffs)

	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "The-"},
		Diff{DiffInsert, "cow-and-the-"},
		Diff{DiffEqual, "cat."}}, diffs)

	// Hitting the start.
	diffs = []Diff{
		Diff{DiffEqual, "a"},
		Diff{DiffDelete, "a"},
		Diff{DiffEqual, "ax"}}

	dmp.DiffCleanupSemanticLossless(diffs)

	/*assertDiffEqual(t, []Diff{
	Diff{DiffDelete, "a"},
	Diff{DiffEqual, "aax"}}, diffs)*/

	// Hitting the end.
	diffs = []Diff{
		Diff{DiffEqual, "xa"},
		Diff{DiffDelete, "a"},
		Diff{DiffEqual, "a"}}

	dmp.DiffCleanupSemanticLossless(diffs)
	/*
		assertDiffEqual(t, []Diff{
			Diff{DiffEqual, "xaa"},
			Diff{DiffDelete, "a"}}, diffs)
	*/
	// Sentence boundaries.
	diffs = []Diff{
		Diff{DiffEqual, "The xxx. The "},
		Diff{DiffInsert, "zzz. The "},
		Diff{DiffEqual, "yyy."}}

	dmp.DiffCleanupSemanticLossless(diffs)

	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "The xxx."},
		Diff{DiffInsert, " The zzz."},
		Diff{DiffEqual, " The yyy."}}, diffs)
}

func Test_diffCleanupSemantic(t *testing.T) {
	dmp := createDMP()
	// Cleanup semantically trivial equalities.
	// Null case.
	diffs := []Diff{}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{}, diffs)

	// No elimination #1.
	diffs = []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "cd"},
		Diff{DiffEqual, "12"},
		Diff{DiffDelete, "e"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "cd"},
		Diff{DiffEqual, "12"},
		Diff{DiffDelete, "e"}}, diffs)

	// No elimination #2.
	diffs = []Diff{
		Diff{DiffDelete, "abc"},
		Diff{DiffInsert, "ABC"},
		Diff{DiffEqual, "1234"},
		Diff{DiffDelete, "wxyz"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abc"},
		Diff{DiffInsert, "ABC"},
		Diff{DiffEqual, "1234"},
		Diff{DiffDelete, "wxyz"}}, diffs)

	// Simple elimination.
	diffs = []Diff{
		Diff{DiffDelete, "a"},
		Diff{DiffEqual, "b"},
		Diff{DiffDelete, "c"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abc"},
		Diff{DiffInsert, "b"}}, diffs)

	// Backpass elimination.
	diffs = []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffEqual, "cd"},
		Diff{DiffDelete, "e"},
		Diff{DiffEqual, "f"},
		Diff{DiffInsert, "g"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abcdef"},
		Diff{DiffInsert, "cdfg"}}, diffs)

	// Multiple eliminations.
	diffs = []Diff{
		Diff{DiffInsert, "1"},
		Diff{DiffEqual, "A"},
		Diff{DiffDelete, "B"},
		Diff{DiffInsert, "2"},
		Diff{DiffEqual, "_"},
		Diff{DiffInsert, "1"},
		Diff{DiffEqual, "A"},
		Diff{DiffDelete, "B"},
		Diff{DiffInsert, "2"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "AB_AB"},
		Diff{DiffInsert, "1A2_1A2"}}, diffs)

	// Word boundaries.
	diffs = []Diff{
		Diff{DiffEqual, "The c"},
		Diff{DiffDelete, "ow and the c"},
		Diff{DiffEqual, "at."}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffEqual, "The "},
		Diff{DiffDelete, "cow and the "},
		Diff{DiffEqual, "cat."}}, diffs)

	// No overlap elimination.
	diffs = []Diff{
		Diff{DiffDelete, "abcxx"},
		Diff{DiffInsert, "xxdef"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abcxx"},
		Diff{DiffInsert, "xxdef"}}, diffs)

	// Overlap elimination.
	diffs = []Diff{
		Diff{DiffDelete, "abcxxx"},
		Diff{DiffInsert, "xxxdef"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abc"},
		Diff{DiffEqual, "xxx"},
		Diff{DiffInsert, "def"}}, diffs)

	// Reverse overlap elimination.
	diffs = []Diff{
		Diff{DiffDelete, "xxxabc"},
		Diff{DiffInsert, "defxxx"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffInsert, "def"},
		Diff{DiffEqual, "xxx"},
		Diff{DiffDelete, "abc"}}, diffs)

	// Two overlap eliminations.
	diffs = []Diff{
		Diff{DiffDelete, "abcd1212"},
		Diff{DiffInsert, "1212efghi"},
		Diff{DiffEqual, "----"},
		Diff{DiffDelete, "A3"},
		Diff{DiffInsert, "3BC"}}
	diffs = dmp.DiffCleanupSemantic(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abcd"},
		Diff{DiffEqual, "1212"},
		Diff{DiffInsert, "efghi"},
		Diff{DiffEqual, "----"},
		Diff{DiffDelete, "A"},
		Diff{DiffEqual, "3"},
		Diff{DiffInsert, "BC"}}, diffs)
}

func Test_diffCleanupEfficiency(t *testing.T) {
	dmp := createDMP()
	// Cleanup operationally trivial equalities.
	dmp.DiffEditCost = 4
	// Null case.
	diffs := []Diff{}
	diffs = dmp.DiffCleanupEfficiency(diffs)
	assertDiffEqual(t, []Diff{}, diffs)

	// No elimination.
	diffs = []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "12"},
		Diff{DiffEqual, "wxyz"},
		Diff{DiffDelete, "cd"},
		Diff{DiffInsert, "34"}}
	diffs = dmp.DiffCleanupEfficiency(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "12"},
		Diff{DiffEqual, "wxyz"},
		Diff{DiffDelete, "cd"},
		Diff{DiffInsert, "34"}}, diffs)

	// Four-edit elimination.
	diffs = []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "12"},
		Diff{DiffEqual, "xyz"},
		Diff{DiffDelete, "cd"},
		Diff{DiffInsert, "34"}}
	diffs = dmp.DiffCleanupEfficiency(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abxyzcd"},
		Diff{DiffInsert, "12xyz34"}}, diffs)

	// Three-edit elimination.
	diffs = []Diff{
		Diff{DiffInsert, "12"},
		Diff{DiffEqual, "x"},
		Diff{DiffDelete, "cd"},
		Diff{DiffInsert, "34"}}
	diffs = dmp.DiffCleanupEfficiency(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "xcd"},
		Diff{DiffInsert, "12x34"}}, diffs)

	// Backpass elimination.
	diffs = []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "12"},
		Diff{DiffEqual, "xy"},
		Diff{DiffInsert, "34"},
		Diff{DiffEqual, "z"},
		Diff{DiffDelete, "cd"},
		Diff{DiffInsert, "56"}}
	diffs = dmp.DiffCleanupEfficiency(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abxyzcd"},
		Diff{DiffInsert, "12xy34z56"}}, diffs)

	// High cost elimination.
	dmp.DiffEditCost = 5
	diffs = []Diff{
		Diff{DiffDelete, "ab"},
		Diff{DiffInsert, "12"},
		Diff{DiffEqual, "wxyz"},
		Diff{DiffDelete, "cd"},
		Diff{DiffInsert, "34"}}
	diffs = dmp.DiffCleanupEfficiency(diffs)
	assertDiffEqual(t, []Diff{
		Diff{DiffDelete, "abwxyzcd"},
		Diff{DiffInsert, "12wxyz34"}}, diffs)
	dmp.DiffEditCost = 4
}

/*
func Test_diffPrettyHtml(t *testing.T) {
    dmp := createDMP()
    // Pretty print.
    diffs := []Diff{
        Diff{DiffEqual, "a\n"},
        Diff{DiffDelete, "<B>b</B>"},
        Diff{DiffInsert, "c&d"}}
    assert.Equal(t, "<span>a&para;<br></span><del style=\"background:#ffe6e6;\">&lt;B&gt;b&lt;/B&gt;</del><ins style=\"background:#e6ffe6;\">c&amp;d</ins>",
        dmp.DiffPrettyHtml(diffs))
}*/

func Test_diffText(t *testing.T) {
	dmp := createDMP()
	// Compute the source and destination texts.
	diffs := []Diff{
		Diff{DiffEqual, "jump"},
		Diff{DiffDelete, "s"},
		Diff{DiffInsert, "ed"},
		Diff{DiffEqual, " over "},
		Diff{DiffDelete, "the"},
		Diff{DiffInsert, "a"},
		Diff{DiffEqual, " lazy"}}
	assert.Equal(t, "jumps over the lazy", dmp.DiffText1(diffs))
	assert.Equal(t, "jumped over a lazy", dmp.DiffText2(diffs))
}

func Test_diffDelta(t *testing.T) {
	dmp := createDMP()
	// Convert a diff into delta string.
	diffs := []Diff{
		Diff{DiffEqual, "jump"},
		Diff{DiffDelete, "s"},
		Diff{DiffInsert, "ed"},
		Diff{DiffEqual, " over "},
		Diff{DiffDelete, "the"},
		Diff{DiffInsert, "a"},
		Diff{DiffEqual, " lazy"},
		Diff{DiffInsert, "old dog"}}

	text1 := dmp.DiffText1(diffs)
	assert.Equal(t, "jumps over the lazy", text1)

	delta := dmp.DiffToDelta(diffs)
	assert.Equal(t, "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", delta)

	// Convert delta string into a diff.
	_seq1, err := dmp.DiffFromDelta(text1, delta)
	assertDiffEqual(t, diffs, _seq1)

	// Generates error (19 < 20).
	_, err = dmp.DiffFromDelta(text1+"x", delta)
	if err == nil {
		panic(1) //assert.Fail("diff_fromDelta: Too long.");
	}

	// Generates error (19 > 18).
	_, err = dmp.DiffFromDelta(text1[1:], delta)
	if err == nil {
		panic(1) //assert.Fail("diff_fromDelta: Too short.");
	}

	// Generates error (%c3%xy invalid Unicode).
	/*
		seq, err := dmp.DiffFromDelta("", "+%c3%xy")
		if err == nil {
			panic(1) //assert.Fail("diff_fromDelta: Invalid character.");
		}
	*/

	// Test deltas with special characters.
	diffs = []Diff{
		Diff{DiffEqual, "\u0680 \x00 \t %"},
		Diff{DiffDelete, "\u0681 \x01 \n ^"},
		Diff{DiffInsert, "\u0682 \x02 \\ |"}}
	text1 = dmp.DiffText1(diffs)
	assert.Equal(t, "\u0680 \x00 \t %\u0681 \x01 \n ^", text1)

	delta = dmp.DiffToDelta(diffs)
	// Lowercase, due to UrlEncode uses lower.
	assert.Equal(t, "=7\t-7\t+%DA%82 %02 %5C %7C", delta)

	_res1, _ := dmp.DiffFromDelta(text1, delta)
	assertDiffEqual(t, diffs, _res1)

	// Verify pool of unchanged characters.
	diffs = []Diff{
		Diff{DiffInsert, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # "}}
	text2 := dmp.DiffText2(diffs)
	assert.Equal(t, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # ", text2, "diff_text2: Unchanged characters.")

	delta = dmp.DiffToDelta(diffs)
	assert.Equal(t, "+A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # ", delta, "diff_toDelta: Unchanged characters.")

	// Convert delta string into a diff.
	_res2, _ := dmp.DiffFromDelta("", delta)
	assertDiffEqual(t, diffs, _res2)
}

func Test_diffXIndex(t *testing.T) {
	dmp := createDMP()
	// Translate a location in text1 to text2.
	diffs := []Diff{
		Diff{DiffDelete, "a"},
		Diff{DiffInsert, "1234"},
		Diff{DiffEqual, "xyz"}}
	assert.Equal(t, 5, dmp.DiffXIndex(diffs, 2), "diff_xIndex: Translation on equality.")

	diffs = []Diff{
		Diff{DiffEqual, "a"},
		Diff{DiffDelete, "1234"},
		Diff{DiffEqual, "xyz"}}
	assert.Equal(t, 1, dmp.DiffXIndex(diffs, 3), "diff_xIndex: Translation on deletion.")
}

func Test_diffLevenshtein(t *testing.T) {
	dmp := createDMP()
	diffs := []Diff{
		Diff{DiffDelete, "abc"},
		Diff{DiffInsert, "1234"},
		Diff{DiffEqual, "xyz"}}
	assert.Equal(t, 4, dmp.DiffLevenshtein(diffs), "diff_levenshtein: Levenshtein with trailing equality.")

	diffs = []Diff{
		Diff{DiffEqual, "xyz"},
		Diff{DiffDelete, "abc"},
		Diff{DiffInsert, "1234"}}
	assert.Equal(t, 4, dmp.DiffLevenshtein(diffs), "diff_levenshtein: Levenshtein with leading equality.")

	diffs = []Diff{
		Diff{DiffDelete, "abc"},
		Diff{DiffEqual, "xyz"},
		Diff{DiffInsert, "1234"}}
	assert.Equal(t, 7, dmp.DiffLevenshtein(diffs), "diff_levenshtein: Levenshtein with middle equality.")
}

func Test_diffBisect(t *testing.T) {
    dmp := createDMP()
    // Normal.
    a := "cat"
    b := "map"
    // Since the resulting diff hasn't been normalized, it would be ok if
    // the insertion and deletion pairs are swapped.
    // If the order changes, tweak this test as required.
	diffs := []Diff{
		Diff{DiffDelete, "c"},
		Diff{DiffInsert, "m"},
		Diff{DiffEqual, "a"},
		Diff{DiffDelete, "t"},
		Diff{DiffInsert, "p"}}

	fmt.Println(dmp.DiffBisect(a, b, int32(time.Date(9999, time.December, 31, 23, 59, 59, 59, time.UTC).Unix())))

	assertDiffEqual(t, diffs, dmp.DiffBisect(a, b, int32(time.Date(9999, time.December, 31, 23, 59, 59, 59, time.UTC).Unix()))) //TODO

    // Timeout.
    diffs = []Diff{Diff{DiffDelete, "cat"}, Diff{DiffInsert, "map"}}
	assertDiffEqual(t, diffs, dmp.DiffBisect(a, b, int32(time.Date(0001, time.January, 01, 00, 00, 00, 00, time.UTC).Unix()))) //TODO
}

func Test_diffMain(t *testing.T) {
	dmp := createDMP()
	// Perform a trivial diff.
	diffs := []Diff{}
	assertDiffEqual(t, diffs, dmp.DiffMain("", "", false))

	diffs = []Diff{Diff{DiffEqual, "abc"}}
	assertDiffEqual(t, diffs, dmp.DiffMain("abc", "abc", false))

	diffs = []Diff{Diff{DiffEqual, "ab"}, Diff{DiffInsert, "123"}, Diff{DiffEqual, "c"}}
	assertDiffEqual(t, diffs, dmp.DiffMain("abc", "ab123c", false))

	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "123"}, Diff{DiffEqual, "bc"}}
	assertDiffEqual(t, diffs, dmp.DiffMain("a123bc", "abc", false))

	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffInsert, "123"}, Diff{DiffEqual, "b"}, Diff{DiffInsert, "456"}, Diff{DiffEqual, "c"}}
	assertDiffEqual(t, diffs, dmp.DiffMain("abc", "a123b456c", false))

	diffs = []Diff{Diff{DiffEqual, "a"}, Diff{DiffDelete, "123"}, Diff{DiffEqual, "b"}, Diff{DiffDelete, "456"}, Diff{DiffEqual, "c"}}
	assertDiffEqual(t, diffs, dmp.DiffMain("a123b456c", "abc", false))

	// Perform a real diff.
	// Switch off the timeout.
	dmp.DiffTimeout = 0
	diffs = []Diff{Diff{DiffDelete, "a"}, Diff{DiffInsert, "b"}}
<<<<<<< HEAD
	assertSeqEqual(diffs, dmp.DiffMain("a", "b", false))
=======
	assertDiffEqual(t, diffs, dmp.diffMain("a", "b", false))
>>>>>>> f737cb4... reworked assertions in tests to be more informative

	diffs = []Diff{
		Diff{DiffDelete, "Apple"},
		Diff{DiffInsert, "Banana"},
		Diff{DiffEqual, "s are a"},
		Diff{DiffInsert, "lso"},
		Diff{DiffEqual, " fruit."}}

<<<<<<< HEAD
	assertSeqEqual(diffs, dmp.DiffMain("Apples are a fruit.", "Bananas are also fruit.", false))
=======
	assertDiffEqual(t, diffs, dmp.diffMain("Apples are a fruit.", "Bananas are also fruit.", false))
>>>>>>> f737cb4... reworked assertions in tests to be more informative

	diffs = []Diff{
		Diff{DiffDelete, "a"},
		Diff{DiffInsert, "\u0680"},
		Diff{DiffEqual, "x"},
		Diff{DiffDelete, "\t"},
		Diff{DiffInsert, "\u0000"}}


<<<<<<< HEAD
	assertSeqEqual(diffs, dmp.DiffMain("ax\t", "\u0680x\u0000", false))
	diffs = []Diff{Diff{DiffDelete, "1"}, Diff{DiffEqual, "a"}, Diff{DiffDelete, "y"}, Diff{DiffEqual, "b"}, Diff{DiffDelete, "2"}, Diff{DiffInsert, "xab"}}
	assertSeqEqual(diffs, dmp.DiffMain("1ayb2", "abxab", false))

	diffs = []Diff{Diff{DiffInsert, "xaxcx"}, Diff{DiffEqual, "abc"}, Diff{DiffDelete, "y"}}
	assertSeqEqual(diffs, dmp.DiffMain("abcy", "xaxcxabc", false))

	diffs = []Diff{Diff{DiffDelete, "ABCD"}, Diff{DiffEqual, "a"}, Diff{DiffDelete, "="}, Diff{DiffInsert, "-"}, Diff{DiffEqual, "bcd"}, Diff{DiffDelete, "="}, Diff{DiffInsert, "-"}, Diff{DiffEqual, "efghijklmnopqrs"}, Diff{DiffDelete, "EFGHIJKLMNOefg"}}
	assertSeqEqual(diffs, dmp.DiffMain("ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg", "a-bcd-efghijklmnopqrs", false))

	diffs = []Diff{Diff{DiffInsert, " "}, Diff{DiffEqual, "a"}, Diff{DiffInsert, "nd"}, Diff{DiffEqual, " [[Pennsylvania]]"}, Diff{DiffDelete, " and [[New"}}
	assertSeqEqual(diffs, dmp.DiffMain("a [[Pennsylvania]] and [[New", " and [[Pennsylvania]]", false))
=======
	assertDiffEqual(t, diffs, dmp.diffMain("ax\t", "\u0680x\u0000", false))
	diffs = []Diff{Diff{DiffDelete, "1"}, Diff{DiffEqual, "a"}, Diff{DiffDelete, "y"}, Diff{DiffEqual, "b"}, Diff{DiffDelete, "2"}, Diff{DiffInsert, "xab"}}
	assertDiffEqual(t, diffs, dmp.diffMain("1ayb2", "abxab", false))

	diffs = []Diff{Diff{DiffInsert, "xaxcx"}, Diff{DiffEqual, "abc"}, Diff{DiffDelete, "y"}}
	assertDiffEqual(t, diffs, dmp.diffMain("abcy", "xaxcxabc", false))

	diffs = []Diff{Diff{DiffDelete, "ABCD"}, Diff{DiffEqual, "a"}, Diff{DiffDelete, "="}, Diff{DiffInsert, "-"}, Diff{DiffEqual, "bcd"}, Diff{DiffDelete, "="}, Diff{DiffInsert, "-"}, Diff{DiffEqual, "efghijklmnopqrs"}, Diff{DiffDelete, "EFGHIJKLMNOefg"}}
	assertDiffEqual(t, diffs, dmp.diffMain("ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg", "a-bcd-efghijklmnopqrs", false))

	diffs = []Diff{Diff{DiffInsert, " "}, Diff{DiffEqual, "a"}, Diff{DiffInsert, "nd"}, Diff{DiffEqual, " [[Pennsylvania]]"}, Diff{DiffDelete, " and [[New"}}
	assertDiffEqual(t, diffs, dmp.diffMain("a [[Pennsylvania]] and [[New", " and [[Pennsylvania]]", false))
>>>>>>> f737cb4... reworked assertions in tests to be more informative

	dmp.DiffTimeout = 0.1 // 100ms
	a := "`Twas brillig, and the slithy toves\nDid gyre and gimble in the wabe:\nAll mimsy were the borogoves,\nAnd the mome raths outgrabe.\n"
	b := "I am the very model of a modern major general,\nI've information vegetable, animal, and mineral,\nI know the kings of England, and I quote the fights historical,\nFrom Marathon to Waterloo, in order categorical.\n"
	// Increase the text lengths by 1024 times to ensure a timeout.
	for x := 0; x < 10; x++ {
		a = a + a
		b = b + b
	}
	startTime := time.Now().Unix()
	startTime *= 1000
	dmp.DiffMain(a, b)
	endTime := time.Now().Unix()
	endTime *= 1000
	// Test that we took at least the timeout period.
	softAssert(t, dmp.DiffTimeout*1000 <= float64(endTime-startTime), "")
	// Test that we didn't take forever (be forgiving).
	// Theoretically this test could fail very occasionally if the
	// OS task swaps or locks up for a second at the wrong moment.
	softAssert(t, dmp.DiffTimeout*1000*2 > float64(endTime-startTime), "")
	dmp.DiffTimeout = 0

	// Test the linemode speedup.
	// Must be long to pass the 100 char cutoff.
	a = "1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n"
	b = "abcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\nabcdefghij\n"
<<<<<<< HEAD
	assertSeqEqual(dmp.DiffMain(a, b, true), dmp.DiffMain(a, b, false))

	a = "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	b = "abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij"
	assertSeqEqual(dmp.DiffMain(a, b, true), dmp.DiffMain(a, b, false))

	a = "1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n"
	b = "abcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n"
	texts_linemode := diffRebuildtexts(dmp.DiffMain(a, b, true))
	texts_textmode := diffRebuildtexts(dmp.DiffMain(a, b, false))
	assertSeqEqual(texts_textmode, texts_linemode)
=======
	assertDiffEqual(t, dmp.diffMain(a, b, true), dmp.diffMain(a, b, false))

	a = "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	b = "abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij"
	assertDiffEqual(t, dmp.diffMain(a, b, true), dmp.diffMain(a, b, false))

	a = "1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n"
	b = "abcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n"
	texts_linemode := diffRebuildtexts(dmp.diffMain(a, b, true))
	texts_textmode := diffRebuildtexts(dmp.diffMain(a, b, false))
	assertStrEqual(t, texts_textmode, texts_linemode)
>>>>>>> f737cb4... reworked assertions in tests to be more informative

	// Test null inputs -- not needed because nulls can't be passed in C#.
}

func Test_match_alphabet(t *testing.T) {
	dmp := createDMP()
	// Initialise the bitmasks for Bitap.
	bitmask := map[byte]int{
		'a': 4,
		'b': 2,
		'c': 1,
	}
	assertMapEqual(t, bitmask, dmp.MatchAlphabet("abc"))

	bitmask = map[byte]int{
		'a': 37,
		'b': 18,
		'c': 8,
	}
	assertMapEqual(t, bitmask, dmp.MatchAlphabet("abcaba"))
}

func Test_match_bitap(t *testing.T) {
	dmp := createDMP()

	// Bitap algorithm.
	dmp.MatchDistance = 100
	dmp.MatchThreshold = 0.5
	assert.Equal(t, 5, dmp.MatchBitap("abcdefghijk", "fgh", 5), "match_bitap: Exact match #1.")

	assert.Equal(t, 5, dmp.MatchBitap("abcdefghijk", "fgh", 0), "match_bitap: Exact match #2.")

	assert.Equal(t, 4, dmp.MatchBitap("abcdefghijk", "efxhi", 0), "match_bitap: Fuzzy match #1.")

	assert.Equal(t, 2, dmp.MatchBitap("abcdefghijk", "cdefxyhijk", 5), "match_bitap: Fuzzy match #2.")

	assert.Equal(t, -1, dmp.MatchBitap("abcdefghijk", "bxy", 1), "match_bitap: Fuzzy match #3.")

	assert.Equal(t, 2, dmp.MatchBitap("123456789xx0", "3456789x0", 2), "match_bitap: Overflow.")

	assert.Equal(t, 0, dmp.MatchBitap("abcdef", "xxabc", 4), "match_bitap: Before start match.")

	assert.Equal(t, 3, dmp.MatchBitap("abcdef", "defyy", 4), "match_bitap: Beyond end match.")

	assert.Equal(t, 0, dmp.MatchBitap("abcdef", "xabcdefy", 0), "match_bitap: Oversized pattern.")

	dmp.MatchThreshold = 0.4
	assert.Equal(t, 4, dmp.MatchBitap("abcdefghijk", "efxyhi", 1), "match_bitap: Threshold #1.")

	dmp.MatchThreshold = 0.3
	assert.Equal(t, -1, dmp.MatchBitap("abcdefghijk", "efxyhi", 1), "match_bitap: Threshold #2.")

	dmp.MatchThreshold = 0.0
	assert.Equal(t, 1, dmp.MatchBitap("abcdefghijk", "bcdef", 1), "match_bitap: Threshold #3.")

	dmp.MatchThreshold = 0.5
	assert.Equal(t, 0, dmp.MatchBitap("abcdexyzabcde", "abccde", 3), "match_bitap: Multiple select #1.")

	assert.Equal(t, 8, dmp.MatchBitap("abcdexyzabcde", "abccde", 5), "match_bitap: Multiple select #2.")

	dmp.MatchDistance = 10 // Strict location.
	assert.Equal(t, -1, dmp.MatchBitap("abcdefghijklmnopqrstuvwxyz", "abcdefg", 24), "match_bitap: Distance test #1.")

	assert.Equal(t, 0, dmp.MatchBitap("abcdefghijklmnopqrstuvwxyz", "abcdxxefg", 1), "match_bitap: Distance test #2.")

	dmp.MatchDistance = 1000 // Loose location.
	assert.Equal(t, 0, dmp.MatchBitap("abcdefghijklmnopqrstuvwxyz", "abcdefg", 24), "match_bitap: Distance test #3.")
}

func Test_MatchMain(t *testing.T) {
	dmp := createDMP()
	// Full match.
	assert.Equal(t, 0, dmp.MatchMain("abcdef", "abcdef", 1000), "MatchMain: Equality.")

	assert.Equal(t, -1, dmp.MatchMain("", "abcdef", 1), "MatchMain: Null text.")

	assert.Equal(t, 3, dmp.MatchMain("abcdef", "", 3), "MatchMain: Null pattern.")

	assert.Equal(t, 3, dmp.MatchMain("abcdef", "de", 3), "MatchMain: Exact match.")

	assert.Equal(t, 3, dmp.MatchMain("abcdef", "defy", 4), "MatchMain: Beyond end match.")

	assert.Equal(t, 0, dmp.MatchMain("abcdef", "abcdefy", 0), "MatchMain: Oversized pattern.")

	dmp.MatchThreshold = 0.7
	assert.Equal(t, 4, dmp.MatchMain("I am the very model of a modern major general.", " that berry ", 5), "MatchMain: Complex match.")
	dmp.MatchThreshold = 0.5

	// Test null inputs -- not needed because nulls can't be passed in C#.
}

func Test_patch_patchObj(t *testing.T) {
	// Patch Object.
	p := Patch{}
	p.start1 = 20
	p.start2 = 21
	p.length1 = 18
	p.length2 = 17
	p.diffs = []Diff{
		Diff{DiffEqual, "jump"},
		Diff{DiffDelete, "s"},
		Diff{DiffInsert, "ed"},
		Diff{DiffEqual, " over "},
		Diff{DiffDelete, "the"},
		Diff{DiffInsert, "a"},
		Diff{DiffEqual, "\nlaz"}}
	strp := "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n %0Alaz\n"

	assert.Equal(t, strp, p.String(), "Patch: toString.")
}

func Test_patch_fromText(t *testing.T) {
	dmp := createDMP()

	_v1, _ := dmp.PatchFromText("")
	softAssert(t, len(_v1) == 0, "patch_fromText: #0.")
	strp := "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n %0Alaz\n"
	_v2, _ := dmp.PatchFromText(strp)
	assert.Equal(t, strp, _v2[0].String(), "patch_fromText: #1.")

	_v3, _ := dmp.PatchFromText("@@ -1 +1 @@\n-a\n+b\n")
	assert.Equal(t, "@@ -1 +1 @@\n-a\n+b\n", _v3[0].String(), "patch_fromText: #2.")

	_v4, _ := dmp.PatchFromText("@@ -1,3 +0,0 @@\n-abc\n")
	assert.Equal(t, "@@ -1,3 +0,0 @@\n-abc\n", _v4[0].String(), "patch_fromText: #3.")

	_v5, _ := dmp.PatchFromText("@@ -0,0 +1,3 @@\n+abc\n")
	assert.Equal(t, "@@ -0,0 +1,3 @@\n+abc\n", _v5[0].String(), "patch_fromText: #4.")

	// Generates error.
	_, err := dmp.PatchFromText("Bad\nPatch\n")
	softAssert(t, err != nil, "There should be an error")
}

func Test_patch_toText(t *testing.T) {
	dmp := createDMP()
	strp := "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"
	var patches []Patch
	patches, _ = dmp.PatchFromText(strp)
	result := dmp.PatchToText(patches)
	assert.Equal(t, strp, result)

	strp = "@@ -1,9 +1,9 @@\n-f\n+F\n oo+fooba\n@@ -7,9 +7,9 @@\n obar\n-,\n+.\n  tes\n"
	patches, _ = dmp.PatchFromText(strp)
	result = dmp.PatchToText(patches)
	assert.Equal(t, strp, result)
}

func Test_patch_addContext(t *testing.T) {
	dmp := createDMP()
	dmp.PatchMargin = 4
	var p Patch
	_p, _ := dmp.PatchFromText("@@ -21,4 +21,10 @@\n-jump\n+somersault\n")
	p = _p[0]
	p = dmp.PatchAddContext(p, "The quick brown fox jumps over the lazy dog.")
	assert.Equal(t, "@@ -17,12 +17,18 @@\n fox \n-jump\n+somersault\n s ov\n", p.String(), "patch_addContext: Simple case.")

	_p, _ = dmp.PatchFromText("@@ -21,4 +21,10 @@\n-jump\n+somersault\n")
	p = _p[0]
	p = dmp.PatchAddContext(p, "The quick brown fox jumps.")
	assert.Equal(t, "@@ -17,10 +17,16 @@\n fox \n-jump\n+somersault\n s.\n", p.String(), "patch_addContext: Not enough trailing context.")

	_p, _ = dmp.PatchFromText("@@ -3 +3,2 @@\n-e\n+at\n")
	p = _p[0]
	p = dmp.PatchAddContext(p, "The quick brown fox jumps.")
	assert.Equal(t, "@@ -1,7 +1,8 @@\n Th\n-e\n+at\n  qui\n", p.String(), "patch_addContext: Not enough leading context.")

	_p, _ = dmp.PatchFromText("@@ -3 +3,2 @@\n-e\n+at\n")
	p = _p[0]
	p = dmp.PatchAddContext(p, "The quick brown fox jumps.  The quick brown fox crashes.")
	assert.Equal(t, "@@ -1,27 +1,28 @@\n Th\n-e\n+at\n  quick brown fox jumps. \n", p.String(), "patch_addContext: Ambiguity.")
}

func Test_patch_make(t *testing.T) {
	dmp := createDMP()
	var patches []Patch
	patches = dmp.PatchMake("", "")
	assert.Equal(t, "", dmp.PatchToText(patches), "patch_make: Null case.")

	text1 := "The quick brown fox jumps over the lazy dog."
	text2 := "That quick brown fox jumped over a lazy dog."
	expectedPatch := "@@ -1,8 +1,7 @@\n Th\n-at\n+e\n  qui\n@@ -21,17 +21,18 @@\n jump\n-ed\n+s\n  over \n-a\n+the\n  laz\n"
	// The second patch must be "-21,17 +21,18", not "-22,17 +21,18" due to rolling context.
	patches = dmp.PatchMake(text2, text1)
	assert.Equal(t, expectedPatch, dmp.PatchToText(patches), "patch_make: Text2+Text1 inputs.")

	expectedPatch = "@@ -1,11 +1,12 @@\n Th\n-e\n+at\n  quick b\n@@ -22,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"
	patches = dmp.PatchMake(text1, text2)
	assert.Equal(t, expectedPatch, dmp.PatchToText(patches), "patch_make: Text1+Text2 inputs.")

	diffs := dmp.DiffMain(text1, text2, false)
	patches = dmp.PatchMake(diffs)
	assert.Equal(t, expectedPatch, dmp.PatchToText(patches), "patch_make: Diff input.")

	patches = dmp.PatchMake(text1, diffs)
	assert.Equal(t, expectedPatch, dmp.PatchToText(patches), "patch_make: Text1+Diff inputs.")

	patches = dmp.PatchMake(text1, text2, diffs)
	assert.Equal(t, expectedPatch, dmp.PatchToText(patches), "patch_make: Text1+Text2+Diff inputs (deprecated).")

	patches = dmp.PatchMake("`1234567890-=[]\\;',./", "~!@#$%^&*()_+{}|:\"<>?")
	assert.Equal(t, "@@ -1,21 +1,21 @@\n-%601234567890-=%5b%5d%5c;',./\n+~!@#$%25%5e&*()_+%7b%7d%7c:%22%3c%3e?\n",
		dmp.PatchToText(patches),
		"patch_toText: Character encoding.")

	diffs = []Diff{
		Diff{DiffDelete, "`1234567890-=[]\\;',./"},
		Diff{DiffInsert, "~!@#$%^&*()_+{}|:\"<>?"}}

	_p1, _ := dmp.PatchFromText("@@ -1,21 +1,21 @@\n-%601234567890-=%5B%5D%5C;',./\n+~!@#$%25%5E&*()_+%7B%7D%7C:%22%3C%3E?\n")
	assertDiffEqual(t, diffs,
		_p1[0].diffs,
	)

	text1 = ""
	for x := 0; x < 100; x++ {
		text1 += "abcdef"
	}
	text2 = text1 + "123"
	expectedPatch = "@@ -573,28 +573,31 @@\n cdefabcdefabcdefabcdefabcdef\n+123\n"
	patches = dmp.PatchMake(text1, text2)
	assert.Equal(t, expectedPatch, dmp.PatchToText(patches), "patch_make: Long string with repeats.")

	// Test null inputs -- not needed because nulls can't be passed in C#.
}

func Test_PatchSplitMax(t *testing.T) {
	// Assumes that Match_MaxBits is 32.
	dmp := createDMP()
	var patches []Patch

	patches = dmp.PatchMake("abcdefghijklmnopqrstuvwxyz01234567890", "XabXcdXefXghXijXklXmnXopXqrXstXuvXwxXyzX01X23X45X67X89X0")
	dmp.PatchSplitMax(patches)
	assert.Equal(t, "@@ -1,32 +1,46 @@\n+X\n ab\n+X\n cd\n+X\n ef\n+X\n gh\n+X\n ij\n+X\n kl\n+X\n mn\n+X\n op\n+X\n qr\n+X\n st\n+X\n uv\n+X\n wx\n+X\n yz\n+X\n 012345\n@@ -25,13 +39,18 @@\n zX01\n+X\n 23\n+X\n 45\n+X\n 67\n+X\n 89\n+X\n 0\n", dmp.PatchToText(patches))

	patches = dmp.PatchMake("abcdef1234567890123456789012345678901234567890123456789012345678901234567890uvwxyz", "abcdefuvwxyz")
	oldToText := dmp.PatchToText(patches)
	dmp.PatchSplitMax(patches)
	assert.Equal(t, oldToText, dmp.PatchToText(patches))

	patches = dmp.PatchMake("1234567890123456789012345678901234567890123456789012345678901234567890", "abc")
	dmp.PatchSplitMax(patches)
	assert.Equal(t, "@@ -1,32 +1,4 @@\n-1234567890123456789012345678\n 9012\n@@ -29,32 +1,4 @@\n-9012345678901234567890123456\n 7890\n@@ -57,14 +1,3 @@\n-78901234567890\n+abc\n", dmp.PatchToText(patches))

	patches = dmp.PatchMake("abcdefghij , h : 0 , t : 1 abcdefghij , h : 0 , t : 1 abcdefghij , h : 0 , t : 1", "abcdefghij , h : 1 , t : 1 abcdefghij , h : 1 , t : 1 abcdefghij , h : 0 , t : 1")
	dmp.PatchSplitMax(patches)
	assert.Equal(t, "@@ -2,32 +2,32 @@\n bcdefghij , h : \n-0\n+1\n  , t : 1 abcdef\n@@ -29,32 +29,32 @@\n bcdefghij , h : \n-0\n+1\n  , t : 1 abcdef\n", dmp.PatchToText(patches))
}

func Test_PatchAddPadding(t *testing.T) {
	dmp := createDMP()
	var patches []Patch
	patches = dmp.PatchMake("", "test")
	assert.Equal(t, "@@ -0,0 +1,4 @@\n+test\n",
		dmp.PatchToText(patches),
		"PatchAddPadding: Both edges full.")
	dmp.PatchAddPadding(patches)
	assert.Equal(t, "@@ -1,8 +1,12 @@\n %01%02%03%04\n+test\n %01%02%03%04\n",
		dmp.PatchToText(patches),
		"PatchAddPadding: Both edges full.")

	patches = dmp.PatchMake("XY", "XtestY")
	assert.Equal(t, "@@ -1,2 +1,6 @@\n X\n+test\n Y\n",
		dmp.PatchToText(patches),
		"PatchAddPadding: Both edges partial.")
	dmp.PatchAddPadding(patches)
	assert.Equal(t, "@@ -2,8 +2,12 @@\n %02%03%04X\n+test\n Y%01%02%03\n",
		dmp.PatchToText(patches),
		"PatchAddPadding: Both edges partial.")

	patches = dmp.PatchMake("XXXXYYYY", "XXXXtestYYYY")
	assert.Equal(t, "@@ -1,8 +1,12 @@\n XXXX\n+test\n YYYY\n",
		dmp.PatchToText(patches),
		"PatchAddPadding: Both edges none.")
	dmp.PatchAddPadding(patches)
	assert.Equal(t, "@@ -5,8 +5,12 @@\n XXXX\n+test\n YYYY\n",
		dmp.PatchToText(patches),
		"PatchAddPadding: Both edges none.")
}

func Test_patchApply(t *testing.T) {
	dmp := createDMP()
	dmp.MatchDistance = 1000
	dmp.MatchThreshold = 0.5
	dmp.PatchDeleteThreshold = 0.5
	patches := []Patch{}
	patches = dmp.PatchMake("", "")
	results0, results1 := dmp.PatchApply(patches, "Hello world.")
	boolArray := results1
	resultStr := results0 + "\t" + string(len(boolArray))
	assert.Equal(t, "Hello world.\t0", resultStr, "patch_apply: Null case.")

	patches = dmp.PatchMake("The quick brown fox jumps over the lazy dog.", "That quick brown fox jumped over a lazy dog.")
	results0, results1 = dmp.PatchApply(patches, "The quick brown fox jumps over the lazy dog.")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "That quick brown fox jumped over a lazy dog.\tTrue\tTrue", resultStr, "patch_apply: Exact match.")

	results0, results1 = dmp.PatchApply(patches, "The quick red rabbit jumps over the tired tiger.")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "That quick red rabbit jumped over a tired tiger.\tTrue\tTrue", resultStr, "patch_apply: Partial match.")

	results0, results1 = dmp.PatchApply(patches, "I am the very model of a modern major general.")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "I am the very model of a modern major general.\tFalse\tFalse", resultStr, "patch_apply: Failed match.")

	patches = dmp.PatchMake("x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy")
	results0, results1 = dmp.PatchApply(patches, "x123456789012345678901234567890-----++++++++++-----123456789012345678901234567890y")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "xabcy\tTrue\tTrue", resultStr, "patch_apply: Big delete, small Diff.")

	patches = dmp.PatchMake("x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy")
	results0, results1 = dmp.PatchApply(patches, "x12345678901234567890---------------++++++++++---------------12345678901234567890y")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "xabc12345678901234567890---------------++++++++++---------------12345678901234567890y\tFalse\tTrue", resultStr, "patch_apply: Big delete, big Diff 1.")

	dmp.PatchDeleteThreshold = 0.6
	patches = dmp.PatchMake("x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy")
	results0, results1 = dmp.PatchApply(patches, "x12345678901234567890---------------++++++++++---------------12345678901234567890y")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "xabcy\tTrue\tTrue", resultStr, "patch_apply: Big delete, big Diff 2.")
	dmp.PatchDeleteThreshold = 0.5

	dmp.MatchThreshold = 0.0
	dmp.MatchDistance = 0
	patches = dmp.PatchMake("abcdefghijklmnopqrstuvwxyz--------------------1234567890", "abcXXXXXXXXXXdefghijklmnopqrstuvwxyz--------------------1234567YYYYYYYYYY890")
	results0, results1 = dmp.PatchApply(patches, "ABCDEFGHIJKLMNOPQRSTUVWXYZ--------------------1234567890")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
	assert.Equal(t, "ABCDEFGHIJKLMNOPQRSTUVWXYZ--------------------1234567YYYYYYYYYY890\tFalse\tTrue", resultStr, "patch_apply: Compensate for failed patch.")
	dmp.MatchThreshold = 0.5
	dmp.MatchDistance = 1000

	patches = dmp.PatchMake("", "test")
	patchStr := dmp.PatchToText(patches)
	dmp.PatchApply(patches, "")
	assert.Equal(t, patchStr, dmp.PatchToText(patches), "patch_apply: No side effects.")

	patches = dmp.PatchMake("The quick brown fox jumps over the lazy dog.", "Woof")
	patchStr = dmp.PatchToText(patches)
	dmp.PatchApply(patches, "The quick brown fox jumps over the lazy dog.")
	assert.Equal(t, patchStr, dmp.PatchToText(patches), "patch_apply: No side effects with major delete.")

	patches = dmp.PatchMake("", "test")
	results0, results1 = dmp.PatchApply(patches, "")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0])
	assert.Equal(t, "test\tTrue", resultStr, "patch_apply: Edge exact match.")

	patches = dmp.PatchMake("XY", "XtestY")
	results0, results1 = dmp.PatchApply(patches, "XY")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0])
	assert.Equal(t, "XtestY\tTrue", resultStr, "patch_apply: Near edge exact match.")

	patches = dmp.PatchMake("y", "y123")
	results0, results1 = dmp.PatchApply(patches, "x")
	boolArray = results1
	resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0])
	assert.Equal(t, "x123\tTrue", resultStr, "patch_apply: Edge partial match.")
}
