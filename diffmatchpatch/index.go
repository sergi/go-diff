package diffmatchpatch

import "fmt"

type index uint32

const runeSkipStart = 0xd800
const runeSkipEnd = 0xdfff + 1
const runeMax = 0x110000 // next invalid code point

func stringToIndex(text string) []index {
	runes := []rune(text)
	indexes := make([]index, len(runes))
	for i, r := range runes {
		if r < runeSkipEnd {
			indexes[i] = index(r)
		} else {
			indexes[i] = index(r) - (runeSkipEnd - runeSkipStart)
		}
	}
	return indexes
}

func indexesToRunes(indexes []index) []rune {
	runes := make([]rune, len(indexes))
	for i, idx := range indexes {
		if idx >= runeMax-(runeSkipEnd-runeSkipStart) {
			panic(fmt.Sprintf("The index %d is too large for indexesToRunes()", idx))
		}
		if idx < runeSkipStart {
			runes[i] = rune(idx)
		} else {
			runes[i] = rune(idx + (runeSkipEnd - runeSkipStart))
		}
	}
	return runes
}

func indexesToString(indexes []index) string {
	return string(indexesToRunes(indexes))
}
