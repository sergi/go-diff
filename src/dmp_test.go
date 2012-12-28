package godiff

import (
    "fmt"
    "github.com/bmizerany/assert"
    //"strings"
    "reflect"
    "strconv"
    "testing"
    "time" //import go package for testing related functionality
    //"unicode/utf8"
    //"strconv"
)

func softAssert(t *testing.T, cond bool, msg string) {
    if !cond {
        print("assertion fail: ", msg, "\n")
        panic(1)
    }
}

func assertSeqEqual(seq1, seq2 interface{}) {
    var fail = func(msg string) {
        fmt.Println("assertion fail: \n", msg, "\n")
        panic(1)
    }

    v1 := reflect.ValueOf(seq1)
    k1 := v1.Kind()
    v2 := reflect.ValueOf(seq2)
    k2 := v2.Kind()

    if k1 != reflect.Array && k1 != reflect.Slice {
        fail("Parameters are not slices or Arrays")
    }

    if k2 != reflect.Array && k2 != reflect.Slice {
        fail("Parameters are not slices or Arrays")
    }

    if v1.Len() != v2.Len() {
        fail("Sequences of different length:\n" + string(v1.Len()) + "\n" + string(v2.Len()))
    }

    for i := 0; i < v1.Len(); i++ {
        if v1.Index(i).String() != v2.Index(i).String() {
            fail("[" + v1.Index(i).Kind().String() + "] " + v1.Index(i).String() +
                " != [" + v2.Index(i).Kind().String() + "] " + v2.Index(i).String())
            break
        }
    }
}

func diff_rebuildtexts(diffs []change) []string {
    text := []string{"", ""}
    for _, myDiff := range diffs {
        if myDiff.Type != DIFF_INSERT {
            text[0] += myDiff.Text
        }
        if myDiff.Type != DIFF_DELETE {
            text[1] += myDiff.Text
        }
    }
    return text
}

func Test_diff_commonPrefix(t *testing.T) {
    dmp := createDMP()
    // Detect any common suffix.
    // Null case.
    assert.Equal(t, 0, dmp.diff_commonPrefix("abc", "xyz"), "'abc' and 'xyz' should not be equal")

    // Non-null case.
    assert.Equal(t, 4, dmp.diff_commonPrefix("1234abcdef", "1234xyz"), "")

    // Whole case.
    assert.Equal(t, 4, dmp.diff_commonPrefix("1234", "1234xyz"), "")
}

func Test_diff_commonSuffixTest(t *testing.T) {
    dmp := createDMP()
    // Detect any common suffix.
    // Null case.
    assert.Equal(t, 0, dmp.diff_commonSuffix("abc", "xyz"), "")

    // Non-null case.
    assert.Equal(t, 4, dmp.diff_commonSuffix("abcdef1234", "xyz1234"), "")

    // Whole case.
    assert.Equal(t, 4, dmp.diff_commonSuffix("1234", "xyz1234"), "")
}

func Test_diff_commonOverlapTest(t *testing.T) {
    dmp := createDMP()
    // Detect any suffix/prefix overlap.
    // Null case.
    assert.Equal(t, 0, dmp.diff_commonOverlap("", "abcd"), "")

    // Whole case.
    assert.Equal(t, 3, dmp.diff_commonOverlap("abc", "abcd"), "")

    // No overlap.
    assert.Equal(t, 0, dmp.diff_commonOverlap("123456", "abcd"), "")

    // Overlap.
    assert.Equal(t, 3, dmp.diff_commonOverlap("123456xxx", "xxxabcd"), "")

    // Unicode.
    // Some overly clever languages (C#) may treat ligatures as equal to their
    // component letters.  E.g. U+FB01 == 'fi'
    assert.Equal(t, 0, dmp.diff_commonOverlap("fi", "\ufb01i"), "")
}

func Test_diff_halfmatchTest(t *testing.T) {
    dmp := createDMP()
    dmp.DiffTimeout = 1
    // No match.
    softAssert(t, dmp.diff_halfMatch("1234567890", "abcdef") == nil, "")
    softAssert(t, dmp.diff_halfMatch("12345", "23") == nil, "")

    // Single Match.
    assertSeqEqual(
        []string{"12", "90", "a", "z", "345678"},
        dmp.diff_halfMatch("1234567890", "a345678z"))

    assertSeqEqual([]string{"a", "z", "12", "90", "345678"}, dmp.diff_halfMatch("a345678z", "1234567890"))

    assertSeqEqual([]string{"abc", "z", "1234", "0", "56789"}, dmp.diff_halfMatch("abc56789z", "1234567890"))

    assertSeqEqual([]string{"a", "xyz", "1", "7890", "23456"}, dmp.diff_halfMatch("a23456xyz", "1234567890"))

    // Multiple Matches.
    assertSeqEqual([]string{"12123", "123121", "a", "z", "1234123451234"}, dmp.diff_halfMatch("121231234123451234123121", "a1234123451234z"))

    assertSeqEqual([]string{"", "-=-=-=-=-=", "x", "", "x-=-=-=-=-=-=-="}, dmp.diff_halfMatch("x-=-=-=-=-=-=-=-=-=-=-=-=", "xx-=-=-=-=-=-=-="))

    assertSeqEqual([]string{"-=-=-=-=-=", "", "", "y", "-=-=-=-=-=-=-=y"}, dmp.diff_halfMatch("-=-=-=-=-=-=-=-=-=-=-=-=y", "-=-=-=-=-=-=-=yy"))

    // Non-optimal halfmatch.
    // Optimal diff would be -q+x=H-i+e=lloHe+Hu=llo-Hew+y not -qHillo+x=HelloHe-w+Hulloy
    assertSeqEqual([]string{"qHillo", "w", "x", "Hulloy", "HelloHe"}, dmp.diff_halfMatch("qHilloHelloHew", "xHelloHeHulloy"))

    // Optimal no halfmatch.
    dmp.DiffTimeout = 0
    softAssert(t, dmp.diff_halfMatch("qHilloHelloHew", "xHelloHeHulloy") == nil, "")
}

func Test_diff_linesToChars(t *testing.T) {
    dmp := createDMP()
    // Convert lines down to characters.
    tmpVector := []string{"", "alpha\n", "beta\n"}

    result0, result1, result2 := dmp.diff_linesToChars("alpha\nbeta\nalpha\n", "beta\nalpha\nbeta\n")
    assert.Equal(t, "\u0001\u0002\u0001", result0, "")
    assert.Equal(t, "\u0002\u0001\u0002", result1, "")
    assertSeqEqual(tmpVector, result2)

    tmpVector = []string{"", "alpha\r\n", "beta\r\n", "\r\n"}
    result0, result1, result2 = dmp.diff_linesToChars("", "alpha\r\nbeta\r\n\r\n\r\n")
    assert.Equal(t, "", result0, "")
    assert.Equal(t, "\u0001\u0002\u0003\u0003", result1, "")
    assertSeqEqual(tmpVector, result2)

    tmpVector = []string{"", "a", "b"}
    result0, result1, result2 = dmp.diff_linesToChars("a", "b")
    assert.Equal(t, "\u0001", result0, "")
    assert.Equal(t, "\u0002", result1, "")
    assertSeqEqual(tmpVector, result2)

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

       result0, result1, result2 = dmp.diff_linesToChars(lines, "")

       assert.Equal(t, chars, result0)
       assert.Equal(t, "", result1, "")
       assertSeqEqual(tmpVector, result2)
    */
}

func Test_diff_charsToLines(t *testing.T) {
    dmp := createDMP()
    // Convert chars up to lines.
    diffs := []change{
        change{DIFF_EQUAL, "\u0001\u0002\u0001"},
        change{DIFF_INSERT, "\u0002\u0001\u0002"}}

    tmpVector := []string{"", "alpha\n", "beta\n"}
    dmp.diff_charsToLines(diffs, tmpVector)
    assertSeqEqual([]change{
        change{DIFF_EQUAL, "alpha\nbeta\nalpha\n"},
        change{DIFF_INSERT, "beta\nalpha\nbeta\n"}}, diffs)

    // More than 256 to reveal any 8-bit limitations.
    n := 300
    tmpVector = []string{}
    lineList := []rune{}
    charList := []rune{}

    for x := 1; x < n+1; x++ {
        tmpVector = append(tmpVector, string(x)+"\n")
        lineList = append(lineList, rune(x), '\n')
        charList = append(charList, rune(x))
    }

    assert.Equal(t, n, len(tmpVector))
    assert.Equal(t, n, len(charList))

    tmpVector = append([]string{""}, tmpVector...)
    diffs = []change{change{DIFF_DELETE, string(charList)}}
    dmp.diff_charsToLines(diffs, tmpVector)
    assertSeqEqual([]change{
        change{DIFF_DELETE, string(lineList)}}, diffs)
}

func Test_diff_cleanupMerge(t *testing.T) {
    dmp := createDMP()
    // Cleanup a messy diff.
    // Null case.
    diffs := []change{}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{}, diffs)

    // No change case.
    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "b"}, change{DIFF_INSERT, "c"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "b"}, change{DIFF_INSERT, "c"}}, diffs)

    // Merge equalities.
    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_EQUAL, "b"}, change{DIFF_EQUAL, "c"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_EQUAL, "abc"}}, diffs)

    // Merge deletions.
    diffs = []change{change{DIFF_DELETE, "a"}, change{DIFF_DELETE, "b"}, change{DIFF_DELETE, "c"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_DELETE, "abc"}}, diffs)

    // Merge insertions.
    diffs = []change{change{DIFF_INSERT, "a"}, change{DIFF_INSERT, "b"}, change{DIFF_INSERT, "c"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_INSERT, "abc"}}, diffs)

    // Merge interweave.
    diffs = []change{change{DIFF_DELETE, "a"}, change{DIFF_INSERT, "b"}, change{DIFF_DELETE, "c"}, change{DIFF_INSERT, "d"}, change{DIFF_EQUAL, "e"}, change{DIFF_EQUAL, "f"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_DELETE, "ac"}, change{DIFF_INSERT, "bd"}, change{DIFF_EQUAL, "ef"}}, diffs)

    // Prefix and suffix detection.
    diffs = []change{change{DIFF_DELETE, "a"}, change{DIFF_INSERT, "abc"}, change{DIFF_DELETE, "dc"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "d"}, change{DIFF_INSERT, "b"}, change{DIFF_EQUAL, "c"}}, diffs)

    // Prefix and suffix detection with equalities.
    diffs = []change{change{DIFF_EQUAL, "x"}, change{DIFF_DELETE, "a"}, change{DIFF_INSERT, "abc"}, change{DIFF_DELETE, "dc"}, change{DIFF_EQUAL, "y"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_EQUAL, "xa"}, change{DIFF_DELETE, "d"}, change{DIFF_INSERT, "b"}, change{DIFF_EQUAL, "cy"}}, diffs)

    // Slide edit left.
    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_INSERT, "ba"}, change{DIFF_EQUAL, "c"}}
    dmp.diff_cleanupMerge(&diffs)
    fmt.Println("*****", diffs)
    assertSeqEqual([]change{change{DIFF_INSERT, "ab"}, change{DIFF_EQUAL, "ac"}}, diffs)

    // Slide edit right.
    diffs = []change{change{DIFF_EQUAL, "c"}, change{DIFF_INSERT, "ab"}, change{DIFF_EQUAL, "a"}}
    dmp.diff_cleanupMerge(&diffs)

    fmt.Println("*****", diffs)
    assertSeqEqual([]change{change{DIFF_EQUAL, "ca"}, change{DIFF_INSERT, "ba"}}, diffs)

    // Slide edit left recursive.
    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "b"}, change{DIFF_EQUAL, "c"}, change{DIFF_DELETE, "ac"}, change{DIFF_EQUAL, "x"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_DELETE, "abc"}, change{DIFF_EQUAL, "acx"}}, diffs)

    // Slide edit right recursive.
    diffs = []change{change{DIFF_EQUAL, "x"}, change{DIFF_DELETE, "ca"}, change{DIFF_EQUAL, "c"}, change{DIFF_DELETE, "b"}, change{DIFF_EQUAL, "a"}}
    dmp.diff_cleanupMerge(&diffs)
    assertSeqEqual([]change{change{DIFF_EQUAL, "xca"}, change{DIFF_DELETE, "cba"}}, diffs)
}

func Test_diff_cleanupSemanticLossless(t *testing.T) {
    dmp := createDMP()
    // Slide diffs to match logical boundaries.
    // Null case.
    diffs := []change{}
    dmp.diff_cleanupSemanticLossless(diffs)
    assertSeqEqual([]change{}, diffs)

    // Blank lines.
    diffs = []change{
        change{DIFF_EQUAL, "AAA\r\n\r\nBBB"},
        change{DIFF_INSERT, "\r\nDDD\r\n\r\nBBB"},
        change{DIFF_EQUAL, "\r\nEEE"},
    }

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_EQUAL, "AAA\r\n\r\n"},
        change{DIFF_INSERT, "BBB\r\nDDD\r\n\r\n"},
        change{DIFF_EQUAL, "BBB\r\nEEE"}}, diffs)

    // Line boundaries.
    diffs = []change{
        change{DIFF_EQUAL, "AAA\r\nBBB"},
        change{DIFF_INSERT, " DDD\r\nBBB"},
        change{DIFF_EQUAL, " EEE"}}

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_EQUAL, "AAA\r\n"},
        change{DIFF_INSERT, "BBB DDD\r\n"},
        change{DIFF_EQUAL, "BBB EEE"}}, diffs)

    // Word boundaries.
    diffs = []change{
        change{DIFF_EQUAL, "The c"},
        change{DIFF_INSERT, "ow and the c"},
        change{DIFF_EQUAL, "at."}}

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_EQUAL, "The "},
        change{DIFF_INSERT, "cow and the "},
        change{DIFF_EQUAL, "cat."}}, diffs)

    // Alphanumeric boundaries.
    diffs = []change{
        change{DIFF_EQUAL, "The-c"},
        change{DIFF_INSERT, "ow-and-the-c"},
        change{DIFF_EQUAL, "at."}}

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_EQUAL, "The-"},
        change{DIFF_INSERT, "cow-and-the-"},
        change{DIFF_EQUAL, "cat."}}, diffs)

    // Hitting the start.
    diffs = []change{
        change{DIFF_EQUAL, "a"},
        change{DIFF_DELETE, "a"},
        change{DIFF_EQUAL, "ax"}}

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_DELETE, "a"},
        change{DIFF_EQUAL, "aax"}}, diffs)

    // Hitting the end.
    diffs = []change{
        change{DIFF_EQUAL, "xa"},
        change{DIFF_DELETE, "a"},
        change{DIFF_EQUAL, "a"}}

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_EQUAL, "xaa"},
        change{DIFF_DELETE, "a"}}, diffs)

    // Sentence boundaries.
    diffs = []change{
        change{DIFF_EQUAL, "The xxx. The "},
        change{DIFF_INSERT, "zzz. The "},
        change{DIFF_EQUAL, "yyy."}}

    dmp.diff_cleanupSemanticLossless(diffs)

    assertSeqEqual([]change{
        change{DIFF_EQUAL, "The xxx."},
        change{DIFF_INSERT, " The zzz."},
        change{DIFF_EQUAL, " The yyy."}}, diffs)
}

func Test_diff_cleanupSemantic(t *testing.T) {
    dmp := createDMP()
    // Cleanup semantically trivial equalities.
    // Null case.
    diffs := []change{}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{}, diffs)

    // No elimination #1.
    diffs = []change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "cd"},
        change{DIFF_EQUAL, "12"},
        change{DIFF_DELETE, "e"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "cd"},
        change{DIFF_EQUAL, "12"},
        change{DIFF_DELETE, "e"}}, diffs)

    // No elimination #2.
    diffs = []change{
        change{DIFF_DELETE, "abc"},
        change{DIFF_INSERT, "ABC"},
        change{DIFF_EQUAL, "1234"},
        change{DIFF_DELETE, "wxyz"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abc"},
        change{DIFF_INSERT, "ABC"},
        change{DIFF_EQUAL, "1234"},
        change{DIFF_DELETE, "wxyz"}}, diffs)

    // Simple elimination.
    diffs = []change{
        change{DIFF_DELETE, "a"},
        change{DIFF_EQUAL, "b"},
        change{DIFF_DELETE, "c"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abc"},
        change{DIFF_INSERT, "b"}}, diffs)

    // Backpass elimination.
    diffs = []change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_EQUAL, "cd"},
        change{DIFF_DELETE, "e"},
        change{DIFF_EQUAL, "f"},
        change{DIFF_INSERT, "g"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abcdef"},
        change{DIFF_INSERT, "cdfg"}}, diffs)

    // Multiple eliminations.
    diffs = []change{
        change{DIFF_INSERT, "1"},
        change{DIFF_EQUAL, "A"},
        change{DIFF_DELETE, "B"},
        change{DIFF_INSERT, "2"},
        change{DIFF_EQUAL, "_"},
        change{DIFF_INSERT, "1"},
        change{DIFF_EQUAL, "A"},
        change{DIFF_DELETE, "B"},
        change{DIFF_INSERT, "2"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "AB_AB"},
        change{DIFF_INSERT, "1A2_1A2"}}, diffs)

    // Word boundaries.
    diffs = []change{
        change{DIFF_EQUAL, "The c"},
        change{DIFF_DELETE, "ow and the c"},
        change{DIFF_EQUAL, "at."}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_EQUAL, "The "},
        change{DIFF_DELETE, "cow and the "},
        change{DIFF_EQUAL, "cat."}}, diffs)

    // No overlap elimination.
    diffs = []change{
        change{DIFF_DELETE, "abcxx"},
        change{DIFF_INSERT, "xxdef"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abcxx"},
        change{DIFF_INSERT, "xxdef"}}, diffs)

    // Overlap elimination.
    diffs = []change{
        change{DIFF_DELETE, "abcxxx"},
        change{DIFF_INSERT, "xxxdef"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abc"},
        change{DIFF_EQUAL, "xxx"},
        change{DIFF_INSERT, "def"}}, diffs)

    // Reverse overlap elimination.
    diffs = []change{
        change{DIFF_DELETE, "xxxabc"},
        change{DIFF_INSERT, "defxxx"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_INSERT, "def"},
        change{DIFF_EQUAL, "xxx"},
        change{DIFF_DELETE, "abc"}}, diffs)

    // Two overlap eliminations.
    diffs = []change{
        change{DIFF_DELETE, "abcd1212"},
        change{DIFF_INSERT, "1212efghi"},
        change{DIFF_EQUAL, "----"},
        change{DIFF_DELETE, "A3"},
        change{DIFF_INSERT, "3BC"}}
    dmp.diff_cleanupSemantic(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abcd"},
        change{DIFF_EQUAL, "1212"},
        change{DIFF_INSERT, "efghi"},
        change{DIFF_EQUAL, "----"},
        change{DIFF_DELETE, "A"},
        change{DIFF_EQUAL, "3"},
        change{DIFF_INSERT, "BC"}}, diffs)
}

func Test_diff_cleanupEfficiency(t *testing.T) {
    dmp := createDMP()
    // Cleanup operationally trivial equalities.
    dmp.DiffEditCost = 4
    // Null case.
    diffs := []change{}
    dmp.diff_cleanupEfficiency(diffs)
    assertSeqEqual([]change{}, diffs)

    // No elimination.
    diffs = []change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "12"},
        change{DIFF_EQUAL, "wxyz"},
        change{DIFF_DELETE, "cd"},
        change{DIFF_INSERT, "34"}}
    dmp.diff_cleanupEfficiency(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "12"},
        change{DIFF_EQUAL, "wxyz"},
        change{DIFF_DELETE, "cd"},
        change{DIFF_INSERT, "34"}}, diffs)

    // Four-edit elimination.
    diffs = []change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "12"},
        change{DIFF_EQUAL, "xyz"},
        change{DIFF_DELETE, "cd"},
        change{DIFF_INSERT, "34"}}
    dmp.diff_cleanupEfficiency(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abxyzcd"},
        change{DIFF_INSERT, "12xyz34"}}, diffs)

    // Three-edit elimination.
    diffs = []change{
        change{DIFF_INSERT, "12"},
        change{DIFF_EQUAL, "x"},
        change{DIFF_DELETE, "cd"},
        change{DIFF_INSERT, "34"}}
    dmp.diff_cleanupEfficiency(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "xcd"},
        change{DIFF_INSERT, "12x34"}}, diffs)

    // Backpass elimination.
    diffs = []change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "12"},
        change{DIFF_EQUAL, "xy"},
        change{DIFF_INSERT, "34"},
        change{DIFF_EQUAL, "z"},
        change{DIFF_DELETE, "cd"},
        change{DIFF_INSERT, "56"}}
    dmp.diff_cleanupEfficiency(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abxyzcd"},
        change{DIFF_INSERT, "12xy34z56"}}, diffs)

    // High cost elimination.
    dmp.DiffEditCost = 5
    diffs = []change{
        change{DIFF_DELETE, "ab"},
        change{DIFF_INSERT, "12"},
        change{DIFF_EQUAL, "wxyz"},
        change{DIFF_DELETE, "cd"},
        change{DIFF_INSERT, "34"}}
    dmp.diff_cleanupEfficiency(diffs)
    assertSeqEqual([]change{
        change{DIFF_DELETE, "abwxyzcd"},
        change{DIFF_INSERT, "12wxyz34"}}, diffs)
    dmp.DiffEditCost = 4
}

/*
func Test_diff_prettyHtml(t *testing.T) {
    dmp := createDMP()
    // Pretty print.
    diffs := []change{
        change{DIFF_EQUAL, "a\n"},
        change{DIFF_DELETE, "<B>b</B>"},
        change{DIFF_INSERT, "c&d"}}
    assert.Equal(t, "<span>a&para;<br></span><del style=\"background:#ffe6e6;\">&lt;B&gt;b&lt;/B&gt;</del><ins style=\"background:#e6ffe6;\">c&amp;d</ins>",
        dmp.diff_prettyHtml(diffs))
}*/

func Test_diff_text(t *testing.T) {
    dmp := createDMP()
    // Compute the source and destination texts.
    diffs := []change{
        change{DIFF_EQUAL, "jump"},
        change{DIFF_DELETE, "s"},
        change{DIFF_INSERT, "ed"},
        change{DIFF_EQUAL, " over "},
        change{DIFF_DELETE, "the"},
        change{DIFF_INSERT, "a"},
        change{DIFF_EQUAL, " lazy"}}
    assert.Equal(t, "jumps over the lazy", dmp.diff_text1(diffs))
    assert.Equal(t, "jumped over a lazy", dmp.diff_text2(diffs))
}

func Test_diff_delta(t *testing.T) {
    dmp := createDMP()
    // Convert a diff into delta string.
    diffs := []change{
        change{DIFF_EQUAL, "jump"},
        change{DIFF_DELETE, "s"},
        change{DIFF_INSERT, "ed"},
        change{DIFF_EQUAL, " over "},
        change{DIFF_DELETE, "the"},
        change{DIFF_INSERT, "a"},
        change{DIFF_EQUAL, " lazy"},
        change{DIFF_INSERT, "old dog"}}

    text1 := dmp.diff_text1(diffs)
    assert.Equal(t, "jumps over the lazy", text1)

    delta := dmp.diff_toDelta(diffs)
    assert.Equal(t, "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", delta)

    // Convert delta string into a diff.
    assertSeqEqual(diffs, dmp.diff_fromDelta(text1, delta))

    // Generates error (19 < 20).
    //try {
    dmp.diff_fromDelta(text1+"x", delta)
    //assert.Fail("diff_fromDelta: Too long.");
    /*
       } catch (ArgumentException) {
         // Exception expected.
       }
    */

    // Generates error (19 > 18).
    //try {
    dmp.diff_fromDelta(text1[1:], delta)
    //  assert.Fail("diff_fromDelta: Too short.");
    //} catch (ArgumentException) {
    // Exception expected.
    //}

    // Generates error (%c3%xy invalid Unicode).
    //try {
    dmp.diff_fromDelta("", "+%c3%xy")
    //assert.Fail("diff_fromDelta: Invalid character.");
    //} catch (ArgumentException) {
    // Exception expected.
    //}

    // Test deltas with special characters.
    zero := "0"
    one := "1"
    two := "2"
    diffs = []change{
        change{DIFF_EQUAL, "\u0680 " + zero + " \t %"},
        change{DIFF_DELETE, "\u0681 " + one + " \n ^"},
        change{DIFF_INSERT, "\u0682 " + two + " \\ |"}}
    text1 = dmp.diff_text1(diffs)
    assert.Equal(t, "\u0680 "+zero+" \t %\u0681 "+one+" \n ^", text1)

    delta = dmp.diff_toDelta(diffs)
    // Lowercase, due to UrlEncode uses lower.
    assert.Equal(t, "=7\t-7\t+%da%82 %02 %5c %7c", delta)

    assertSeqEqual(diffs, dmp.diff_fromDelta(text1, delta))

    // Verify pool of unchanged characters.
    diffs = []change{
        change{DIFF_INSERT, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # "}}
    text2 := dmp.diff_text2(diffs)
    assert.Equal(t, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # ", text2, "diff_text2: Unchanged characters.")

    delta = dmp.diff_toDelta(diffs)
    assert.Equal(t, "+A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # ", delta, "diff_toDelta: Unchanged characters.")

    // Convert delta string into a diff.
    assertSeqEqual(diffs, dmp.diff_fromDelta("", delta))
}

func Test_diff_xIndex(t *testing.T) {
    dmp := createDMP()
    // Translate a location in text1 to text2.
    diffs := []change{
        change{DIFF_DELETE, "a"},
        change{DIFF_INSERT, "1234"},
        change{DIFF_EQUAL, "xyz"}}
    assert.Equal(t, 5, dmp.diff_xIndex(diffs, 2), "diff_xIndex: Translation on equality.")

    diffs = []change{
        change{DIFF_EQUAL, "a"},
        change{DIFF_DELETE, "1234"},
        change{DIFF_EQUAL, "xyz"}}
    assert.Equal(t, 1, dmp.diff_xIndex(diffs, 3), "diff_xIndex: Translation on deletion.")
}

func Test_diff_levenshtein(t *testing.T) {
    dmp := createDMP()
    diffs := []change{
        change{DIFF_DELETE, "abc"},
        change{DIFF_INSERT, "1234"},
        change{DIFF_EQUAL, "xyz"}}
    assert.Equal(t, 4, dmp.diff_levenshtein(diffs), "diff_levenshtein: Levenshtein with trailing equality.")

    diffs = []change{
        change{DIFF_EQUAL, "xyz"},
        change{DIFF_DELETE, "abc"},
        change{DIFF_INSERT, "1234"}}
    assert.Equal(t, 4, dmp.diff_levenshtein(diffs), "diff_levenshtein: Levenshtein with leading equality.")

    diffs = []change{
        change{DIFF_DELETE, "abc"},
        change{DIFF_EQUAL, "xyz"},
        change{DIFF_INSERT, "1234"}}
    assert.Equal(t, 7, dmp.diff_levenshtein(diffs), "diff_levenshtein: Levenshtein with middle equality.")
}

/*
func Test_diff_bisect(t *testing.T) {
    dmp := createDMP()
    // Normal.
    a := "cat"
    b := "map"
    // Since the resulting diff hasn't been normalized, it would be ok if
    // the insertion and deletion pairs are swapped.
    // If the order changes, tweak this test as required.
    diffs := []change{change{DIFF_DELETE, "c"}, change{DIFF_INSERT, "m"}, change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "t"}, change{DIFF_INSERT, "p"}}
    assertSeqEqual(diffs, dmp.diff_bisect(a, b, DateTime.MaxValue)) TODO

    // Timeout.
    diffs = []change{change{DIFF_DELETE, "cat"}, change{DIFF_INSERT, "map"}}
    assertSeqEqual(diffs, dmp.diff_bisect(a, b, DateTime.MinValue)) TODO
}
*/

func Test_diff_main(t *testing.T) {
    dmp := createDMP()
    // Perform a trivial diff.
    diffs := []change{}
    assertSeqEqual(diffs, dmp.diff_main("", "", false))

    diffs = []change{change{DIFF_EQUAL, "abc"}}
    assertSeqEqual(diffs, dmp.diff_main("abc", "abc", false))

    diffs = []change{change{DIFF_EQUAL, "ab"}, change{DIFF_INSERT, "123"}, change{DIFF_EQUAL, "c"}}
    assertSeqEqual(diffs, dmp.diff_main("abc", "ab123c", false))

    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "123"}, change{DIFF_EQUAL, "bc"}}
    assertSeqEqual(diffs, dmp.diff_main("a123bc", "abc", false))

    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_INSERT, "123"}, change{DIFF_EQUAL, "b"}, change{DIFF_INSERT, "456"}, change{DIFF_EQUAL, "c"}}
    assertSeqEqual(diffs, dmp.diff_main("abc", "a123b456c", false))

    diffs = []change{change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "123"}, change{DIFF_EQUAL, "b"}, change{DIFF_DELETE, "456"}, change{DIFF_EQUAL, "c"}}
    assertSeqEqual(diffs, dmp.diff_main("a123b456c", "abc", false))

    // Perform a real diff.
    // Switch off the timeout.
    dmp.DiffTimeout = 0
    diffs = []change{change{DIFF_DELETE, "a"}, change{DIFF_INSERT, "b"}}
    assertSeqEqual(diffs, dmp.diff_main("a", "b", false))

    diffs = []change{change{DIFF_DELETE, "Apple"}, change{DIFF_INSERT, "Banana"}, change{DIFF_EQUAL, "s are a"}, change{DIFF_INSERT, "lso"}, change{DIFF_EQUAL, " fruit."}}
    assertSeqEqual(diffs, dmp.diff_main("Apples are a fruit.", "Bananas are also fruit.", false))

    diffs = []change{change{DIFF_DELETE, "a"}, change{DIFF_INSERT, "\u0680"}, change{DIFF_EQUAL, "x"}, change{DIFF_DELETE, "\t"}, change{DIFF_INSERT, "0"}}
    assertSeqEqual(diffs, dmp.diff_main("ax\t", "\u0680x"+"0", false))

    diffs = []change{change{DIFF_DELETE, "1"}, change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "y"}, change{DIFF_EQUAL, "b"}, change{DIFF_DELETE, "2"}, change{DIFF_INSERT, "xab"}}
    assertSeqEqual(diffs, dmp.diff_main("1ayb2", "abxab", false))

    diffs = []change{change{DIFF_INSERT, "xaxcx"}, change{DIFF_EQUAL, "abc"}, change{DIFF_DELETE, "y"}}
    assertSeqEqual(diffs, dmp.diff_main("abcy", "xaxcxabc", false))

    diffs = []change{change{DIFF_DELETE, "ABCD"}, change{DIFF_EQUAL, "a"}, change{DIFF_DELETE, "="}, change{DIFF_INSERT, "-"}, change{DIFF_EQUAL, "bcd"}, change{DIFF_DELETE, "="}, change{DIFF_INSERT, "-"}, change{DIFF_EQUAL, "efghijklmnopqrs"}, change{DIFF_DELETE, "EFGHIJKLMNOefg"}}
    assertSeqEqual(diffs, dmp.diff_main("ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg", "a-bcd-efghijklmnopqrs", false))

    diffs = []change{change{DIFF_INSERT, " "}, change{DIFF_EQUAL, "a"}, change{DIFF_INSERT, "nd"}, change{DIFF_EQUAL, " [[Pennsylvania]]"}, change{DIFF_DELETE, " and [[New"}}
    assertSeqEqual(diffs, dmp.diff_main("a [[Pennsylvania]] and [[New", " and [[Pennsylvania]]", false))

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
    dmp.diff_main(a, b)
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
    assertSeqEqual(dmp.diff_main(a, b, true), dmp.diff_main(a, b, false))

    a = "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
    b = "abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghijabcdefghij"
    assertSeqEqual(dmp.diff_main(a, b, true), dmp.diff_main(a, b, false))

    a = "1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n1234567890\n"
    b = "abcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n1234567890\n1234567890\n1234567890\nabcdefghij\n"
    texts_linemode := diff_rebuildtexts(dmp.diff_main(a, b, true))
    texts_textmode := diff_rebuildtexts(dmp.diff_main(a, b, false))
    assertSeqEqual(texts_textmode, texts_linemode)

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
    assertSeqEqual(bitmask, dmp.match_alphabet("abc"))

    bitmask = map[byte]int{
        'a': 37,
        'b': 18,
        'c': 8,
    }
    assertSeqEqual(bitmask, dmp.match_alphabet("abcaba"))
}

func Test_match_bitap(t *testing.T) {
    dmp := createDMP()

    // Bitap algorithm.
    dmp.MatchDistance = 100
    dmp.MatchThreshold = 0.5
    assert.Equal(t, 5, dmp.match_bitap("abcdefghijk", "fgh", 5), "match_bitap: Exact match #1.")

    assert.Equal(t, 5, dmp.match_bitap("abcdefghijk", "fgh", 0), "match_bitap: Exact match #2.")

    assert.Equal(t, 4, dmp.match_bitap("abcdefghijk", "efxhi", 0), "match_bitap: Fuzzy match #1.")

    assert.Equal(t, 2, dmp.match_bitap("abcdefghijk", "cdefxyhijk", 5), "match_bitap: Fuzzy match #2.")

    assert.Equal(t, -1, dmp.match_bitap("abcdefghijk", "bxy", 1), "match_bitap: Fuzzy match #3.")

    assert.Equal(t, 2, dmp.match_bitap("123456789xx0", "3456789x0", 2), "match_bitap: Overflow.")

    assert.Equal(t, 0, dmp.match_bitap("abcdef", "xxabc", 4), "match_bitap: Before start match.")

    assert.Equal(t, 3, dmp.match_bitap("abcdef", "defyy", 4), "match_bitap: Beyond end match.")

    assert.Equal(t, 0, dmp.match_bitap("abcdef", "xabcdefy", 0), "match_bitap: Oversized pattern.")

    dmp.MatchThreshold = 0.4
    assert.Equal(t, 4, dmp.match_bitap("abcdefghijk", "efxyhi", 1), "match_bitap: Threshold #1.")

    dmp.MatchThreshold = 0.3
    assert.Equal(t, -1, dmp.match_bitap("abcdefghijk", "efxyhi", 1), "match_bitap: Threshold #2.")

    dmp.MatchThreshold = 0.0
    assert.Equal(t, 1, dmp.match_bitap("abcdefghijk", "bcdef", 1), "match_bitap: Threshold #3.")

    dmp.MatchThreshold = 0.5
    assert.Equal(t, 0, dmp.match_bitap("abcdexyzabcde", "abccde", 3), "match_bitap: Multiple select #1.")

    assert.Equal(t, 8, dmp.match_bitap("abcdexyzabcde", "abccde", 5), "match_bitap: Multiple select #2.")

    dmp.MatchDistance = 10 // Strict location.
    assert.Equal(t, -1, dmp.match_bitap("abcdefghijklmnopqrstuvwxyz", "abcdefg", 24), "match_bitap: Distance test #1.")

    assert.Equal(t, 0, dmp.match_bitap("abcdefghijklmnopqrstuvwxyz", "abcdxxefg", 1), "match_bitap: Distance test #2.")

    dmp.MatchDistance = 1000 // Loose location.
    assert.Equal(t, 0, dmp.match_bitap("abcdefghijklmnopqrstuvwxyz", "abcdefg", 24), "match_bitap: Distance test #3.")
}

func Test_match_main(t *testing.T) {
    dmp := createDMP()
    // Full match.
    assert.Equal(t, 0, dmp.match_main("abcdef", "abcdef", 1000), "match_main: Equality.")

    assert.Equal(t, -1, dmp.match_main("", "abcdef", 1), "match_main: Null text.")

    assert.Equal(t, 3, dmp.match_main("abcdef", "", 3), "match_main: Null pattern.")

    assert.Equal(t, 3, dmp.match_main("abcdef", "de", 3), "match_main: Exact match.")

    assert.Equal(t, 3, dmp.match_main("abcdef", "defy", 4), "match_main: Beyond end match.")

    assert.Equal(t, 0, dmp.match_main("abcdef", "abcdefy", 0), "match_main: Oversized pattern.")

    dmp.MatchThreshold = 0.7
    assert.Equal(t, 4, dmp.match_main("I am the very model of a modern major general.", " that berry ", 5), "match_main: Complex match.")
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
    p.diffs = []change{
        change{DIFF_EQUAL, "jump"},
        change{DIFF_DELETE, "s"},
        change{DIFF_INSERT, "ed"},
        change{DIFF_EQUAL, " over "},
        change{DIFF_DELETE, "the"},
        change{DIFF_INSERT, "a"},
        change{DIFF_EQUAL, "\nlaz"}}
    strp := "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n %0alaz\n"
    assert.Equal(t, strp, p.String(), "Patch: toString.")
}

func Test_patch_fromText(t *testing.T) {
    dmp := createDMP()
    softAssert(t, len(dmp.patch_fromText("")) == 0, "patch_fromText: #0.")

    strp := "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n %0alaz\n"
    assert.Equal(t, strp, dmp.patch_fromText(strp)[0].String(), "patch_fromText: #1.")

    assert.Equal(t, "@@ -1 +1 @@\n-a\n+b\n", dmp.patch_fromText("@@ -1 +1 @@\n-a\n+b\n")[0].String(), "patch_fromText: #2.")

    assert.Equal(t, "@@ -1,3 +0,0 @@\n-abc\n", dmp.patch_fromText("@@ -1,3 +0,0 @@\n-abc\n")[0].String(), "patch_fromText: #3.")

    assert.Equal(t, "@@ -0,0 +1,3 @@\n+abc\n", dmp.patch_fromText("@@ -0,0 +1,3 @@\n+abc\n")[0].String(), "patch_fromText: #4.")

    // Generates error.
    //try {
    //dmp.patch_fromText("Bad\nPatch\n");
    //assert.Fail("patch_fromText: #5.");
    //} catch (ArgumentException) {
    // Exception expected.
    //  }
}

func Test_patch_toText(t *testing.T) {
    dmp := createDMP()
    strp := "@@ -21,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"
    var patches []Patch
    patches = dmp.patch_fromText(strp)
    result := dmp.patch_toText(patches)
    assert.Equal(t, strp, result)

    strp = "@@ -1,9 +1,9 @@\n-f\n+F\n oo+fooba\n@@ -7,9 +7,9 @@\n obar\n-,\n+.\n  tes\n"
    patches = dmp.patch_fromText(strp)
    result = dmp.patch_toText(patches)
    assert.Equal(t, strp, result)
}

func Test_patch_addContext(t *testing.T) {
    dmp := createDMP()
    dmp.PatchMargin = 4
    var p Patch
    p = dmp.patch_fromText("@@ -21,4 +21,10 @@\n-jump\n+somersault\n")[0]
    dmp.patch_addContext(p, "The quick brown fox jumps over the lazy dog.")
    assert.Equal(t, "@@ -17,12 +17,18 @@\n fox \n-jump\n+somersault\n s ov\n", p.String(), "patch_addContext: Simple case.")

    p = dmp.patch_fromText("@@ -21,4 +21,10 @@\n-jump\n+somersault\n")[0]
    dmp.patch_addContext(p, "The quick brown fox jumps.")
    assert.Equal(t, "@@ -17,10 +17,16 @@\n fox \n-jump\n+somersault\n s.\n", p.String(), "patch_addContext: Not enough trailing context.")

    p = dmp.patch_fromText("@@ -3 +3,2 @@\n-e\n+at\n")[0]
    dmp.patch_addContext(p, "The quick brown fox jumps.")
    assert.Equal(t, "@@ -1,7 +1,8 @@\n Th\n-e\n+at\n  qui\n", p.String(), "patch_addContext: Not enough leading context.")

    p = dmp.patch_fromText("@@ -3 +3,2 @@\n-e\n+at\n")[0]
    dmp.patch_addContext(p, "The quick brown fox jumps.  The quick brown fox crashes.")
    assert.Equal(t, "@@ -1,27 +1,28 @@\n Th\n-e\n+at\n  quick brown fox jumps. \n", p.String(), "patch_addContext: Ambiguity.")
}

func Test_patch_make(t *testing.T) {
    dmp := createDMP()
    var patches []Patch
    patches = dmp.patch_make("", "")
    assert.Equal(t, "", dmp.patch_toText(patches), "patch_make: Null case.")

    text1 := "The quick brown fox jumps over the lazy dog."
    text2 := "That quick brown fox jumped over a lazy dog."
    expectedPatch := "@@ -1,8 +1,7 @@\n Th\n-at\n+e\n  qui\n@@ -21,17 +21,18 @@\n jump\n-ed\n+s\n  over \n-a\n+the\n  laz\n"
    // The second patch must be "-21,17 +21,18", not "-22,17 +21,18" due to rolling context.
    patches = dmp.patch_make(text2, text1)
    assert.Equal(t, expectedPatch, dmp.patch_toText(patches), "patch_make: Text2+Text1 inputs.")

    expectedPatch = "@@ -1,11 +1,12 @@\n Th\n-e\n+at\n  quick b\n@@ -22,18 +22,17 @@\n jump\n-s\n+ed\n  over \n-the\n+a\n  laz\n"
    patches = dmp.patch_make(text1, text2)
    assert.Equal(t, expectedPatch, dmp.patch_toText(patches), "patch_make: Text1+Text2 inputs.")

    diffs := dmp.diff_main(text1, text2, false)
    patches = dmp.patch_make(diffs)
    assert.Equal(t, expectedPatch, dmp.patch_toText(patches), "patch_make: Diff input.")

    patches = dmp.patch_make(text1, diffs)
    assert.Equal(t, expectedPatch, dmp.patch_toText(patches), "patch_make: Text1+Diff inputs.")

    patches = dmp.patch_make(text1, text2, diffs)
    assert.Equal(t, expectedPatch, dmp.patch_toText(patches), "patch_make: Text1+Text2+Diff inputs (deprecated).")

    patches = dmp.patch_make("`1234567890-=[]\\;',./", "~!@#$%^&*()_+{}|:\"<>?")
    assert.Equal(t, "@@ -1,21 +1,21 @@\n-%601234567890-=%5b%5d%5c;',./\n+~!@#$%25%5e&*()_+%7b%7d%7c:%22%3c%3e?\n",
        dmp.patch_toText(patches),
        "patch_toText: Character encoding.")

    diffs = []change{
        change{DIFF_DELETE, "`1234567890-=[]\\;',./"},
        change{DIFF_INSERT, "~!@#$%^&*()_+{}|:\"<>?"}}
    assertSeqEqual(diffs,
        dmp.patch_fromText("@@ -1,21 +1,21 @@\n-%601234567890-=%5B%5D%5C;',./\n+~!@#$%25%5E&*()_+%7B%7D%7C:%22%3C%3E?\n")[0].diffs,
    )

    text1 = ""
    for x := 0; x < 100; x++ {
        text1 += "abcdef"
    }
    text2 = text1 + "123"
    expectedPatch = "@@ -573,28 +573,31 @@\n cdefabcdefabcdefabcdefabcdef\n+123\n"
    patches = dmp.patch_make(text1, text2)
    assert.Equal(t, expectedPatch, dmp.patch_toText(patches), "patch_make: Long string with repeats.")

    // Test null inputs -- not needed because nulls can't be passed in C#.
}

func Test_patch_splitMax(t *testing.T) {
    // Assumes that Match_MaxBits is 32.
    dmp := createDMP()
    var patches []Patch

    patches = dmp.patch_make("abcdefghijklmnopqrstuvwxyz01234567890", "XabXcdXefXghXijXklXmnXopXqrXstXuvXwxXyzX01X23X45X67X89X0")
    dmp.patch_splitMax(patches)
    assert.Equal(t, "@@ -1,32 +1,46 @@\n+X\n ab\n+X\n cd\n+X\n ef\n+X\n gh\n+X\n ij\n+X\n kl\n+X\n mn\n+X\n op\n+X\n qr\n+X\n st\n+X\n uv\n+X\n wx\n+X\n yz\n+X\n 012345\n@@ -25,13 +39,18 @@\n zX01\n+X\n 23\n+X\n 45\n+X\n 67\n+X\n 89\n+X\n 0\n", dmp.patch_toText(patches))

    patches = dmp.patch_make("abcdef1234567890123456789012345678901234567890123456789012345678901234567890uvwxyz", "abcdefuvwxyz")
    oldToText := dmp.patch_toText(patches)
    dmp.patch_splitMax(patches)
    assert.Equal(t, oldToText, dmp.patch_toText(patches))

    patches = dmp.patch_make("1234567890123456789012345678901234567890123456789012345678901234567890", "abc")
    dmp.patch_splitMax(patches)
    assert.Equal(t, "@@ -1,32 +1,4 @@\n-1234567890123456789012345678\n 9012\n@@ -29,32 +1,4 @@\n-9012345678901234567890123456\n 7890\n@@ -57,14 +1,3 @@\n-78901234567890\n+abc\n", dmp.patch_toText(patches))

    patches = dmp.patch_make("abcdefghij , h : 0 , t : 1 abcdefghij , h : 0 , t : 1 abcdefghij , h : 0 , t : 1", "abcdefghij , h : 1 , t : 1 abcdefghij , h : 1 , t : 1 abcdefghij , h : 0 , t : 1")
    dmp.patch_splitMax(patches)
    assert.Equal(t, "@@ -2,32 +2,32 @@\n bcdefghij , h : \n-0\n+1\n  , t : 1 abcdef\n@@ -29,32 +29,32 @@\n bcdefghij , h : \n-0\n+1\n  , t : 1 abcdef\n", dmp.patch_toText(patches))
}

func Test_patch_addPadding(t *testing.T) {
    dmp := createDMP()
    var patches []Patch
    patches = dmp.patch_make("", "test")
    assert.Equal(t, "@@ -0,0 +1,4 @@\n+test\n",
        dmp.patch_toText(patches),
        "patch_addPadding: Both edges full.")
    dmp.patch_addPadding(patches)
    assert.Equal(t, "@@ -1,8 +1,12 @@\n %01%02%03%04\n+test\n %01%02%03%04\n",
        dmp.patch_toText(patches),
        "patch_addPadding: Both edges full.")

    patches = dmp.patch_make("XY", "XtestY")
    assert.Equal(t, "@@ -1,2 +1,6 @@\n X\n+test\n Y\n",
        dmp.patch_toText(patches),
        "patch_addPadding: Both edges partial.")
    dmp.patch_addPadding(patches)
    assert.Equal(t, "@@ -2,8 +2,12 @@\n %02%03%04X\n+test\n Y%01%02%03\n",
        dmp.patch_toText(patches),
        "patch_addPadding: Both edges partial.")

    patches = dmp.patch_make("XXXXYYYY", "XXXXtestYYYY")
    assert.Equal(t, "@@ -1,8 +1,12 @@\n XXXX\n+test\n YYYY\n",
        dmp.patch_toText(patches),
        "patch_addPadding: Both edges none.")
    dmp.patch_addPadding(patches)
    assert.Equal(t, "@@ -5,8 +5,12 @@\n XXXX\n+test\n YYYY\n",
        dmp.patch_toText(patches),
        "patch_addPadding: Both edges none.")
}

func Test_patch_apply(t *testing.T) {
    dmp := createDMP()
    dmp.MatchDistance = 1000
    dmp.MatchThreshold = 0.5
    dmp.PatchDeleteThreshold = 0.5
    patches := []Patch{}
    patches = dmp.patch_make("", "")
    results0, results1 := dmp.patch_apply(patches, "Hello world.")
    boolArray := results1
    resultStr := results0 + "\t" + string(len(boolArray))
    assert.Equal(t, "Hello world.\t0", resultStr, "patch_apply: Null case.")

    patches = dmp.patch_make("The quick brown fox jumps over the lazy dog.", "That quick brown fox jumped over a lazy dog.")
    results0, results1 = dmp.patch_apply(patches, "The quick brown fox jumps over the lazy dog.")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "That quick brown fox jumped over a lazy dog.\tTrue\tTrue", resultStr, "patch_apply: Exact match.")

    results0, results1 = dmp.patch_apply(patches, "The quick red rabbit jumps over the tired tiger.")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "That quick red rabbit jumped over a tired tiger.\tTrue\tTrue", resultStr, "patch_apply: Partial match.")

    results0, results1 = dmp.patch_apply(patches, "I am the very model of a modern major general.")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "I am the very model of a modern major general.\tFalse\tFalse", resultStr, "patch_apply: Failed match.")

    patches = dmp.patch_make("x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy")
    results0, results1 = dmp.patch_apply(patches, "x123456789012345678901234567890-----++++++++++-----123456789012345678901234567890y")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "xabcy\tTrue\tTrue", resultStr, "patch_apply: Big delete, small change.")

    patches = dmp.patch_make("x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy")
    results0, results1 = dmp.patch_apply(patches, "x12345678901234567890---------------++++++++++---------------12345678901234567890y")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "xabc12345678901234567890---------------++++++++++---------------12345678901234567890y\tFalse\tTrue", resultStr, "patch_apply: Big delete, big change 1.")

    dmp.PatchDeleteThreshold = 0.6
    patches = dmp.patch_make("x1234567890123456789012345678901234567890123456789012345678901234567890y", "xabcy")
    results0, results1 = dmp.patch_apply(patches, "x12345678901234567890---------------++++++++++---------------12345678901234567890y")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "xabcy\tTrue\tTrue", resultStr, "patch_apply: Big delete, big change 2.")
    dmp.PatchDeleteThreshold = 0.5

    dmp.MatchThreshold = 0.0
    dmp.MatchDistance = 0
    patches = dmp.patch_make("abcdefghijklmnopqrstuvwxyz--------------------1234567890", "abcXXXXXXXXXXdefghijklmnopqrstuvwxyz--------------------1234567YYYYYYYYYY890")
    results0, results1 = dmp.patch_apply(patches, "ABCDEFGHIJKLMNOPQRSTUVWXYZ--------------------1234567890")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0]) + "\t" + strconv.FormatBool(boolArray[1])
    assert.Equal(t, "ABCDEFGHIJKLMNOPQRSTUVWXYZ--------------------1234567YYYYYYYYYY890\tFalse\tTrue", resultStr, "patch_apply: Compensate for failed patch.")
    dmp.MatchThreshold = 0.5
    dmp.MatchDistance = 1000

    patches = dmp.patch_make("", "test")
    patchStr := dmp.patch_toText(patches)
    dmp.patch_apply(patches, "")
    assert.Equal(t, patchStr, dmp.patch_toText(patches), "patch_apply: No side effects.")

    patches = dmp.patch_make("The quick brown fox jumps over the lazy dog.", "Woof")
    patchStr = dmp.patch_toText(patches)
    dmp.patch_apply(patches, "The quick brown fox jumps over the lazy dog.")
    assert.Equal(t, patchStr, dmp.patch_toText(patches), "patch_apply: No side effects with major delete.")

    patches = dmp.patch_make("", "test")
    results0, results1 = dmp.patch_apply(patches, "")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0])
    assert.Equal(t, "test\tTrue", resultStr, "patch_apply: Edge exact match.")

    patches = dmp.patch_make("XY", "XtestY")
    results0, results1 = dmp.patch_apply(patches, "XY")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0])
    assert.Equal(t, "XtestY\tTrue", resultStr, "patch_apply: Near edge exact match.")

    patches = dmp.patch_make("y", "y123")
    results0, results1 = dmp.patch_apply(patches, "x")
    boolArray = results1
    resultStr = results0 + "\t" + strconv.FormatBool(boolArray[0])
    assert.Equal(t, "x123\tTrue", resultStr, "patch_apply: Edge partial match.")
}
