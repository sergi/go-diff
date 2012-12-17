package godiff

import (
    "fmt"
    "strings"
    "testing" //import go package for testing related functionality
    "unicode/utf8"
    //"strconv"

)

func assert(cond bool, msg string) {
    if !cond {
        print("assertion fail: ", msg, "\n")
        panic(1)
    }
}

func assertSeqStrEqual(seq1 []string, seq2 []string, msg string) {
    var fail = func() {
        fmt.Println("assertion fail: ", msg, "\n")
        panic(1)
    }

    if len(seq1) != len(seq2) {
        fail()
    }

    for i, _ := range seq1 {
        if seq1[i] != seq2[i] {
            fail()
        }
    }
}

func Test_diff_commonPrefix(t *testing.T) {
    dmp := createDMP()
    // Detect any common suffix.
    // Null case.
    assert(0 == dmp.diff_commonPrefix("abc", "xyz"), "'abc' and 'xyz' should not be equal")

    // Non-null case.
    assert(4 == dmp.diff_commonPrefix("1234abcdef", "1234xyz"), "")

    // Whole case.
    assert(4 == dmp.diff_commonPrefix("1234", "1234xyz"), "")

    t.Log("commonPrefix test passed.")
}

func Test_diff_commonSuffixTest(t *testing.T) {
    dmp := createDMP()
    // Detect any common suffix.
    // Null case.
    assert(0 == dmp.diff_commonSuffix("abc", "xyz"), "")

    // Non-null case.
    assert(4 == dmp.diff_commonSuffix("abcdef1234", "xyz1234"), "")

    // Whole case.
    assert(4 == dmp.diff_commonSuffix("1234", "xyz1234"), "")
    t.Log("commonSuffix test passed.")
}

func Test_diff_commonOverlapTest(t *testing.T) {
    dmp := createDMP()
    // Detect any suffix/prefix overlap.
    // Null case.
    assert(0 == dmp.diff_commonOverlap("", "abcd"), "")

    // Whole case.
    assert(3 == dmp.diff_commonOverlap("abc", "abcd"), "")

    // No overlap.
    assert(0 == dmp.diff_commonOverlap("123456", "abcd"), "")

    // Overlap.
    assert(3 == dmp.diff_commonOverlap("123456xxx", "xxxabcd"), "")

    // Unicode.
    // Some overly clever languages (C#) may treat ligatures as equal to their
    // component letters.  E.g. U+FB01 == 'fi'
    assert(0 == dmp.diff_commonOverlap("fi", "\ufb01i"), "")
}

func Test_diff_halfmatchTest(t *testing.T) {
    dmp := createDMP()
    dmp.DiffTimeout = 1
    // No match.
    assert(dmp.diff_halfMatch("1234567890", "abcdef") == nil, "")

    assert(dmp.diff_halfMatch("12345", "23") == nil, "")

    // Single Match.
    assertSeqStrEqual(
        []string{"12", "90", "a", "z", "345678"},
        dmp.diff_halfMatch("1234567890", "a345678z"), "")

    assertSeqStrEqual([]string{"a", "z", "12", "90", "345678"}, dmp.diff_halfMatch("a345678z", "1234567890"), "")

    assertSeqStrEqual([]string{"abc", "z", "1234", "0", "56789"}, dmp.diff_halfMatch("abc56789z", "1234567890"), "")

    assertSeqStrEqual([]string{"a", "xyz", "1", "7890", "23456"}, dmp.diff_halfMatch("a23456xyz", "1234567890"), "")

    // Multiple Matches.
    assertSeqStrEqual([]string{"12123", "123121", "a", "z", "1234123451234"}, dmp.diff_halfMatch("121231234123451234123121", "a1234123451234z"), "")

    assertSeqStrEqual([]string{"", "-=-=-=-=-=", "x", "", "x-=-=-=-=-=-=-="}, dmp.diff_halfMatch("x-=-=-=-=-=-=-=-=-=-=-=-=", "xx-=-=-=-=-=-=-="), "")

    assertSeqStrEqual([]string{"-=-=-=-=-=", "", "", "y", "-=-=-=-=-=-=-=y"}, dmp.diff_halfMatch("-=-=-=-=-=-=-=-=-=-=-=-=y", "-=-=-=-=-=-=-=yy"), "")

    // Non-optimal halfmatch.
    // Optimal diff would be -q+x=H-i+e=lloHe+Hu=llo-Hew+y not -qHillo+x=HelloHe-w+Hulloy
    assertSeqStrEqual([]string{"qHillo", "w", "x", "Hulloy", "HelloHe"}, dmp.diff_halfMatch("qHilloHelloHew", "xHelloHeHulloy"), "")

    // Optimal no halfmatch.
    dmp.DiffTimeout = 0
    assert(dmp.diff_halfMatch("qHilloHelloHew", "xHelloHeHulloy") == nil, "")
}

func Test_diff_linesToCharsTest(t *testing.T) {
    dmp := createDMP()
    // Convert lines down to characters.
    tmpVector := []string{"", "alpha\n", "beta\n"}

    result0, result1, result2 := dmp.diff_linesToChars("alpha\nbeta\nalpha\n", "beta\nalpha\nbeta\n")
    assert("\u0001\u0002\u0001" == result0, "")
    assert("\u0002\u0001\u0002" == result1, "")
    assertSeqStrEqual(tmpVector, result2, "")

    tmpVector = []string{"", "alpha\r\n", "beta\r\n", "\r\n"}
    result0, result1, result2 = dmp.diff_linesToChars("", "alpha\r\nbeta\r\n\r\n\r\n")
    assert("" == result0, "")
    assert("\u0001\u0002\u0003\u0003" == result1, "")
    assertSeqStrEqual(tmpVector, result2, "")

    tmpVector = []string{"", "a", "b"}
    result0, result1, result2 = dmp.diff_linesToChars("a", "b")
    assert("\u0001" == result0, "")
    assert("\u0002" == result1, "")
    assertSeqStrEqual(tmpVector, result2, "")

    // More than 256 to reveal any 8-bit limitations.
    n := 300
    tmpVector = []string{}
    lineList := []string{}
    charList := []rune{}

    for x := 1; x < n+1; x++ {
        tmpVector = append(tmpVector, string(x)+"\n")
        lineList = append(lineList, string(x)+"\n")
        charList = append(charList, rune(x))
    }

    assert(n == len(tmpVector), "")

    lines := strings.Join(lineList, "")
    chars := string(charList)

    assert(n == utf8.RuneCountInString(chars), "")
    tmpVector = append(tmpVector, "")

    result0, result1, result2 = dmp.diff_linesToChars(lines, "")

    assert(chars == result0, "")
    assert("" == result1, "")
    assertSeqStrEqual(tmpVector, result2, "")
}
