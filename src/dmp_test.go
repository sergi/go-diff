package godiff

import (
    "fmt"
    "github.com/bmizerany/assert"
    //"strings"
    "reflect"
    "testing" //import go package for testing related functionality
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
    assertSeqEqual([]change{change{DIFF_INSERT, "ab"}, change{DIFF_EQUAL, "ac"}}, diffs)

    // Slide edit right.
    diffs = []change{change{DIFF_EQUAL, "c"}, change{DIFF_INSERT, "ab"}, change{DIFF_EQUAL, "a"}}
    dmp.diff_cleanupMerge(&diffs)

    fmt.Println(diffs)
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
