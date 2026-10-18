// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
//
// Rune-based Match helpers — mirror MatchMain / MatchBitap / MatchAlphabet but
// operate on []rune instead of UTF-8 bytes. Lengths and positions are counted
// in runes (one Unicode code point each), which matches the reference JS
// diff-match-patch's UTF-16 code unit semantics for all BMP characters.
//
// Motivation: the existing byte-based Bitap on multi-byte text (CJK, em dash,
// emoji) uses len(text) (byte count), patches[].Length1 (byte count), and
// substring slicing by byte index. The reference JS implementation uses
// UTF-16 unit counts everywhere. When Go callers receive patch text emitted
// by a JS counterpart (whose Length/Start numbers are UTF-16 units), the
// byte-based Go path produces drift:
//   - PatchSplitMax cuts CJK segments ~3x more finely (90 bytes vs 28 units),
//     shortening Bitap patterns and weakening fuzzy match accuracy.
//   - expected_loc = Start2 + delta drifts by N bytes for N multi-byte chars
//     preceding the patch — fuzzy match's MatchDistance window may no longer
//     cover the real anchor location.
//   - MatchAlphabet keyed by byte conflates the three bytes of each CJK rune,
//     scrambling the Bitap alphabet on dense CJK text.
//
// These functions provide a parallel rune-based path. Algorithm structure is
// preserved 1:1; only the length / indexing model differs.

package diffmatchpatch

import "math"

// MatchMainRunes — rune-mirror of MatchMain. text and pattern are []rune.
// loc / return value are rune indices into text.
func (dmp *DiffMatchPatch) MatchMainRunes(text, pattern []rune, loc int) int {
	loc = int(math.Max(0, math.Min(float64(loc), float64(len(text)))))
	if runesEqual(text, pattern) {
		return 0
	} else if len(text) == 0 {
		return -1
	} else if loc+len(pattern) <= len(text) && runesEqual(text[loc:loc+len(pattern)], pattern) {
		return loc
	}
	return dmp.MatchBitapRunes(text, pattern, loc)
}

// MatchBitapRunes — rune-mirror of MatchBitap.
func (dmp *DiffMatchPatch) MatchBitapRunes(text, pattern []rune, loc int) int {
	s := dmp.MatchAlphabetRunes(pattern)

	scoreThreshold := dmp.MatchThreshold
	bestLoc := runesIndexOf(text, pattern, loc)
	if bestLoc != -1 {
		scoreThreshold = math.Min(dmp.matchBitapScoreRunes(0, bestLoc, loc, pattern), scoreThreshold)
		bestLoc = runesLastIndexBefore(text, pattern, loc+len(pattern))
		if bestLoc != -1 {
			scoreThreshold = math.Min(dmp.matchBitapScoreRunes(0, bestLoc, loc, pattern), scoreThreshold)
		}
	}

	matchmask := 1 << uint(len(pattern)-1)
	bestLoc = -1

	var binMin, binMid int
	binMax := len(pattern) + len(text)
	lastRd := []int{}
	for d := 0; d < len(pattern); d++ {
		binMin = 0
		binMid = binMax
		for binMin < binMid {
			if dmp.matchBitapScoreRunes(d, loc+binMid, loc, pattern) <= scoreThreshold {
				binMin = binMid
			} else {
				binMax = binMid
			}
			binMid = (binMax-binMin)/2 + binMin
		}
		binMax = binMid
		start := int(math.Max(1, float64(loc-binMid+1)))
		finish := int(math.Min(float64(loc+binMid), float64(len(text))) + float64(len(pattern)))

		rd := make([]int, finish+2)
		rd[finish+1] = (1 << uint(d)) - 1

		for j := finish; j >= start; j-- {
			var charMatch int
			if len(text) <= j-1 {
				charMatch = 0
			} else if _, ok := s[text[j-1]]; !ok {
				charMatch = 0
			} else {
				charMatch = s[text[j-1]]
			}

			if d == 0 {
				rd[j] = ((rd[j+1] << 1) | 1) & charMatch
			} else {
				rd[j] = ((rd[j+1]<<1)|1)&charMatch | (((lastRd[j+1] | lastRd[j]) << 1) | 1) | lastRd[j+1]
			}
			if (rd[j] & matchmask) != 0 {
				score := dmp.matchBitapScoreRunes(d, j-1, loc, pattern)
				if score <= scoreThreshold {
					scoreThreshold = score
					bestLoc = j - 1
					if bestLoc > loc {
						start = int(math.Max(1, float64(2*loc-bestLoc)))
					} else {
						break
					}
				}
			}
		}
		if dmp.matchBitapScoreRunes(d+1, loc, loc, pattern) > scoreThreshold {
			break
		}
		lastRd = rd
	}
	return bestLoc
}

func (dmp *DiffMatchPatch) matchBitapScoreRunes(e, x, loc int, pattern []rune) float64 {
	accuracy := float64(e) / float64(len(pattern))
	proximity := math.Abs(float64(loc - x))
	if dmp.MatchDistance == 0 {
		if proximity == 0 {
			return accuracy
		}
		return 1.0
	}
	return accuracy + (proximity / float64(dmp.MatchDistance))
}

// MatchAlphabetRunes — rune-mirror of MatchAlphabet. key is rune (not byte),
// so multi-byte chars get their own alphabet entry instead of conflating with
// their continuation bytes (the latter making sergi byte-Alphabet broken on CJK).
func (dmp *DiffMatchPatch) MatchAlphabetRunes(pattern []rune) map[rune]int {
	s := map[rune]int{}
	for _, c := range pattern {
		if _, ok := s[c]; !ok {
			s[c] = 0
		}
	}
	i := 0
	for _, c := range pattern {
		s[c] |= int(uint(1) << uint(len(pattern)-i-1))
		i++
	}
	return s
}

// runesLastIndexBefore — rune-version of lastIndexOf (last occurrence of pattern
// in target ending at or before beforeLoc). sergi has runesIndex/runesIndexOf
// already; this completes the trio.
func runesLastIndexBefore(target, pattern []rune, beforeLoc int) int {
	if len(pattern) == 0 {
		if beforeLoc > len(target) {
			return len(target)
		}
		return beforeLoc
	}
	maxStart := beforeLoc - len(pattern)
	if maxStart > len(target)-len(pattern) {
		maxStart = len(target) - len(pattern)
	}
	for i := maxStart; i >= 0; i-- {
		if runesEqual(target[i:i+len(pattern)], pattern) {
			return i
		}
	}
	return -1
}
