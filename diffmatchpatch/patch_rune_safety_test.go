package diffmatchpatch

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// alignRuneStart should back any byte index inside a multi-byte UTF-8
// character back to that character's first byte.
func TestAlignRuneStart(t *testing.T) {
	// "abc中文def" — UTF-8: a=1 b=1 c=1 中=3(byte 3..5) 文=3(6..8) d=1 e=1 f=1
	s := "abc中文def"
	cases := []struct {
		in, want int
		desc     string
	}{
		{0, 0, "0 unchanged"},
		{1, 1, "after 'a' already aligned"},
		{3, 3, "start of 中 aligned"},
		{4, 3, "middle of 中 backs to start"},
		{5, 3, "end-middle of 中 backs to start"},
		{6, 6, "start of 文 aligned"},
		{7, 6, "middle of 文 backs to start"},
		{9, 9, "after 文 = start of d, aligned"},
		{12, 12, "end-of-string"},
		{-1, -1, "negative unchanged"},
		{999, 999, "past end unchanged"},
	}
	for _, c := range cases {
		got := alignRuneStart(s, c.in)
		if got != c.want {
			t.Errorf("alignRuneStart(%q, %d) = %d, want %d  (%s)", s, c.in, got, c.want, c.desc)
		}
	}
}

// alignRuneEnd should move any byte index inside a multi-byte UTF-8
// character forward to the start of the next character.
func TestAlignRuneEnd(t *testing.T) {
	s := "abc中文def"
	cases := []struct {
		in, want int
		desc     string
	}{
		{0, 0, "0 unchanged"},
		{3, 3, "start of 中 aligned"},
		{4, 6, "middle of 中 moves to start of 文"},
		{5, 6, "end-middle of 中 moves to start of 文"},
		{7, 9, "middle of 文 moves to start of d"},
		{12, 12, "end-of-string"},
		{-1, -1, "negative unchanged"},
		{999, 999, "past end unchanged"},
	}
	for _, c := range cases {
		got := alignRuneEnd(s, c.in)
		if got != c.want {
			t.Errorf("alignRuneEnd(%q, %d) = %d, want %d  (%s)", s, c.in, got, c.want, c.desc)
		}
	}
}

// A replaced text over 32 bytes takes the long-delete path even when it is
// only 20 characters. text2 must keep its last rune there, or the trailing
// 日 is not deleted while the patch still reports success.
func TestPatchApplyMultibyteMonsterDelete(t *testing.T) {
	a := "a日aaaaéaaaabaééaé日a日"
	b := "a日aaabaaa日aaaaaaaaéaaaé"
	dmp := New()
	got, applied := dmp.PatchApply(dmp.PatchMake(a, b), a)
	assert.Equal(t, []bool{true}, applied)
	assert.Equal(t, b, got)
}

// The end context added by PatchMake must keep a whole rune when the margin
// ends inside one. A shortened context made PatchApply pad the patch as if it
// touched the end of the text, so the insertion landed one rune early.
func TestPatchMakeApplyContextEndsInsideRune(t *testing.T) {
	a := " aba— 🚀"
	b := " a aba—a 🚀"
	dmp := New()
	got, applied := dmp.PatchApply(dmp.PatchMake(a, b), a)
	assert.Equal(t, []bool{true}, applied)
	assert.Equal(t, b, got)
}

// An insertion that lands in the padding must not leave part of a rune behind
// when the padding is stripped.
func TestPatchApplyStripPaddingKeepsValidUTF8(t *testing.T) {
	dmp := New()
	patches := dmp.PatchMake("日bxa a", "b xa —")
	got, _ := dmp.PatchApply(patches, "日bxa")
	assert.True(t, isValidUTF8(got), "got %q", got)
}

// When a patch is applied to a text that has changed, earlier edits can leave
// the text shorter than the indices computed for later ones. PatchApply must
// clamp those indices instead of panicking.
func TestPatchApplyChangedTextNoPanic(t *testing.T) {
	dmp := New()
	for _, c := range []struct{ a, b, target string }{
		{" a—ab中文—中文中文", "", " a—ab中文—"},
		{"日日 日ééa中文", "a中文 x", "日日a日é"},
	} {
		assert.NotPanics(t, func() {
			got, _ := dmp.PatchApply(dmp.PatchMake(c.a, c.b), c.target)
			assert.True(t, isValidUTF8(got), "got %q", got)
		}, "a=%q b=%q target=%q", c.a, c.b, c.target)
	}
}

// The end context of a split patch must not end inside a multi-byte rune.
func TestDiffText1PrefixRuneAligned(t *testing.T) {
	diffs := []Diff{{DiffEqual, "ab"}, {DiffInsert, "xyz"}, {DiffDelete, "中文"}, {DiffEqual, "cd"}}
	for n, want := range map[int]string{0: "", 2: "ab", 3: "ab", 4: "ab", 5: "ab中", 7: "ab中", 8: "ab中文", 9: "ab中文c", 20: "ab中文cd"} {
		assert.Equal(t, want, diffText1Prefix(diffs, n), "n=%d", n)
	}
	assert.Equal(t, "", diffText1Prefix([]Diff{{DiffEqual, "中"}}, 2))
}

// Issue #132 minimal repro: applying a patch containing a multi-byte
// character with a short anchor used to panic with "slice bounds out of range".
// After the rune-safe alignment, PatchApply should return cleanly (apply may
// fail with results[i]=false, but must never panic).
func TestPatchApply_Issue132_NoPanic(t *testing.T) {
	dmp := New()
	patches, err := dmp.PatchFromText("@@ -1,2 +1,3 @@\n %E2%98%9E \n+r\n")
	if err != nil {
		t.Fatalf("PatchFromText: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic leaked: %v", r)
		}
	}()
	_, results := dmp.PatchApply(patches, "☞ 𝗢𝗥𝗗𝗘𝗥 ")
	if results == nil {
		t.Fatalf("results nil")
	}
}

// PatchSplitMax on a CJK-heavy base must produce diff.Text slices that are
// all valid UTF-8 (prior to this fix, byte-level slicing of multi-byte
// characters could leave continuation-byte prefixes in the output).
func TestPatchSplitMax_NoInvalidUTF8(t *testing.T) {
	dmp := New()
	// CJK base larger than MatchMaxBits (32) forces PatchSplitMax to engage.
	old := "中文段落一" + strings.Repeat("汉字", 30) + "结束"
	newText := "中文段落一" + strings.Repeat("汉字", 30) + "结尾"
	patches := dmp.PatchMake(old, dmp.DiffMain(old, newText, false))
	if len(patches) == 0 {
		t.Skip("no patches generated")
	}
	// PatchMake itself must produce valid-UTF-8 diff text (PatchAddContext's
	// prefix/suffix slice now aligns to rune boundaries).
	for i, p := range patches {
		for j, d := range p.diffs {
			if !isValidUTF8(d.Text) {
				t.Errorf("PatchMake patch %d diff %d invalid UTF-8: %q", i, j, d.Text)
			}
		}
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PatchSplitMax panic: %v", r)
		}
	}()
	// PatchSplitMax must preserve valid UTF-8 (precontext / postcontext /
	// diffText cut points all rune-aligned).
	for i, p := range dmp.PatchSplitMax(patches) {
		for j, d := range p.diffs {
			if !isValidUTF8(d.Text) {
				t.Errorf("PatchSplitMax patch %d diff %d invalid UTF-8: %q  type=%d", i, j, d.Text, d.Type)
			}
		}
	}
}

// minimal UTF-8 validity check inlined to avoid adding a stdlib import vs.
// the existing import block in patch.go (utf8 is already there for the fix).
func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		r, sz := utf8DecodeFirst(s[i:])
		if r == 0xFFFD && sz == 1 {
			return false
		}
		i += sz
	}
	return true
}

func utf8DecodeFirst(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	b := s[0]
	switch {
	case b < 0x80:
		return rune(b), 1
	case b < 0xc0:
		return 0xFFFD, 1
	case b < 0xe0:
		if len(s) < 2 {
			return 0xFFFD, 1
		}
		return rune(b&0x1f)<<6 | rune(s[1]&0x3f), 2
	case b < 0xf0:
		if len(s) < 3 {
			return 0xFFFD, 1
		}
		return rune(b&0x0f)<<12 | rune(s[1]&0x3f)<<6 | rune(s[2]&0x3f), 3
	default:
		if len(s) < 4 {
			return 0xFFFD, 1
		}
		return rune(b&0x07)<<18 | rune(s[1]&0x3f)<<12 | rune(s[2]&0x3f)<<6 | rune(s[3]&0x3f), 4
	}
}
