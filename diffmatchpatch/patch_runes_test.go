package diffmatchpatch

import (
	"strings"
	"testing"
)

// PatchApplyRunes 在含中文 base 上的端到端验证: PatchMake 生成的 patch 经 PatchToText
// → PatchFromText round-trip 后,通过 PatchApplyRunes 应字节级还原 target。
// 这条路径在 byte-based PatchApply 上由于 PatchSplitMax 把 CJK 段切碎 + Bitap
// alphabet 误把 UTF-8 continuation byte 当独立 char,fuzzy 容错弱,极端 case 422。
func TestPatchApplyRunes_RoundTripCJK(t *testing.T) {
	dmp := New()
	old := "中文段落" + strings.Repeat("汉字", 30) + "结束"
	new := "中文段落" + strings.Repeat("汉字", 30) + "新结尾添加更多内容"

	patches := dmp.PatchMake(old, dmp.DiffMain(old, new, false))
	patchText := dmp.PatchToText(patches)
	parsed, err := dmp.PatchFromText(patchText)
	if err != nil {
		t.Fatalf("PatchFromText: %v", err)
	}

	gotRunes, applied := dmp.PatchApplyRunes(parsed, []rune(old))
	for i, ok := range applied {
		if !ok {
			t.Errorf("patch %d apply failed", i)
		}
	}
	got := string(gotRunes)
	if got != new {
		t.Errorf("rune-apply mismatch:\n  got:  %q\n  want: %q", got, new)
	}
}

// 纯 ASCII 场景: PatchApplyRunes 应该跟 PatchApply (byte-mode) 字节级等价。
func TestPatchApplyRunes_ASCII_EquivToByteApply(t *testing.T) {
	dmp := New()
	old := strings.Repeat("the quick brown fox jumps over the lazy dog.\n", 5)
	new := strings.Repeat("the SLOW brown fox JUMPS over the lazy dog.\n", 5)
	patches := dmp.PatchMake(old, dmp.DiffMain(old, new, false))

	gotByte, _ := dmp.PatchApply(patches, old)
	gotRunes, _ := dmp.PatchApplyRunes(patches, []rune(old))
	if string(gotRunes) != gotByte {
		t.Errorf("ASCII path divergence:\n  byte: %q\n  rune: %q", gotByte, string(gotRunes))
	}
	if string(gotRunes) != new {
		t.Errorf("rune-apply mismatch:\n  got:  %q\n  want: %q", string(gotRunes), new)
	}
}

// MatchMainRunes 单元: rune index 命中 (跟 MatchMain byte index 命中对照)。
func TestMatchMainRunes_BasicCJK(t *testing.T) {
	dmp := New()
	text := []rune("前缀文本 abc 中文锚点 def 后缀")
	pattern := []rune("中文锚点")
	loc := 10 // rune index, near where the pattern is
	got := dmp.MatchMainRunes(text, pattern, loc)
	// 期望命中 "中文锚点" 在 text 中的 rune index
	want := -1
	for i := 0; i+len(pattern) <= len(text); i++ {
		match := true
		for j := range pattern {
			if text[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			want = i
			break
		}
	}
	if got != want {
		t.Errorf("MatchMainRunes = %d, want %d", got, want)
	}
}

// DiffXIndexRunes 单元: 计 rune index 不是 byte index。
func TestDiffXIndexRunes_CountsRunes(t *testing.T) {
	dmp := New()
	// 用 DiffMain 构造一个 diffs 列表 (含中文)
	diffs := dmp.DiffMain("中文 hello", "中文 world", false)
	// loc=3 (rune index past "中文 ") 应映射到 target 同位置 (相同 prefix)
	got := dmp.DiffXIndexRunes(diffs, 3)
	if got != 3 {
		t.Errorf("DiffXIndexRunes(loc=3) = %d, want 3 (rune-aligned prefix length)", got)
	}
}
