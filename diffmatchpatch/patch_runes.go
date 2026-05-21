// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
//
// Rune-based Patch helpers — mirror Patch{Apply,SplitMax,AddPadding} and
// supporting Diff helpers but count length / slice by rune index instead of
// UTF-8 byte. See match_runes.go for motivation.
//
// API surface:
//   - PatchApplyRunes(patches, text) ([]rune, []bool) — new public entry
//     point. Callers convert their string to []rune before the call and
//     string() the returned []rune. Patch.Start1/Start2/Length1/Length2 are
//     interpreted as rune counts (matching the reference JS implementation's
//     UTF-16 unit semantics for BMP characters).
//   - The original PatchApply (byte-based) is unchanged.

package diffmatchpatch

import (
	"math"
	"unicode/utf8"
)

// runesEqualString — runes vs string equality, avoids materialising one side.
// Used in perfect-match shortcut in PatchApplyRunes (compare patch-source-text
// with extracted base segment).
func runesEqualString(rs []rune, s string) bool {
	if len(s) != utf8.RuneCountInString(s) && len(rs) != utf8.RuneCountInString(s) {
		// short-circuit length mismatch via rune count
	}
	i := 0
	for _, r := range s {
		if i >= len(rs) || rs[i] != r {
			return false
		}
		i++
	}
	return i == len(rs)
}

// DiffText1Runes returns the source text of diffs as a rune slice.
// (sergi's DiffText1 returns string concatenation by byte; we keep rune slices
// to avoid byte/rune conversions in hot loops.)
func (dmp *DiffMatchPatch) DiffText1Runes(diffs []Diff) []rune {
	out := []rune{}
	for _, d := range diffs {
		if d.Type != DiffInsert {
			out = append(out, []rune(d.Text)...)
		}
	}
	return out
}

func (dmp *DiffMatchPatch) DiffText2Runes(diffs []Diff) []rune {
	out := []rune{}
	for _, d := range diffs {
		if d.Type != DiffDelete {
			out = append(out, []rune(d.Text)...)
		}
	}
	return out
}

// DiffXIndexRunes — rune-mirror of DiffXIndex. loc is rune index in text1
// (concatenation of equality+deletion diffs), returns rune index in text2.
func (dmp *DiffMatchPatch) DiffXIndexRunes(diffs []Diff, loc int) int {
	chars1 := 0
	chars2 := 0
	lastChars1 := 0
	lastChars2 := 0
	var lastDiff Diff
	for _, aDiff := range diffs {
		runeLen := utf8.RuneCountInString(aDiff.Text)
		if aDiff.Type != DiffInsert {
			chars1 += runeLen
		}
		if aDiff.Type != DiffDelete {
			chars2 += runeLen
		}
		if chars1 > loc {
			lastDiff = aDiff
			break
		}
		lastChars1 = chars1
		lastChars2 = chars2
	}
	if lastDiff.Type == DiffDelete {
		return lastChars2
	}
	return lastChars2 + (loc - lastChars1)
}

// PatchAddPaddingRunes — rune-mirror of PatchAddPadding. Returns padding as
// []rune (each ASCII control char 0x01..0x04 is 1 rune == 1 UTF-16 unit).
// All patches' Start1/Start2 and length fields are bumped in rune units.
func (dmp *DiffMatchPatch) PatchAddPaddingRunes(patches []Patch) []rune {
	paddingLength := dmp.PatchMargin
	nullPadding := make([]rune, 0, paddingLength)
	for x := 1; x <= paddingLength; x++ {
		nullPadding = append(nullPadding, rune(x))
	}

	for i := range patches {
		patches[i].Start1 += paddingLength
		patches[i].Start2 += paddingLength
	}

	// Add some padding on start of first diff.
	if len(patches[0].diffs) == 0 || patches[0].diffs[0].Type != DiffEqual {
		patches[0].diffs = append([]Diff{{DiffEqual, string(nullPadding)}}, patches[0].diffs...)
		patches[0].Start1 -= paddingLength
		patches[0].Start2 -= paddingLength
		patches[0].Length1 += paddingLength
		patches[0].Length2 += paddingLength
	} else {
		firstRunes := utf8.RuneCountInString(patches[0].diffs[0].Text)
		if paddingLength > firstRunes {
			extra := paddingLength - firstRunes
			patches[0].diffs[0].Text = string(nullPadding[firstRunes:]) + patches[0].diffs[0].Text
			patches[0].Start1 -= extra
			patches[0].Start2 -= extra
			patches[0].Length1 += extra
			patches[0].Length2 += extra
		}
	}

	last := len(patches) - 1
	if len(patches[last].diffs) == 0 || patches[last].diffs[len(patches[last].diffs)-1].Type != DiffEqual {
		patches[last].diffs = append(patches[last].diffs, Diff{DiffEqual, string(nullPadding)})
		patches[last].Length1 += paddingLength
		patches[last].Length2 += paddingLength
	} else {
		lastDiff := patches[last].diffs[len(patches[last].diffs)-1]
		lastRunes := utf8.RuneCountInString(lastDiff.Text)
		if paddingLength > lastRunes {
			extra := paddingLength - lastRunes
			patches[last].diffs[len(patches[last].diffs)-1].Text += string(nullPadding[:extra])
			patches[last].Length1 += extra
			patches[last].Length2 += extra
		}
	}

	return nullPadding
}

// PatchSplitMaxRunes — rune-mirror of PatchSplitMax. patch_size /
// Patch_Margin compared against rune count, diffText sliced by rune index.
// This is the key fix vs sergi byte-mode: a 30-CJK-char patch is no longer
// chopped into ~9-char tiny segments (which weakens Bitap pattern strength).
func (dmp *DiffMatchPatch) PatchSplitMaxRunes(patches []Patch) []Patch {
	patchSize := dmp.MatchMaxBits
	for x := 0; x < len(patches); x++ {
		if patches[x].Length1 <= patchSize {
			continue
		}
		bigpatch := patches[x]
		patches = append(patches[:x], patches[x+1:]...)
		x--

		Start1 := bigpatch.Start1
		Start2 := bigpatch.Start2
		precontext := []rune{}
		for len(bigpatch.diffs) != 0 {
			patch := Patch{}
			empty := true
			patch.Start1 = Start1 - len(precontext)
			patch.Start2 = Start2 - len(precontext)
			if len(precontext) != 0 {
				patch.Length1 = len(precontext)
				patch.Length2 = len(precontext)
				patch.diffs = append(patch.diffs, Diff{DiffEqual, string(precontext)})
			}
			for len(bigpatch.diffs) != 0 && patch.Length1 < patchSize-dmp.PatchMargin {
				diffType := bigpatch.diffs[0].Type
				diffRunes := []rune(bigpatch.diffs[0].Text)
				diffRuneLen := len(diffRunes)
				if diffType == DiffInsert {
					patch.Length2 += diffRuneLen
					Start2 += diffRuneLen
					patch.diffs = append(patch.diffs, bigpatch.diffs[0])
					bigpatch.diffs = bigpatch.diffs[1:]
					empty = false
				} else if diffType == DiffDelete && len(patch.diffs) == 1 && patch.diffs[0].Type == DiffEqual && diffRuneLen > 2*patchSize {
					patch.Length1 += diffRuneLen
					Start1 += diffRuneLen
					empty = false
					patch.diffs = append(patch.diffs, Diff{diffType, bigpatch.diffs[0].Text})
					bigpatch.diffs = bigpatch.diffs[1:]
				} else {
					cutAt := patchSize - patch.Length1 - dmp.PatchMargin
					if cutAt > diffRuneLen {
						cutAt = diffRuneLen
					}
					if cutAt < 0 {
						cutAt = 0
					}
					// loop-progress invariant: cutAt == 0 时强制吃 1 rune
					if cutAt == 0 && diffRuneLen > 0 {
						cutAt = 1
					}
					taken := diffRunes[:cutAt]
					takenStr := string(taken)
					takenRuneLen := len(taken)

					patch.Length1 += takenRuneLen
					Start1 += takenRuneLen
					if diffType == DiffEqual {
						patch.Length2 += takenRuneLen
						Start2 += takenRuneLen
					} else {
						empty = false
					}
					patch.diffs = append(patch.diffs, Diff{diffType, takenStr})
					if takenRuneLen == diffRuneLen {
						bigpatch.diffs = bigpatch.diffs[1:]
					} else {
						bigpatch.diffs[0].Text = string(diffRunes[cutAt:])
					}
				}
			}

			precontext = dmp.DiffText2Runes(patch.diffs)
			if len(precontext) > dmp.PatchMargin {
				precontext = precontext[len(precontext)-dmp.PatchMargin:]
			}

			rawPost := dmp.DiffText1Runes(bigpatch.diffs)
			var postcontext []rune
			if len(rawPost) > dmp.PatchMargin {
				postcontext = rawPost[:dmp.PatchMargin]
			} else {
				postcontext = rawPost
			}
			if len(postcontext) != 0 {
				patch.Length1 += len(postcontext)
				patch.Length2 += len(postcontext)
				if len(patch.diffs) != 0 && patch.diffs[len(patch.diffs)-1].Type == DiffEqual {
					patch.diffs[len(patch.diffs)-1].Text += string(postcontext)
				} else {
					patch.diffs = append(patch.diffs, Diff{DiffEqual, string(postcontext)})
				}
			}
			if !empty {
				x++
				patches = append(patches[:x], append([]Patch{patch}, patches[x:]...)...)
			}
		}
	}
	return patches
}

// PatchApplyRunes — rune-mirror of PatchApply. text is rune slice;
// Patch.Start1/Start2/Length1/Length2 treated as rune counts (== UTF-16 unit
// for BMP chars, matching Google JS DMP wire format).
//
// Behavior parity with PatchApply preserved at structure level. The
// difference is that all length comparisons, slice indices, MatchMain calls,
// and substring extractions operate in rune space, eliminating the byte /
// UTF-16-unit mismatch that causes silent fuzzy drift on multi-byte text.
func (dmp *DiffMatchPatch) PatchApplyRunes(patches []Patch, text []rune) ([]rune, []bool) {
	if len(patches) == 0 {
		return text, []bool{}
	}

	patches = dmp.PatchDeepCopy(patches)

	nullPadding := dmp.PatchAddPaddingRunes(patches)
	padded := make([]rune, 0, len(text)+2*len(nullPadding))
	padded = append(padded, nullPadding...)
	padded = append(padded, text...)
	padded = append(padded, nullPadding...)
	text = padded

	patches = dmp.PatchSplitMaxRunes(patches)

	x := 0
	delta := 0
	results := make([]bool, len(patches))
	for _, aPatch := range patches {
		expectedLoc := aPatch.Start2 + delta
		text1 := dmp.DiffText1Runes(aPatch.diffs)
		var startLoc int
		endLoc := -1
		if len(text1) > dmp.MatchMaxBits {
			startLoc = dmp.MatchMainRunes(text, text1[:dmp.MatchMaxBits], expectedLoc)
			if startLoc != -1 {
				endLoc = dmp.MatchMainRunes(text,
					text1[len(text1)-dmp.MatchMaxBits:], expectedLoc+len(text1)-dmp.MatchMaxBits)
				if endLoc == -1 || startLoc >= endLoc {
					startLoc = -1
				}
			}
		} else {
			startLoc = dmp.MatchMainRunes(text, text1, expectedLoc)
		}
		if startLoc == -1 {
			results[x] = false
			delta -= aPatch.Length2 - aPatch.Length1
		} else {
			results[x] = true
			delta = startLoc - expectedLoc
			var text2 []rune
			if endLoc == -1 {
				end := int(math.Min(float64(startLoc+len(text1)), float64(len(text))))
				text2 = text[startLoc:end]
			} else {
				end := int(math.Min(float64(endLoc+dmp.MatchMaxBits), float64(len(text))))
				text2 = text[startLoc:end]
			}
			if runesEqual(text1, text2) {
				replacement := dmp.DiffText2Runes(aPatch.diffs)
				newText := make([]rune, 0, len(text)+len(replacement)-len(text1))
				newText = append(newText, text[:startLoc]...)
				newText = append(newText, replacement...)
				newText = append(newText, text[startLoc+len(text1):]...)
				text = newText
			} else {
				diffs := dmp.DiffMain(string(text1), string(text2), false)
				if len(text1) > dmp.MatchMaxBits && float64(dmp.DiffLevenshtein(diffs))/float64(len(text1)) > dmp.PatchDeleteThreshold {
					results[x] = false
				} else {
					diffs = dmp.DiffCleanupSemanticLossless(diffs)
					index1 := 0
					for _, aDiff := range aPatch.diffs {
						if aDiff.Type != DiffEqual {
							index2 := dmp.DiffXIndexRunes(diffs, index1)
							if aDiff.Type == DiffInsert {
								insertRunes := []rune(aDiff.Text)
								insertAt := startLoc + index2
								if insertAt < 0 {
									insertAt = 0
								}
								if insertAt > len(text) {
									insertAt = len(text)
								}
								newText := make([]rune, 0, len(text)+len(insertRunes))
								newText = append(newText, text[:insertAt]...)
								newText = append(newText, insertRunes...)
								newText = append(newText, text[insertAt:]...)
								text = newText
							} else if aDiff.Type == DiffDelete {
								delLen := utf8.RuneCountInString(aDiff.Text)
								startIdx := startLoc + index2
								endIdx := startLoc + dmp.DiffXIndexRunes(diffs, index1+delLen)
								if startIdx < 0 {
									startIdx = 0
								}
								if endIdx > len(text) {
									endIdx = len(text)
								}
								if endIdx < startIdx {
									endIdx = startIdx
								}
								newText := make([]rune, 0, len(text)-(endIdx-startIdx))
								newText = append(newText, text[:startIdx]...)
								newText = append(newText, text[endIdx:]...)
								text = newText
							}
						}
						if aDiff.Type != DiffDelete {
							index1 += utf8.RuneCountInString(aDiff.Text)
						}
					}
				}
			}
		}
		x++
	}
	// strip padding
	if len(text) >= 2*len(nullPadding) {
		text = text[len(nullPadding) : len(text)-len(nullPadding)]
	}
	return text, results
}
