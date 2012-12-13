package godiff

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DiffMatchPatch struct {
	DiffTimeout          int32
	DiffEditCost         int32
	MatchDistance        int32
	PatchDeleteThreshold float32
	PatchMargin          int32
	MatchMaxBits         byte
}

type change struct {
	Type byte
	Text string
}

type Lines_diff struct {
	chars1    string
	chars2    string
	lineArray []string
}

const DIFF_DELETE = -1
const DIFF_INSERT = 1
const DIFF_EQUAL = 0
const max32 = int32(uint32(1<<31) - 1)

// Define some regex patterns for matching boundaries.
var nonAlphaNumericRegex_, _ = regexp.Compile(`[^a-zA-Z0-9]`)
var whitespaceRegex_, _ = regexp.Compile(`\s`)
var linebreakRegex_, _ = regexp.Compile(`[\r\n]`)
var blanklineEndRegex_, _ = regexp.Compile(`\n\r?\n$`)
var blanklineStartRegex_, _ = regexp.Compile(`^\r?\n\r?\n`)

func concat(old1, old2 []change) []change {
	newslice := make([]change, len(old1)+len(old2))
	copy(newslice, old1)
	copy(newslice[len(old1):], old2)
	return newslice
}

func splice(slice []int, index int, amount int, elements ...int) []int {
	slice = append(slice[:index], append(elements, slice[index+amount:]...)...)
	return slice
}

type Patch struct {
	diffs   []change
	start1  int
	start2  int
	length1 int
	length2 int
}

func (patch *Patch) String() {
	var coords1, coords2 string

	if patch.length1 == 0 {
		coords1 = patch.start1 + ",0"
	} else if patch.length1 == 1 {
		coords1 = strconv.FormatInt(patch.start1+1, 32)
	} else {
		coords1 = (patch.start1 + 1) + "," + patch.length1
	}

	if patch.length2 == 0 {
		coords2 = patch.start2 + ",0"
	} else if patch.length2 == 1 {
		coords2 = strconv.FormatInt(this.start2+1, 32)
	} else {
		coords2 = (patch.start2 + 1) + "," + patch.length2
	}

	var text bytes.Buffer
	text.WriteString("@@ -" + coords1 + " +" + coords2 + " @@\n")

	// Escape the body of the patch with %xx notation.
	for _, aDiff := range patch.diffs {
		switch aDiff.Type {
		case DIFF_INSERT:
			text.WriteString('+')
			break
		case DIFF_DELETE:
			text.WriteString('-')
			break
		case DIFF_EQUAL:
			text.WriteString(' ')
			break
		}

		text.WriteString(url.QueryEscape(aDiff.text).Replace('+', ' '))
		text.WriteString("\n")
	}

	return unescapeForEncodeUriCompatability(text.String())
}

func (dmp *DiffMatchPatch) diff_main(text1 string, text2 string, opt ...interface{}) []change {
	checklines := true, deadline

	if opt {
		if opt[0] != nil {
			checklines = opt[0]
		}

		if opt[1] == nil {
			if dmp.DiffTimeout <= 0 {
				deadline = max32
			} else {
				deadline = time.Now().Unix() + dmp.DiffTimeout*1000
			}
		}
	}

	diffs := []change{}
	if text1 == text2 {
		if text1 != nil {
			diffs = append(diffs, diff{DIFF_EQUAL, text1})
		}
		return diffs
	}

	commonlength := dmp.diff_commonPrefix(text1, text2)
	commonprefix := text1[0:commonlength]
	text1 = text1[commonlength:]
	text2 = text2[commonlength:]

	// Trim off common suffix (speedup).
	commonlength = dmp.diff_commonSuffix(text1, text2)
	commonsuffix := text1[len(text1)-commonlength:]
	text1 = text1[0 : len(text1)-commonlength]
	text2 = text2[0 : len(text2)-commonlength]

	// Compute the diff on the middle block.
	diffs = dmp.diff_compute_(text1, text2, checklines, deadline)

	// Restore the prefix and suffix.
	if commonprefix != nil {
		diffs = append([]change{change{DIFF_EQUAL, commonprefix}}, diffs)
	}

	if commonsuffix != nil {
		diffs = append(diffs, []change{change{DIFF_EQUAL, commonsuffix}})
	}

	dmp.diff_cleanupMerge(diffs)
	return diffs
}

func (dmp *DiffMatchPatch) diff_compute_(text1, text2, checklines, deadline) {
	diffs = []change{}

	if len(text1) == 0 {
		// Just add some text (speedup).
		return append(diffs, change{DIFF_INSERT, text2})
	}

	if len(text2) == 0 {
		// Just delete some text (speedup).
		return append(diffs, change{DIFF_INSERT, text1})
	}

	var longtext, shorttext string

	if len(text1) > len(text2) {
		longtext = text1
		shorttext = text2
	} else {
		longtext = text2
		shorttext = text1
	}

	var i = longtext.indexOf(shorttext)

	if i != -1 {
		op := DIFF_INSERT
		// Swap insertions for deletions if diff is reversed.
		if len(text1) > len(text2) {
			op = DIFF_DELETE
		}
		// Shorter text is inside the longer text (speedup).
		diffs = []change{
			change{op, longtext[0:i]},
			change{DIFF_EQUAL, shorttext},
			change{op, longtext[i+len(shorttext):]},
		}

		return diffs
	}

	if len(shorttext) == 1 {
		// Single character string.
		// After the previous speedup, the character can't be an equality.
		return []change{
			change{DIFF_DELETE, text1},
			change{DIFF_INSERT, text2},
		}
	}

	// Check to see if the problem can be split in two.
	hm := dmp.diff_halfMatch_(text1, text2)
	if hm != nil {
		// A half-match was found, sort out the return data.
		text1_a := hm[0]
		text1_b := hm[1]
		text2_a := hm[2]
		text2_b := hm[3]
		mid_common := hm[4]
		// Send both pairs off for separate processing.
		diffs_a := dmp.diff_main(text1_a, text2_a, checklines, deadline)
		diffs_b := dmp.diff_main(text1_b, text2_b, checklines, deadline)
		// Merge the results.
		// TODO: Concat should accept several arguments
		concat1 := concat(diffs_a, []change{change{DIFF_EQUAL, mid_common}})
		return concat(concat1, diffs_b)
	}

	if checklines && len(len(text1)) && len(text2) > 100 {
		return dmp.diff_lineMode_(text1, text2, deadline)
	}

	return dmp.diff_bisect_(text1, text2, deadline)
}

/**
 * Do a quick line-level diff on both strings, then rediff the parts for
 * greater accuracy.
 * This speedup can produce non-minimal diffs.
 * @param {string} text1 Old string to be diffed.
 * @param {string} text2 New string to be diffed.
 * @param {number} deadline Time when the diff should be complete by.
 * @return {!Array.<!diff_match_patch.Diff>} Array of diff tuples.
 * @private
 */
func (dmp *DiffMatchPatch) diff_lineMode_(text1, text2, deadline int) {
	// Scan the text on a line-by-line basis first.
	a := dmp.diff_linesToChars_(text1, text2)
	text1 = a.chars1
	text2 = a.chars2
	linearray := a.lineArray

	diffs := dmp.diff_main(text1, text2, false, deadline)

	// Convert the diff back to original text.
	dmp.diff_charsToLines_(diffs, linearray)
	// Eliminate freak matches (e.g. blank lines)
	dmp.diff_cleanupSemantic(diffs)

	// Rediff any replacement blocks, this time character-by-character.
	// Add a dummy entry at the end.
	append(diffs, change{DIFF_EQUAL, ""})

	pointer := 0
	count_delete := 0
	count_insert := 0
	text_delete := ""
	text_insert := ""

	for pointer < len(diffs) {
		switch diffs[pointer].Type {
		case DIFF_INSERT:
			count_insert++
			text_insert += diffs[pointer].Text
			break

		case DIFF_DELETE:
			count_delete++
			text_delete += diffs[pointer].Text
			break

		case DIFF_EQUAL:
			// Upon reaching an equality, check for prior redundancies.
			if count_delete >= 1 && count_insert >= 1 {
				// Delete the offending records and add the merged ones.
				newdiffs := splice(diffs, pointer-count_delete-count_insert,
					count_delete+count_insert)

				pointer = pointer - count_delete - count_insert
				a = dmp.diff_main(text_delete, text_insert, false, deadline)
				for j := len(a) - 1; j >= 0; j-- {
					splice(diffs, pointer, 0, a[j])
				}
				pointer = pointer + len(a)
			}

			count_insert := 0
			count_delete := 0
			text_delete := ""
			text_insert := ""
			break
		}
		pointer++
	}

	return newdiffs[:len(newdiffs-1)] // Remove the dummy entry at the end.
}

func (dmp *DiffMatchPatch) diff_bisect_(text1, text2, deadline int) {
	// Cache the text lengths to prevent multiple calls.
	text1_length := len(text1)
	text2_length := len(text2)

	max_d := math.Ceil((text1_length + text2_length) / 2)
	v_offset := max_d
	v_length := 2 * max_d
	v1 := make([]int, v_length)
	v2 := make([]int, v_length)
	//var v1 = new Array(v_length)
	//var v2 = new Array(v_length)

	// Setting all elements to -1 is faster in Chrome & Firefox than mixing
	// integers and undefined.
	for x := 0; x < v_length; x++ {
		v1[x] = -1
		v2[x] = -1
	}

	v1[v_offset+1] = 0
	v2[v_offset+1] = 0

	delta := text1_length - text2_length
	// If the total number of characters is odd, then the front path will collide
	// with the reverse path.
	front := (delta%2 != 0)
	// Offsets for start and end of k loop.
	// Prevents mapping of space beyond the grid.
	k1start := 0
	k1end := 0
	k2start := 0
	k2end := 0
	for d := 0; d < max_d; d++ {
		// Bail out if deadline is reached.
		if time.Now().Unix() > deadline {
			break
		}

		// Walk the front path one step.
		for k1 := -d + k1start; k1 <= d-k1end; k1 += 2 {
			var k1_offset = v_offset + k1
			var int x1

			if k1 == -d || (k1 != d && v1[k1_offset-1] < v1[k1_offset+1]) {
				x1 = v1[k1_offset+1]
			} else {
				x1 = v1[k1_offset-1] + 1
			}

			y1 := x1 - k1
			for x1 < text1_length && y1 < text2_length &&
				text1[x1] == text2[y1] {
				x1++
				y1++
			}
			v1[k1_offset] = x1
			if x1 > text1_length {
				// Ran off the right of the graph.
				k1end += 2
			} else if y1 > text2_length {
				// Ran off the bottom of the graph.
				k1start += 2
			} else if front {
				k2_offset := v_offset + delta - k1
				if k2_offset >= 0 && k2_offset < v_length && v2[k2_offset] != -1 {
					// Mirror x2 onto top-left coordinate system.
					x2 := text1_length - v2[k2_offset]
					if x1 >= x2 {
						// Overlap detected.
						return dmp.diff_bisectSplit_(text1, text2, x1, y1, deadline)
					}
				}
			}
		}

		// Walk the reverse path one step.
		for k2 := -d + k2start; k2 <= d-k2end; k2 += 2 {
			k2_offset = v_offset + k2
			var int x2
			if k2 == -d || (k2 != d && v2[k2_offset-1] < v2[k2_offset+1]) {
				x2 = v2[k2_offset+1]
			} else {
				x2 = v2[k2_offset-1] + 1
			}
			var y2 = x2 - k2
			for x2 < text1_length &&
				y2 < text2_length &&
				(text1[text1_length-x2-1] == text2[text2_length-y2-1]) {
				x2++
				y2++
			}
			v2[k2_offset] = x2
			if x2 > text1_length {
				// Ran off the left of the graph.
				k2end += 2
			} else if y2 > text2_length {
				// Ran off the top of the graph.
				k2start += 2
			} else if !front {
				var k1_offset = v_offset + delta - k2
				if k1_offset >= 0 && k1_offset < v_length && v1[k1_offset] != -1 {
					x1 := v1[k1_offset]
					y1 := v_offset + x1 - k1_offset
					// Mirror x2 onto top-left coordinate system.
					x2 = text1_length - x2
					if x1 >= x2 {
						// Overlap detected.
						return dmp.diff_bisectSplit_(text1, text2, x1, y1, deadline)
					}
				}
			}
		}
	}
	// Diff took too long and hit the deadline or
	// number of diffs equals number of characters, no commonality at all.
	return []change{
		change{DIFF_DELETE, text1},
		change{DIFF_INSERT, text2},
	}
}

func (dmp *DiffMatchPatch) diff_bisectSplit_(text1, text2, x int, y int, deadline int) {
	text1a := text1[0:x]
	text2a := text2[0:y]
	text1b := text1[x:]
	text2b := text2[y:]

	// Compute both diffs serially.
	diffs := dmp.diff_main(text1a, text2a, false, deadline)
	diffsb := dmp.diff_main(text1b, text2b, false, deadline)

	return concat(diffs, diffsb)
}

func (dmp *DiffMatchPatch) diff_linesToChars_(text1, text2) {
	lineArray := []string{}      // e.g. lineArray[4] == 'Hello\n'
	lineHash := map[string]int{} // e.g. lineHash['Hello\n'] == 4

	// '\x00' is a valid character, but various debuggers don't like it.
	// So we'll insert a junk entry to avoid generating a null character.
	append(lineArray, "")

	/**
	 * Split a text into an array of strings.  Reduce the texts to a string of
	 * hashes where each Unicode character represents one line.
	 * Modifies linearray and linehash through being a closure.
	 * @param {string} text String to encode.
	 * @return {string} Encoded string.
	 * @private
	 */
	diff_linesToCharsMunge_ = func(text string) {
		var chars bytes.Buffer
		// Walk the text, pulling out a substring for each line.
		// text.split('\n') would would temporarily double our memory footprint.
		// Modifying text would create many large strings to garbage collect.
		lineStart := 0
		lineEnd := -1

		// Keeping our own length variable is faster than looking it up.
		lineArrayLength := len(lineArray)

		for lineEnd < len(text)-1 {
			lineEnd = text.indexOf('\n', lineStart)
			if lineEnd == -1 {
				lineEnd = len(text) - 1
			}
			line := text[lineStart : lineEnd+1]
			lineStart = lineEnd + 1

			var ok bool
			lineValue, ok = lineHash[line]

			if ok {
				chars.WriteString(string(lineValue))
			} else {
				append(lineArray, line)
				lineHash[line] = len(lineArray) - 1
				chars.WriteString(len(lineArray) - 1)
				//lineArrayLength += 1
				//lineArray[lineArrayLength] = line
			}
		}
		return chars.String()
	}

	chars1 := diff_linesToCharsMunge_(text1)
	chars2 := diff_linesToCharsMunge_(text2)

	return Lines_diff{chars1, chars2, lineArray}
}

/*
 * Rehydrate the text in a diff from a string of line hashes to real lines of
 * text.
 * @param {!Array.<!diff_match_patch.Diff>} diffs Array of diff tuples.
 * @param {!Array.<string>} lineArray Array of unique strings.
 * @private
 */
func (dmp *DiffMatchPatch) diff_charsToLines_(diffs, lineArray) {
	for x := 0; x < len(diffs); x++ {
		var chars = diffs[x].Text
		var text = []string{}
		for y := 0; y < len(chars); y++ {
			text[y] = lineArray[chars.charCodeAt(y)]
		}
		diffs[x].Text = strings.Join(text, "")
	}
}

/**
 * Determine the common prefix of two strings.
 * @param {string} text1 First string.
 * @param {string} text2 Second string.
 * @return {number} The number of characters common to the start of each
 *     string.
 */
func (dmp *DiffMatchPatch) diff_commonPrefix(text1 string, text2 string) int {
	// Quick check for common null cases.
	// if (text1 == nil || text2 == nil || text1[0]) != text2.charAt(0)) {
	//   return 0
	// }

	n := math.Min(len(text1), len(text2))
	for i := 0; i < n; i++ {
		if text1[i] != text2[i] {
			return i
		}
	}
	return n

	// Binary search.
	// Performance analysis: http://neil.fraser.name/news/2007/10/09/
	/*
		pointermin := 0
		pointermax := math.Min(len(text1), len(text2))
		pointermid := pointermax
		pointerstart := 0
		for pointermin < pointermid {
			if text1[pointerstart:pointermid] ==
				text2[pointerstart:pointermid] {
				pointermin = pointermid
				pointerstart = pointermin
			} else {
				pointermax = pointermid
			}
			pointermid = math.Floor((pointermax-pointermin)/2 + pointermin)
		}
		return pointermid
	*/
}

/**
 * Determine the common suffix of two strings.
 * @param {string} text1 First string.
 * @param {string} text2 Second string.
 * @return {number} The number of characters common to the end of each string.
 */
func (dmp *DiffMatchPatch) diff_commonSuffix(text1 string, text2 string) {
	text1_length = len(text1)
	text2_length = len(text2)
	n := math.Min(text1_length, text2_length)
	for i := 1; i <= n; i++ {
		if text1[text1_length-i] != text2[text2_length-i] {
			return i - 1
		}
	}
	return n
	// Binary search.
	// Performance analysis: http://neil.fraser.name/news/2007/10/09/
	/*
		pointermin := 0
		pointermax := math.Min(len(text1), len(text2))
		pointermid := pointermax
		pointerend := 0
		for pointermin < pointermid {
			if text1[len(text1)-pointermid:len(text1)-pointerend] ==
				text2[len(text2)-pointermid:len(text2)-pointerend] {
				pointermin = pointermid
				pointerend = pointermin
			} else {
				pointermax = pointermid
			}
			pointermid = math.Floor((pointermax-pointermin)/2 + pointermin)
		}
		return pointermid
	*/
}

/**
 * Determine if the suffix of one string is the prefix of another.
 * @param {string} text1 First string.
 * @param {string} text2 Second string.
 * @return {number} The number of characters common to the end of the first
 *     string and the start of the second string.
 * @private
 */
func (dmp *DiffMatchPatch) diff_commonOverlap_(text1, text2) int {
	// Cache the text lengths to prevent multiple calls.
	text1_length := len(text1)
	text2_length := len(text2)
	// Eliminate the null case.
	if text1_length == 0 || text2_length == 0 {
		return 0
	}
	// Truncate the longer string.
	if text1_length > text2_length {
		text1 = text1[text1_length-text2_length:]
	} else if text1_length < text2_length {
		text2 = text2[0:text1_length]
	}
	var text_length = math.Min(text1_length, text2_length)
	// Quick check for the worst case.
	if text1 == text2 {
		return text_length
	}

	// Start by looking for a single character match
	// and increase length until no match is found.
	// Performance analysis: http://neil.fraser.name/news/2010/11/04/
	best := 0
	length := 1
	for {
		pattern := text1[text_length-length:]
		found := text2.indexOf(pattern)
		if found == -1 {
			return best
		}
		length += found
		if found == 0 || text1[text_length-length:] == text2[0:length] {
			best = length
			length++
		}
	}
}

/**
 * Do the two texts share a substring which is at least half the length of the
 * longer text?
 * This speedup can produce non-minimal diffs.
 * @param {string} text1 First string.
 * @param {string} text2 Second string.
 * @return {Array.<string>} Five element Array, containing the prefix of
 *     text1, the suffix of text1, the prefix of text2, the suffix of
 *     text2 and the common middle.  Or null if there was no match.
 * @private
 */
func (dmp *DiffMatchPatch) diff_halfMatch_(text1, text2 string) []string {
	if dmp.Diff_Timeout <= 0 {
		// Don't risk returning a non-optimal diff if we have unlimited time.
		return nil
	}

	var longtext, shorttext string
	if len(text1) > len(text2) {
		longtext = text1
		shorttext = text2
	} else {
		longtext = text2
		shorttext = text1
	}

	//TODO
	if len(longtext) < 4 || len(shorttext)*2 < len(longtext) {
		return null // Pointless.
	}

	/**
	 * Does a substring of shorttext exist within longtext such that the substring
	 * is at least half the length of longtext?
	 * Closure, but does not reference any external variables.
	 * @param {string} longtext Longer string.
	 * @param {string} shorttext Shorter string.
	 * @param {number} i Start index of quarter length substring within longtext.
	 * @return {Array.<string>} Five element Array, containing the prefix of
	 *     longtext, the suffix of longtext, the prefix of shorttext, the suffix
	 *     of shorttext and the common middle.  Or null if there was no match.
	 * @private
	 */
	diff_halfMatchI_ = func(longtext, shorttext, i) {
		// Start with a 1/4 length substring at position i as a seed.
		seed := longtext[i : i+math.Floor(len(longtext)/4)]
		j := -1
		best_common := ""
		best_longtext_a := ""
		best_longtext_b := ""
		best_shorttext_a := ""
		best_shorttext_b := ""

		if j < len(shorttext) {
			j = strings.Index(shorttext, string(j+1))
			for {
				j := strings.Index(shorttext, string(j+1))
				if j == -1 {
					break
				}

				prefixLength := dmp.diff_commonPrefix(longtext[i:], shorttext[j:])
				suffixLength := dmp.diff_commonSuffix(longtext[0:i], shorttext[0:j])

				if len(best_common) < suffixLength+prefixLength {
					best_common = shorttext[j-suffixLength:j] + shorttext[j:j+prefixLength]
					best_longtext_a = longtext[0 : i-suffixLength]
					best_longtext_b = longtext[i+prefixLength:]
					best_shorttext_a = shorttext[0 : j-suffixLength]
					best_shorttext_b = shorttext[j+prefixLength:]
				}
			}
		}

		if len(best_common)*2 >= len(longtext) {
			return []string{
				best_longtext_a,
				best_longtext_b,
				best_shorttext_a,
				best_shorttext_b,
				best_common,
			}
		} else {
			return nil
		}
	}

	// First check if the second quarter is the seed for a half-match.
	hm1 := diff_halfMatchI_(longtext, shorttext,
		math.Ceil(len(longtext)/4))

	// Check again based on the third quarter.
	hm2 := diff_halfMatchI_(longtext, shorttext,
		math.Ceil(len(longtext)/2))

	hm := []string{}

	if hm1 == nil && hm2 == nil {
		return nil
	} else if hm2 == nil {
		hm = hm1
	} else if hm1 == nil {
		hm = hm2
	} else {
		// Both matched.  Select the longest.
		if len(hm1[4]) > len(hm2[4]) {
			hm = hm1
		} else {
			hm = hm2
		}
	}

	// A half-match was found, sort out the return data.
	var text1_a, text1_b, text2_a, text2_b string
	if len(text1) > len(text2) {
		return hm
	} else {
		return []string{hm[2], hm[3], hm[0], hm[1], hm[4]}
	}
}

/**
 * Reduce the number of edits by eliminating semantically trivial equalities.
 * @param {!Array.<!diff_match_patch.Diff>} diffs Array of diff tuples.
 */
func (dmp *DiffMatchPatch) diff_cleanupSemantic(diffs []change) {
	changes := false
	equalities := []int{}    // Stack of indices where equalities are found.
	var equalitiesLength = 0 // Keeping our own length var is faster in JS.
	/** @type {?string} */
	lastequality := null
	// Always equal to diffs[equalities[equalitiesLength - 1]][1]
	pointer := 0 // Index of current position.
	// Number of characters that changed prior to the equality.
	length_insertions1 := 0
	length_deletions1 := 0
	// Number of characters that changed after the equality.
	length_insertions2 := 0
	length_deletions2 := 0

	for pointer < len(diffs) {
		if diffs[pointer].Type == DIFF_EQUAL { // Equality found.
			append(equalities, pointer)
			//equalities[equalitiesLength++] = pointer
			length_insertions1 = length_insertions2
			length_deletions1 = length_deletions2
			length_insertions2 = 0
			length_deletions2 = 0
			lastequality = diffs[pointer].Text
		} else { // An insertion or deletion.
			if diffs[pointer].Type == DIFF_INSERT {
				length_insertions2 += len(diffs[pointer].Text)
			} else {
				length_deletions2 += len(diffs[pointer].Text)
			}
			// Eliminate an equality that is smaller or equal to the edits on both
			// sides of it.
			if lastequality != nil && (len(lastequality) <=
				math.Max(length_insertions1, length_deletions1)) &&
				(len(lastequality) <= math.Max(length_insertions2, length_deletions2)) {
				// Duplicate record.
				//TODO
				// a = append(a[:i], append([]T{x}, a[i:]...)...)

				insPoint := equalities[equalitiesLength-1]
				diffs = append(
					diffs[:insPoint],
					append([]change{change{DIFF_DELETE, lastequality}}, diffs[insPoint:]))
				//diffs.splice(equalities[equalitiesLength - 1], 0, [DIFF_DELETE, lastequality])
				// Change second copy to insert.
				diffs[insPoint+1].Type = DIFF_INSERT
				// Throw away the equality we just deleted.
				_, diffs = diffs[len(diffs)-1], diffs[:len(diffs)-1]

				if equalitiesLength > 0 {
					_, diffs = diffs[len(diffs)-1], diffs[:len(diffs)-1]
					pointer = equalities[len(equalities)-1]
				} else {
					pointer = -1
				}

				length_insertions1 = 0 // Reset the counters.
				length_deletions1 = 0
				length_insertions2 = 0
				length_deletions2 = 0
				lastequality = nil
				changes = true
			}
		}
		pointer++
	}

	// Normalize the diff.
	if changes {
		dmp.diff_cleanupMerge(diffs)
	}
	dmp.diff_cleanupSemanticLossless(diffs)

	// Find any overlaps between deletions and insertions.
	// e.g: <del>abcxxx</del><ins>xxxdef</ins>
	//   -> <del>abc</del>xxx<ins>def</ins>
	// e.g: <del>xxxabc</del><ins>defxxx</ins>
	//   -> <ins>def</ins>xxx<del>abc</del>
	// Only extract an overlap if it is as big as the edit ahead or behind it.
	pointer = 1
	for pointer < len(diffs) {
		if diffs[pointer-1].Type == DIFF_DELETE &&
			diffs[pointer].Type == DIFF_INSERT {

			deletion := diffs[pointer-1].Text
			insertion := diffs[pointer].Text
			overlap_length1 := dmp.diff_commonOverlap_(deletion, insertion)
			overlap_length2 := dmp.diff_commonOverlap_(insertion, deletion)
			if overlap_length1 >= overlap_length2 {
				if overlap_length1 >= len(deletion)/2 ||
					overlap_length1 >= len(insertion)/2 {

					// Overlap found.  Insert an equality and trim the surrounding edits.

					diffs = append(
						diffs[:pointer],
						append([]change{change{DIFF_EQUAL, insertion[0:overlap_length1]}}, diffs[pointer:]))
					//diffs.splice(pointer, 0,
					//    [DIFF_EQUAL, insertion[0 : overlap_length1)]]
					diffs[pointer-1].Text =
						deletion[0 : len(deletion)-overlap_length1]
					diffs[pointer+1].Text = insertion[overlap_length1]
					pointer++
				}
			} else {
				if overlap_length2 >= len(deletion)/2 ||
					overlap_length2 >= len(insertion)/2 {
					// Reverse overlap found.
					// Insert an equality and swap and trim the surrounding edits.
					diffs = append(
						diffs[:pointer],
						append([]change{change{DIFF_EQUAL, insertion[0:overlap_length2]}}, diffs[pointer:]))
					// diffs.splice(pointer, 0,
					//     [DIFF_EQUAL, deletion[0 : overlap_length2)]]
					diffs[pointer-1][0] = DIFF_INSERT
					diffs[pointer-1][1] =
						insertion[0 : len(insertion)-overlap_length2]
					diffs[pointer+1][0] = DIFF_DELETE
					diffs[pointer+1][1] =
						deletion[overlap_length2:]
					pointer++
				}
			}
			pointer++
		}
		pointer++
	}
}

/**
 * Look for single edits surrounded on both sides by equalities
 * which can be shifted sideways to align the edit to a word boundary.
 * e.g: The c<ins>at c</ins>ame. -> The <ins>cat </ins>came.
 * @param {!Array.<!diff_match_patch.Diff>} diffs Array of diff tuples.
 */
func (dmp *DiffMatchPatch) diff_cleanupSemanticLossless(diffs []change) {

	/**
	 * Given two strings, compute a score representing whether the internal
	 * boundary falls on logical boundaries.
	 * Scores range from 6 (best) to 0 (worst).
	 * Closure, but does not reference any external variables.
	 * @param {string} one First string.
	 * @param {string} two Second string.
	 * @return {number} The score.
	 * @private
	 */
	diff_cleanupSemanticScore_ := func(one, two string) {
		if len(one) == 0 || len(two) == 0 {
			// Edges are the best.
			return 6
		}

		// Each port of this function behaves slightly differently due to
		// subtle differences in each language's definition of things like
		// 'whitespace'.  Since this function's purpose is largely cosmetic,
		// the choice has been made to use each language's native features
		// rather than force total conformity.
		var char1 = one[len(one)-1]
		var char2 = two[0]

		var nonAlphaNumeric1 = char1.match(diff_match_patch.nonAlphaNumericRegex_)
		var nonAlphaNumeric2 = char2.match(diff_match_patch.nonAlphaNumericRegex_)
		var whitespace1 = nonAlphaNumeric1 &&
			char1.match(diff_match_patch.whitespaceRegex_)
		var whitespace2 = nonAlphaNumeric2 &&
			char2.match(diff_match_patch.whitespaceRegex_)
		var lineBreak1 = whitespace1 &&
			char1.match(diff_match_patch.linebreakRegex_)
		var lineBreak2 = whitespace2 &&
			char2.match(diff_match_patch.linebreakRegex_)
		var blankLine1 = lineBreak1 &&
			one.match(diff_match_patch.blanklineEndRegex_)
		var blankLine2 = lineBreak2 &&
			two.match(diff_match_patch.blanklineStartRegex_)

		if blankLine1 || blankLine2 {
			// Five points for blank lines.
			return 5
		} else if lineBreak1 || lineBreak2 {
			// Four points for line breaks.
			return 4
		} else if nonAlphaNumeric1 && !whitespace1 && whitespace2 {
			// Three points for end of sentences.
			return 3
		} else if whitespace1 || whitespace2 {
			// Two points for whitespace.
			return 2
		} else if nonAlphaNumeric1 || nonAlphaNumeric2 {
			// One point for non-alphanumeric.
			return 1
		}
		return 0
	}

	pointer := 1
	// Intentionally ignore the first and last element (don't need checking).
	for pointer < len(diffs)-1 {
		if diffs[pointer-1].Type == DIFF_EQUAL &&
			diffs[pointer+1].Type == DIFF_EQUAL {
			// This is a single edit surrounded by equalities.
			equality1 := diffs[pointer-1].Text
			edit := diffs[pointer].Text
			equality2 := diffs[pointer+1].Text

			// First, shift the edit as far left as possible.
			var commonOffset = dmp.diff_commonSuffix(equality1, edit)
			if commonOffset > 0 {
				commonString := edit[len(edit)-commonOffset:]
				equality1 = equality1[0 : len(equality1)-commonOffset]
				edit = commonString + edit[0:len(edit)-commonOffset]
				equality2 = commonString + equality2
			}

			// Second, step character by character right, looking for the best fit.
			bestEquality1 := equality1
			bestEdit := edit
			bestEquality2 := equality2
			bestScore := diff_cleanupSemanticScore_(equality1, edit) +
				diff_cleanupSemanticScore_(edit, equality2)

			for edit[0] == equality2[0] {
				equality1 += edit[0]
				edit = edit[1:] + equality2[0]
				equality2 = equality2[1:]
				score := diff_cleanupSemanticScore_(equality1, edit) +
					diff_cleanupSemanticScore_(edit, equality2)
				// The >= encourages trailing rather than leading whitespace on edits.
				if score >= bestScore {
					bestScore = score
					bestEquality1 = equality1
					bestEdit = edit
					bestEquality2 = equality2
				}
			}

			if diffs[pointer-1].Text != bestEquality1 {
				// We have an improvement, save it back to the diff.
				if len(bestEquality1 != 0) {
					diffs[pointer-1].Text = bestEquality1
				} else {
					//diffs.splice(pointer-1, 1) // FIXME
					append(a[:pointer-1], a[pointer:])
					pointer--
				}

				diffs[pointer].Text = bestEdit
				if bestEquality2 {
					diffs[pointer+1].Text = bestEquality2
				} else {
					diffs.splice(pointer+1, 1)
					append(a[:pointer+1], a[pointer+2:])
					pointer--
				}
			}
		}
		pointer++
	}
}

/**
 * Reduce the number of edits by eliminating operationally trivial equalities.
 * @param {!Array.<!diff_match_patch.Diff>} diffs Array of diff tuples.
 */
func (dmp *DiffMatchPatch) diff_cleanupEfficiency(diffs) {
	changes := false
	equalities := make([]int, 0) // Stack of indices where equalities are found.
	//equalitiesLength := 0     // Keeping our own length var is faster in JS.
	/** @type {?string} */
	lastequality := nil
	// Always equal to diffs[equalities[equalitiesLength - 1]][1]
	pointer := 0 // Index of current position.
	// Is there an insertion operation before the last equality.
	pre_ins := false
	// Is there a deletion operation before the last equality.
	pre_del := false
	// Is there an insertion operation after the last equality.
	post_ins := false
	// Is there a deletion operation after the last equality.
	post_del := false
	for pointer < len(diffs) {
		if diffs[pointer].Type == DIFF_EQUAL { // Equality found.
			if len(diffs[pointer].Text) < dmp.Diff_EditCost &&
				(post_ins || post_del) {
				// Candidate found.
				//equalities[equalitiesLength++] = pointer
				append(equalities, pointer)
				pre_ins = post_ins
				pre_del = post_del
				lastequality = diffs[pointer].Text
			} else {
				// Not a candidate, and can never become one.
				//equalitiesLength = 0
				equalities = []int{}
				lastequality = ""
			}
			post_ins = false
			post_del = false
		} else { // An insertion or deletion.
			if diffs[pointer].Type == DIFF_DELETE {
				post_del = true
			} else {
				post_ins = true
			}
			/*
			 * Five types to be split:
			 * <ins>A</ins><del>B</del>XY<ins>C</ins><del>D</del>
			 * <ins>A</ins>X<ins>C</ins><del>D</del>
			 * <ins>A</ins><del>B</del>X<ins>C</ins>
			 * <ins>A</del>X<ins>C</ins><del>D</del>
			 * <ins>A</ins><del>B</del>X<del>C</del>
			 */
			if lastequality && ((pre_ins && pre_del && post_ins && post_del) ||
				((len(lastequality) < dmp.Diff_EditCost/2) &&
					(pre_ins+pre_del+post_ins+post_del) == 3)) {
				// Duplicate record.
				append(diffs, change{DIFF_DELETE, lastequality})
				// Change second copy to insert.
				diffs[equalities[len(equalities)-1]+1].Type = DIFF_INSERT
				//equalitiesLength--;  // Throw away the equality we just deleted
				_, equalities = equalities[len(equalities)-1], equalities[:len(equalities)-1]
				lastequality = ""
				if pre_ins && pre_del {
					// No changes made which could affect previous entry, keep going.
					post_ins = true
					post_del = true
					equalities = []int{}
				} else {
					equalitiesLength-- // Throw away the previous equality.
					if len(equalities) > 0 {
						_, equalities = equalities[len(equalities)-1], equalities[:len(equalities)-1]
						pointer = equalities[len(equalities)-1]
					} else {
						pointer = -1
					}
					post_ins = true
					post_del = false
				}
				changes = true
			}
		}
		pointer++
	}

	if changes {
		dmp.diff_cleanupMerge(diffs)
	}
}

/**
 * Reorder and merge like edit sections.  Merge equalities.
 * Any edit section can move as long as it doesn't cross an equality.
 * @param diffs List of Diff objects.
 */
func (dmp *DiffMatchPatch) diff_cleanupMerge(diffs) {
	// Add a dummy entry at the end.
	append(diffs, change{DIFF_EQUAL, ""})
	pointer := 0
	count_delete := 0
	count_insert := 0
	text_delete := ""
	text_insert := ""
	var commonlength int
	for pointer < len(diffs) {
		switch diffs[pointer].Type {
		case DIFF_INSERT:
			count_insert++
			text_insert += diffs[pointer].Text
			pointer++
			break
		case DIFF_DELETE:
			count_delete++
			text_delete += diffs[pointer].Text
			pointer++
			break
		case DIFF_EQUAL:
			// Upon reaching an equality, check for prior redundancies.
			if count_delete+count_insert > 1 {
				if count_delete != 0 && count_insert != 0 {
					// Factor out any common prefixies.
					commonlength = dmp.diff_commonPrefix(text_insert, text_delete)
					if commonlength != 0 {
						if (pointer-count_delete-count_insert) > 0 &&
							diffs[pointer-count_delete-count_insert-1].Type == DIFF_EQUAL {
							diffs[pointer-count_delete-count_insert-1].Text += text_insert[0:commonlength]
						} else {
							diffs = append([]change{DIFF_EQUAL, text_insert[0:commonlength]}, diffs)
							//diffs.Insert(0, new Diff())
							pointer++
						}
						text_insert = text_insert[commonlength:]
						text_delete = text_delete[commonlength:]
					}
					// Factor out any common suffixies.
					commonlength = dmp.diff_commonSuffix(text_insert, text_delete)
					if commonlength != 0 {
						diffs[pointer].Text = text_insert[len(text_insert)-commonlength:] + diffs[pointer].Text
						text_insert = text_insert[0 : len(text_insert)-commonlength]
						text_delete = text_delete[0 : len(text_delete)-commonlength]
					}
				}
				// Delete the offending records and add the merged ones.
				if count_delete == 0 {
					diffs = splice(diffs, pointer-count_insert,
						count_delete+count_insert,
						change{DIFF_INSERT, text_insert})
				} else if count_insert == 0 {
					diffs = splice(diffs, pointer-count_delete,
						count_delete+count_insert, change{DIFF_DELETE, text_delete})
					//change{DIFF_DELETE, text_delete})
				} else {
					diffs = splice(diffs, pointer-count_delete-count_insert,
						count_delete+count_insert,
						change{DIFF_DELETE, text_delete},
						change{DIFF_INSERT, text_insert})
				}

				pointer = pointer - count_delete - count_insert + 1
				if count_delete != 0 {
					pointer = pointer + 1
				}
				if count_insert != 0 {
					pointer = pointer + 1
				}
			} else if pointer != 0 && diffs[pointer-1].Type == DIFF_EQUAL {
				// Merge this equality with the previous one.
				diffs[pointer-1].Text += diffs[pointer].Text
				diffs = append(diffs[:pointer], diffs[pointer+1:]...)
			} else {
				pointer++
			}
			count_insert = 0
			count_delete = 0
			text_delete = string.Empty
			text_insert = string.Empty
			break
		}
	}
	if diffs[len(diffs)-1].len(text) == 0 {
		append(diffs[:len(diffs)-1], diffs[len(diffs)-1:]...) // Remove the dummy entry at the end.
	}

	// Second pass: look for single edits surrounded on both sides by
	// equalities which can be shifted sideways to eliminate an equality.
	// e.g: A<ins>BA</ins>C -> <ins>AB</ins>AC
	changes := false
	pointer = 1
	// Intentionally ignore the first and last element (don't need checking).
	for pointer < (len(diffs) - 1) {
		if diffs[pointer-1].Type == DIFF_EQUAL &&
			diffs[pointer+1].Type == DIFF_EQUAL {
			// This is a single edit surrounded by equalities.
			if diffs[pointer].text.EndsWith(diffs[pointer-1].text,
				StringComparison.Ordinal) {
				// Shift the edit over the previous equality.
				diffs[pointer].Text = diffs[pointer-1].text +
					diffs[pointer].Text[0:len(diffs[pointer].Text)-len(diffs[pointer-1].text)]
				diffs[pointer+1].text = diffs[pointer-1].text
				+diffs[pointer+1].text
				diffs.Splice(pointer-1, 1)
				changes = true
			} else if diffs[pointer].text.StartsWith(diffs[pointer+1].text,
				StringComparison.Ordinal) {
				// Shift the edit over the next equality.
				diffs[pointer-1].text += diffs[pointer+1].text
				diffs[pointer].text =
					diffs[pointer].text[diffs[pointer+1].len(text):]
				+diffs[pointer+1].text
				diffs.Splice(pointer+1, 1)
				changes = true
			}
		}
		pointer++
	}
	// If shifts were made, the diff needs reordering and another shift sweep.
	if changes {
		dmp.diff_cleanupMerge(diffs)
	}
}

/**
 * loc is a location in text1, comAdde and return the equivalent location in
 * text2.
 * e.g. "The cat" vs "The big cat", 1->1, 5->8
 * @param diffs List of Diff objects.
 * @param loc Location within text1.
 * @return Location within text2.
 */
func (dmp *DiffMatchPatch) diff_xIndex(diffs []change, loc int) int {
	chars1 = 0
	chars2 = 0
	last_chars1 = 0
	last_chars2 = 0
	lastDiff := nil
	for i := 0; i < len(diffs); i++ {
		aDiff = *diffs[i]
		if aDiff.Type != DIFF_INSERT {
			// Equality or deletion.
			chars1 += len(aDiff.Text)
		}
		if aDiff.Type != DIFF_DELETE {
			// Equality or insertion.
			chars2 += aDiff.len(text)
		}
		if chars1 > loc {
			// Overshot the location.
			lastDiff = aDiff
			break
		}
		last_chars1 = chars1
		last_chars2 = chars2
	}
	if lastDiff != null && lastDiff.Type == DIFF_DELETE {
		// The location was deleted.
		return last_chars2
	}
	// Add the remaining character length.
	return last_chars2 + (loc - last_chars1)
}

/**
 * Convert a Diff list into a pretty HTML report.
 * @param diffs List of Diff objects.
 * @return HTML representation.
 */
/*
   func (dmp *DiffMatchPatch) diff_prettyHtml(diffs []change) {
     StringBuilder html = new StringBuilder()
     foreach (Diff aDiff in diffs) {
       text := aDiff.text.Replace("&", "&amp;").Replace("<", "&lt;")
         .Replace(">", "&gt;").Replace("\n", "&para;<br>")
       switch (aDiff.Type) {
         case DIFF_INSERT:
           html.Append("<ins style=\"background:#e6ffe6;\">").Append(text)
               .Append("</ins>")
           break
         case DIFF_DELETE:
           html.Append("<del style=\"background:#ffe6e6;\">").Append(text)
               .Append("</del>")
           break
         case DIFF_EQUAL:
           html.Append("<span>").Append(text).Append("</span>")
           break
       }
     }
     return html.ToString()
   }
*/
/**
 * Compute and return the source text (all equalities and deletions).
 * @param diffs List of Diff objects.
 * @return Source text.
 */
func (dmp *DiffMatchPatch) diff_text1(diffs []change) {
	//StringBuilder text = new StringBuilder()
	var text bytes.buffer

	for _, aDiff := range diffs {
		if aDiff.Type != DIFF_INSERT {
			text.WriteString(aDiff.text)
			//text.Append(aDiff.text)
		}
	}
	return text.String()
}

/**
 * Compute and return the destination text (all equalities and insertions).
 * @param diffs List of Diff objects.
 * @return Destination text.
 */
func (dmp *DiffMatchPatch) diff_text2(diffs []change) {
	var text bytes.buffer

	for _, aDiff := range diffs {
		if aDiff.Type != DIFF_DELETE {
			text.WriteString(aDiff.text)
		}
	}
	return text.String()
}

/**
 * Compute the Levenshtein distance; the number of inserted, deleted or
 * substituted characters.
 * @param diffs List of Diff objects.
 * @return Number of changes.
 */
func (dmp *DiffMatchPatch) diff_levenshtein(diffs []change) {
	levenshtein := 0
	insertions := 0
	deletions := 0

	for _, aDiff := range diffs {
		switch aDiff.Type {
		case DIFF_INSERT:
			insertions += len(aDiff).Text
			break
		case DIFF_DELETE:
			deletions += len(aDiff).Text
			break
		case DIFF_EQUAL:
			// A deletion and an insertion is one substitution.
			levenshtein += math.Max(insertions, deletions)
			insertions = 0
			deletions = 0
			break
		}
	}

	levenshtein += math.Max(insertions, deletions)
	return levenshtein
}

/**
 * Crush the diff into an encoded string which describes the operations
 * required to transform text1 into text2.
 * E.g. =3\t-2\t+ing  -> Keep 3 chars, delete 2 chars, insert 'ing'.
 * Operations are tab-separated.  Inserted text is escaped using %xx
 * notation.
 * @param diffs Array of Diff objects.
 * @return Delta text.
 */
func (dmp *DiffMatchPatch) diff_toDelta(diffs []change) {
	//StringBuilder text = new StringBuilder()
	var text bytes.buffer
	for _, aDiff := range diffs {
		switch aDiff.Type {
		case DIFF_INSERT:
			text.WriteString("+")
			text.WriteString(url.QueryEscape(aDiff.Text))
			text = text.Replace('+', ' ')
			text.WriteString("\t")
			break
		case DIFF_DELETE:
			text.Append("-").Append(aDiff.len(text)).Append("\t")
			break
		case DIFF_EQUAL:
			text.Append("=").Append(aDiff.len(text)).Append("\t")
			break
		}
	}
	delta := text.String()
	if len(delta) != 0 {
		// Strip off trailing tab character.
		delta = delta[0 : len(delta)-1]
		delta = unescapeForEncodeUriCompatability(delta)
	}
	return delta
}

/**
 * Given the original text1, and an encoded string which describes the
 * operations required to transform text1 into text2, comAdde the full diff.
 * @param text1 Source string for the diff.
 * @param delta Delta text.
 * @return Array of Diff objects or null if invalid.
 * @throws ArgumentException If invalid input.
 */
func (dmp *DiffMatchPatch) diff_fromDelta(text1, delta string) []change {
	//List<Diff> diffs = new List<Diff>()
	diffs := []change{}
	pointer := 0 // Cursor in text1
	var tokens []string = strings.Split(delta, "\t")

	for _, token := range tokens {
		if len(token) == 0 {
			// Blank tokens are ok (from a trailing \t).
			continue
		}

		// Each token begins with a one character parameter which specifies the
		// operation of this token (delete, insert, equality).
		param := token[1:]
		switch token[0] {
		case '+':
			// decode would change all "+" to " "
			param = string.Replace(param, "+", "%2b")

			param, _ = url.QueryUnescape(param)
			append(diffs, change{DIFF_INSERT, param})
			break
		case '-':
			// Fall through.
		case '=':
			var n int

			n, err := strconv.ParseInt(param, 10, 32)
			if err != nil {
				return log.Fatal(err)
			}

			if n < 0 {
				return log.Fatal("Negative number in diff_fromDelta: " + param)
			}

			text := text1[pointer:n]
			pointer += n

			if token[0] == '=' {
				append(diffs, change{DIFF_EQUAL, text})
			} else {
				append(diffs, change{DIFF_DELETE, text})
			}
			break
		default:
			// Anything else is an error.
			return log.Fatal("Invalid diff operation in diff_fromDelta: " + token[0])
		}
	}
	if pointer != len(text1) {
		return log.Fatal("Delta length (" + pointer + ") smaller than source text length (" + len(text1) + ").")
	}
	return diffs
}

//  MATCH FUNCTIONS

/**
 * Locate the best instance of 'pattern' in 'text' near 'loc'.
 * Returns -1 if no match found.
 * @param text The text to search.
 * @param pattern The pattern to search for.
 * @param loc The location to search around.
 * @return Best match index or -1.
 */
func (dmp *DiffMatchPatch) match_main(text string, pattern string, loc int) int {
	// Check for null inputs not needed since null can't be passed in C#.

	loc = math.Max(0, math.Min(loc, len(text)))
	if text == pattern {
		// Shortcut (potentially not guaranteed by the algorithm)
		return 0
	} else if len(text) == 0 {
		// Nothing to match.
		return -1
	} else if loc+len(pattern) <= len(text) && text[loc:len(pattern)] == pattern {
		// Perfect match at the perfect spot!  (Includes case of null pattern)
		return loc
	} else {
		// Do a fuzzy compare.
		return match_bitap(text, pattern, loc)
	}
}

/**
 * Locate the best instance of 'pattern' in 'text' near 'loc' using the
 * Bitap algorithm.  Returns -1 if no match found.
 * @param text The text to search.
 * @param pattern The pattern to search for.
 * @param loc The location to search around.
 * @return Best match index or -1.
 */
func (dmp *DiffMatchPatch) match_bitap(text string, pattern string, loc int) int {
	// Initialise the alphabet.
	s := match_alphabet(pattern)

	// Highest score beyond which we give up.
	var score_threshold float64 = Match_Threshold
	// Is there a nearby exact match? (speedup)
	best_loc = text.IndexOf(pattern, loc, StringComparison.Ordinal)
	if best_loc != -1 {
		score_threshold = math.Min(match_bitapScore(0, best_loc, loc,
			pattern), score_threshold)
		// What about in the other direction? (speedup)
		best_loc = text.LastIndexOf(pattern,
			math.Min(loc+len(pattern), len(text)),
			StringComparison.Ordinal)
		if best_loc != -1 {
			score_threshold = math.Min(match_bitapScore(0, best_loc, loc,
				pattern), score_threshold)
		}
	}

	// Initialise the bit arrays.
	matchmask := 1 << (len(pattern) - 1)
	best_loc = -1

	var bin_min, bin_mid int
	bin_max := len(pattern) + len(text)
	// Empty initialization added to appease C# compiler.
	last_rd = []int{}
	for d := 0; d < len(pattern); d++ {
		// Scan for the best match; each iteration allows for one more error.
		// Run a binary search to determine how far from 'loc' we can stray at
		// this error level.
		bin_min = 0
		bin_mid = bin_max
		for bin_min < bin_mid {
			if match_bitapScore(d, loc+bin_mid, loc, pattern) <= score_threshold {
				bin_min = bin_mid
			} else {
				bin_max = bin_mid
			}
			bin_mid = (bin_max-bin_min)/2 + bin_min
		}
		// Use the result from this iteration as the maximum for the next.
		bin_max = bin_mid
		start := math.Max(1, loc-bin_mid+1)
		finish := math.Min(loc+bin_mid, len(text)) + len(pattern)

		rd := []int{} //int[finish + 2] //TODO do it with make, length
		rd[finish+1] = (1 << d) - 1

		for j := finish; j >= start; j-- {
			var charMatch int
			if len(text) <= j-1 || !s.ContainsKey(text[j-1]) {
				// Out of range.
				charMatch = 0
			} else {
				charMatch = s[text[j-1]]
			}
			if d == 0 {
				// First pass: exact match.
				rd[j] = ((rd[j+1] << 1) | 1) & charMatch
			} else {
				// Subsequent passes: fuzzy match.
				rd[j] = ((rd[j+1]<<1)|1)&charMatch | (((last_rd[j+1] | last_rd[j]) << 1) | 1) | last_rd[j+1]
			}
			if (rd[j] & matchmask) != 0 {
				score = match_bitapScore(d, j-1, loc, pattern)
				// This match will almost certainly be better than any existing
				// match.  But check anyway.
				if score <= score_threshold {
					// Told you so.
					score_threshold = score
					best_loc = j - 1
					if best_loc > loc {
						// When passing loc, don't exceed our current distance from loc.
						start = math.Max(1, 2*loc-best_loc)
					} else {
						// Already passed loc, downhill from here on in.
						break
					}
				}
			}
		}
		if match_bitapScore(d+1, loc, loc, pattern) > score_threshold {
			// No hope for a (better) match at greater error levels.
			break
		}
		last_rd = rd
	}
	return best_loc
}

/**
 * Compute and return the score for a match with e errors and x location.
 * @param e Number of errors in match.
 * @param x Location of match.
 * @param loc Expected location of match.
 * @param pattern Pattern being sought.
 * @return Overall score for match (0.0 = good, 1.0 = bad).
 */
func (dmp *DiffMatchPatch) match_bitapScore(e, x, loc int, string pattern) {
	var accuracy float = float(e) / len(pattern)
	proximity := math.Abs(loc - x)
	if Match_Distance == 0 {
		// Dodge divide by zero error.
		if proximity == 0 {
			return accuracy
		} else {
			return 1.0
		}
	}
	return accuracy + (proximity / float64(Match_Distance))
}

/**
 * Initialise the alphabet for the Bitap algorithm.
 * @param pattern The text to encode.
 * @return Hash of character locations.
 */
func (dmp *DiffMatchPatch) match_alphabet(string pattern) {
	s := map[string]int{}
	char_pattern = strings.Split(pattern)
	for _, c := range char_pattern {
		if !s[c] {
			s[c] = 0
		}
	}
	i := 0

	for _, c := range char_pattern {
		value := s[c] | (1 << (len(pattern) - i - 1))
		s[c] = value
		i++
	}
	return s
}

//  PATCH FUNCTIONS

/**
 * Increase the context until it is unique,
 * but don't let the pattern expand beyond Match_MaxBits.
 * @param patch The patch to grow.
 * @param text Source text.
 */
func (dmp *DiffMatchPatch) patch_addContext(Patch patch, string text) {
	if len(text) == 0 {
		return
	}

	pattern := text[patch.start2:patch.length1]
	padding := 0

	// Look for the first and last matches of pattern in text.  If two
	// different matches are found, increase the pattern length.
	for text.Index(pattern, StringComparison.Ordinal) != text.LastIndex(pattern, StringComparison.Ordinal) &&
		len(pattern) < Match_MaxBits-Patch_Margin-Patch_Margin {
		padding += Patch_Margin
		pattern = text.JavaSubstring(math.Max(0, patch.start2-padding),
			math.Min(len(text), patch.start2+patch.length1+padding))
	}
	// Add one chunk for good luck.
	padding += Patch_Margin

	// Add the prefix.
	prefix := text.JavaSubstring(math.Max(0, patch.start2-padding), patch.start2)
	if len(prefix) != 0 {
		patch.diffs.Insert(0, change{DIFF_EQUAL, prefix})
	}
	// Add the suffix.
	suffix := text.JavaSubstring(patch.start2+patch.length1,
		math.Min(len(text), patch.start2+patch.length1+padding))
	if len(suffix) != 0 {
		append(patch.diffs, change{DIFF_EQUAL, suffix})
	}

	// Roll back the start points.
	patch.start1 -= len(prefix)
	patch.start2 -= len(prefix)
	// Extend the lengths.
	patch.length1 += len(prefix) + len(suffix)
	patch.length2 += len(prefix) + len(suffix)
}

/**
 * Compute a list of patches to turn text1 into text2.
 * A set of diffs will be computed.
 * @param text1 Old text.
 * @param text2 New text.
 * @return List of Patch objects.
 */
func (dmp *DiffMatchPatch) patch_make(text1, text2 string) {
	// Check for null inputs not needed since null can't be passed in C#.
	// No diffs provided, comAdde our own.
	diffs = diff_main(text1, text2, true)
	if len(diffs) > 2 {
		diff_cleanupSemantic(diffs)
		diff_cleanupEfficiency(diffs)
	}
	return patch_make(text1, diffs)
}

/**
 * Compute a list of patches to turn text1 into text2.
 * text1 will be derived from the provided diffs.
 * @param diffs Array of Diff objects for text1 to text2.
 * @return List of Patch objects.
 */
func (dmp *DiffMatchPatch) patch_make(diffs) {
	// Check for null inputs not needed since null can't be passed in C#.
	// No origin string provided, comAdde our own.
	text1 := diff_text1(diffs)
	return patch_make(text1, diffs)
}

/**
 * Compute a list of patches to turn text1 into text2.
 * text2 is ignored, diffs are the delta between text1 and text2.
 * @param text1 Old text
 * @param text2 Ignored.
 * @param diffs Array of Diff objects for text1 to text2.
 * @return List of Patch objects.
 * @deprecated Prefer patch_make(string text1, List<Diff> diffs).
 */
func (dmp *DiffMatchPatch) patch_make(text1, text2, diffs []change) {
	return patch_make(text1, diffs)
}

/**
 * Compute a list of patches to turn text1 into text2.
 * text2 is not provided, diffs are the delta between text1 and text2.
 * @param text1 Old text.
 * @param diffs Array of Diff objects for text1 to text2.
 * @return List of Patch objects.
 */
func (dmp *DiffMatchPatch) patch_make(text1, diffs []change) {
	// Check for null inputs not needed since null can't be passed in C#.
	patches := []change{}
	if len(diffs) == 0 {
		return patches // Get rid of the null case.
	}

	patch := Patch{}
	char_count1 := 0 // Number of characters into the text1 string.
	char_count2 := 0 // Number of characters into the text2 string.
	// Start with text1 (prepatch_text) and apply the diffs until we arrive at
	// text2 (postpatch_text). We recreate the patches one by one to determine
	// context info.
	prepatch_text := text1
	postpatch_text := text1

	for _, aDiff := range diffs {
		if len(patch.diffs) == 0 && aDiff.Type != DIFF_EQUAL {
			// A new patch starts here.
			patch.start1 = char_count1
			patch.start2 = char_count2
		}

		switch aDiff.Type {
		case DIFF_INSERT:
			append(patch.diffs, aDiff)
			patch.length2 += aDiff.len(text)
			postpatch_text = postpatch_text.Insert(char_count2, aDiff.text)
			break
		case DIFF_DELETE:
			patch.length1 += aDiff.len(text)
			append(patch.diffs, aDiff)
			postpatch_text = postpatch_text.Remove(char_count2,
				aDiff.len(text))
			break
		case DIFF_EQUAL:
			if aDiff.len(text) <= 2*Patch_Margin &&
				len(patch.diffs) != 0 && aDiff != diffs.Last() {
				// Small equality inside a patch.
				append(patch.diffs, aDiff)
				patch.length1 += aDiff.len(text)
				patch.length2 += aDiff.len(text)
			}

			if aDiff.len(text) >= 2*Patch_Margin {
				// Time for a new patch.
				if len(patch.diffs) != 0 {
					patch_addContext(patch, prepatch_text)
					append(patches, patch)
					patch = Patch{}
					// Unlike Unidiff, our patch lists have a rolling context.
					// http://code.google.com/p/google-diff-match-patch/wiki/Unidiff
					// Update prepatch text & pos to reflect the application of the
					// just completed patch.
					prepatch_text = postpatch_text
					char_count1 = char_count2
				}
			}
			break
		}

		// Update the current character count.
		if aDiff.Type != DIFF_INSERT {
			char_count1 += aDiff.len(text)
		}
		if aDiff.Type != DIFF_DELETE {
			char_count2 += aDiff.len(text)
		}
	}
	// Pick up the leftover patch if not empty.
	if len(patch.diffs) != 0 {
		patch_addContext(patch, prepatch_text)
		append(patches, patch)
	}

	return patches
}

/**
 * Given an array of patches, return another array that is identical.
 * @param patches Array of Patch objects.
 * @return Array of Patch objects.
 */
func (dmp *DiffMatchPatch) patch_deepCopy(patches) {
	patchesCopy = []change{}
	for _, aPatch := range patches {
		patchCopy := Patch{}
		for _, aDiff := range aPatch.diffs {
			diffCopy := change{aDiff.Type, aDiff.text}
			append(patchCopy.diffs, diffCopy)
		}
		patchCopy.start1 = aPatch.start1
		patchCopy.start2 = aPatch.start2
		patchCopy.length1 = aPatch.length1
		patchCopy.length2 = aPatch.length2
		append(patchesCopy, patchCopy)
	}
	return patchesCopy
}

/**
 * Merge a set of patches onto the text.  Return a patched text, as well
 * as an array of true/false values indicating which patches were applied.
 * @param patches Array of Patch objects
 * @param text Old text.
 * @return Two element Object array, containing the new text and an array of
 *      bool values.
 */
func (dmp *DiffMatchPatch) patch_apply(patches, text) (string, []bool) {
	if len(patches) == 0 {
		return text, []bool{}
	}

	// Deep copy the patches so that no changes are made to originals.
	patches = patch_deepCopy(patches)

	nullPadding := dmp.patch_addPadding(patches)
	text = nullPadding + text + nullPadding
	patch_splitMax(patches)

	x := 0
	// delta keeps track of the offset between the expected and actual
	// location of the previous patch.  If there are patches expected at
	// positions 10 and 20, but the first patch was found at 12, delta is 2
	// and the second patch has an effective expected position of 22.
	delta := 0
	results = []bool{patches.Count}
	for _, aPatch := range patches {
		expected_loc := aPatch.start2 + delta
		text1 := diff_text1(aPatch.diffs)
		var start_loc int
		end_loc := -1
		if len(text1) > dmp.Match_MaxBits {
			// patch_splitMax will only provide an oversized pattern
			// in the case of a monster delete.
			start_loc = match_main(text, text1[0:dmp.Match_MaxBits], expected_loc)
			if start_loc != -1 {
				end_loc = match_main(text,
					text1[len(text1)-dmp.Match_MaxBits:], expected_loc+len(text1)-dmp.Match_MaxBits)
				if end_loc == -1 || start_loc >= end_loc {
					// Can't find valid trailing context.  Drop this patch.
					start_loc = -1
				}
			}
		} else {
			start_loc = dmp.match_main(text, text1, expected_loc)
		}
		if start_loc == -1 {
			// No match found.  :(
			results[x] = false
			// Subtract the delta for this failed patch from subsequent patches.
			delta -= aPatch.length2 - aPatch.length1
		} else {
			// Found a match.  :)
			results[x] = true
			delta = start_loc - expected_loc
			var text2 string
			if end_loc == -1 {
				text2 = text.JavaSubstring(start_loc,
					math.Min(start_loc+len(text1), len(text)))
			} else {
				text2 = text.JavaSubstring(start_loc,
					math.Min(end_loc+dmp.Match_MaxBits, len(text)))
			}
			if text1 == text2 {
				// Perfect match, just shove the Replacement text in.
				text = text[0:start_loc] + diff_text2(aPatch.diffs)
				+text[start_loc+len(text1):]
			} else {
				// Imperfect match.  Run a diff to get a framework of equivalent
				// indices.
				diffs = diff_main(text1, text2, false)
				if len(text1) > dmp.Match_MaxBits && dmp.diff_levenshtein(diffs)/float64(len(text1)) > dmp.Patch_DeleteThreshold {
					// The end points match, but the content is unacceptably bad.
					results[x] = false
				} else {
					diff_cleanupSemanticLossless(diffs)
					index1 := 0
					for _, aDiff := range aPatch.diffs {
						if aDiff.Type != DIFF_EQUAL {
							index2 := diff_xIndex(diffs, index1)
							if aDiff.Type == DIFF_INSERT {
								// Insertion
								text = text.Insert(start_loc+index2, aDiff.text)
							} else if aDiff.Type == DIFF_DELETE {
								// Deletion
								text = text.Remove(start_loc+index2, diff_xIndex(diffs,
									index1+aDiff.len(text))-index2)
							}
						}
						if aDiff.Type != DIFF_DELETE {
							index1 += aDiff.len(text)
						}
					}
				}
			}
		}
		x++
	}
	// Strip the padding off.
	text = text[len(nullPadding) : len(text)-2*len(nullPadding)]
	return text, results
}

/**
 * Add some padding on text start and end so that edges can match something.
 * Intended to be called only from within patch_apply.
 * @param patches Array of Patch objects.
 * @return The padding string added to each side.
 */
func (dmp *DiffMatchPatch) patch_addPadding(patches []change) {
	paddingLength := dmp.Patch_Margin
	nullPadding := ""
	for x := 1; x <= paddingLength; x++ {
		nullPadding += strconv.FormatInt(x, 10)
	}

	// Bump all the patches forward.
	for _, aPatch := range patches {
		aPatch.start1 += paddingLength
		aPatch.start2 += paddingLength
	}

	// Add some padding on start of first diff.
	patch := patches[0]
	diffs := patch.diffs
	if len(diffs) == 0 || diffs[0].Type != DIFF_EQUAL {
		// Add nullPadding equality.
		diffs.Insert(0, change{DIFF_EQUAL, nullPadding})
		patch.start1 -= paddingLength // Should be 0.
		patch.start2 -= paddingLength // Should be 0.
		patch.length1 += paddingLength
		patch.length2 += paddingLength
	} else if paddingLength > diffs.First().len(text) {
		// Grow first equality.
		firstDiff := diffs[0]
		extraLength := paddingLength - firstDiff.len(text)
		firstDiff.text = nullPadding[firstDiff.len(text):]
		+firstDiff.text
		patch.start1 -= extraLength
		patch.start2 -= extraLength
		patch.length1 += extraLength
		patch.length2 += extraLength
	}

	// Add some padding on end of last diff.
	patch = patches.Last()
	diffs = patch.diffs
	if len(diffs) == 0 || diffs.Last().Type != DIFF_EQUAL {
		// Add nullPadding equality.
		append(diffs, change{DIFF_EQUAL, nullPadding})
		patch.length1 += paddingLength
		patch.length2 += paddingLength
	} else if paddingLength > diffs.Last().len(text) {
		// Grow last equality.
		lastDiff = diffs[len(diffs)-1]
		extraLength := paddingLength - lastDiff.len(text)
		lastDiff.text += nullPadding[0:extraLength]
		patch.length1 += extraLength
		patch.length2 += extraLength
	}

	return nullPadding
}

/**
 * Look through the patches and break up any which are longer than the
 * maximum limit of the match algorithm.
 * Intended to be called only from within patch_apply.
 * @param patches List of Patch objects.
 */
func (dmp *DiffMatchPatch) patch_splitMax(patches []change) {
	patch_size := dmp.Match_MaxBits
	for x := 0; x < len(patches); x++ {
		if patches[x].length1 <= patch_size {
			continue
		}
		bigpatch := patches[x]
		// Remove the big old patch.
		x = x - 1
		patches = splice(patches, x, 1)
		start1 := bigpatch.start1
		start2 := bigpatch.start2
		precontext := ""
		for len(bigpatch.diffs) != 0 {
			// Create one of several smaller patches.
			patch := Patch{}
			empty := true
			patch.start1 = start1 - len(precontext)
			patch.start2 = start2 - len(precontext)
			if len(precontext) != 0 {
				patch.length1 = len(precontext)
				patch.length2 = len(precontext)
				append(patch.diffs, change{DIFF_EQUAL, precontext})
			}
			for len(bigpatch.diffs) != 0 && patch.length1 < patch_size-dmp.Patch_Margin {
				diff_type := bigpatch.diffs[0].Type
				diff_text := bigpatch.diffs[0].text
				if diff_type == DIFF_INSERT {
					// Insertions are harmless.
					patch.length2 += len(diff_text)
					start2 += len(diff_text)
					append(patch.diffs, bigpatch.diffs.First())
					append(bigpatch.diffs[:0], bigpatch.diffs[0:]...)
					empty = false
				} else if diff_type == DIFF_DELETE && len(patch.diffs) == 1 && patch.diff[0].Type == DIFF_EQUAL && len(diff_text) > 2*patch_size {
					// This is a large deletion.  Let it pass in one chunk.
					patch.length1 += len(diff_text)
					start1 += len(diff_text)
					empty = false
					append(patch.diffs, change{diff_type, diff_text})
					append(bigpatch.diffs[:0], bigpatch.diffs[0:]...)
				} else {
					// Deletion or equality.  Only take as much as we can stomach.
					diff_text = diff_text[0:math.Min(len(diff_text), patch_size-patch.length1-Patch_Margin)]
					patch.length1 += len(diff_text)
					start1 += len(diff_text)
					if diff_type == DIFF_EQUAL {
						patch.length2 += len(diff_text)
						start2 += len(diff_text)
					} else {
						empty = false
					}
					append(patch.diffs, change{diff_type, diff_text})
					if diff_text == bigpatch.diffs[0].text {
						append(bigpatch.diffs[:0], bigpatch.diffs[0:]...)
					} else {
						bigpatch.diffs[0].text =
							bigpatch.diffs[0].text[len(diff_text):]
					}
				}
			}
			// Compute the head context for the next patch.
			precontext = dmp.diff_text2(patch.diffs)
			precontext = precontext[math.Max(0, len(precontext)-dmp.Patch_Margin):]

			postcontext := null
			// Append the end context for this patch.
			if diff_text1(bigpatch.diffs).Length > Patch_Margin {
				postcontext = diff_text1(bigpatch.diffs)[0:Patch_Margin]
			} else {
				postcontext = diff_text1(bigpatch.diffs)
			}

			if len(postcontext) != 0 {
				patch.length1 += len(postcontext)
				patch.length2 += len(postcontext)
				if len(patch.diffs) != 0 && patch.diffs[len(patch.diffs)-1].Type == DIFF_EQUAL {
					patch.diffs[len(patch.diffs)-1].text += postcontext
				} else {
					append(patch.diffs, change{DIFF_EQUAL, postcontext})
				}
			}
			if !empty {
				x = x + 1
				splice(patches, x, 0, patch)
			}
		}
	}
}

/**
 * Take a list of patches and return a textual representation.
 * @param patches List of Patch objects.
 * @return Text representation of patches.
 */
func (dmp *DiffMatchPatch) patch_toText(patches []change) string {
	var text bytes.Buffer
	for _, aPatch := range patches {
		text.WriteString(aPatch)
	}
	return text.String()
}

/**
 * Parse a textual representation of patches and return a List of Patch
 * objects.
 * @param textline Text representation of patches.
 * @return List of Patch objects.
 * @throws ArgumentException If invalid input.
 */
func (dmp *DiffMatchPatch) patch_fromText(textline string) []Patch {
	patches := []Patch{}
	if len(textline) == 0 {
		return patches
	}
	text = string.Split(textline, '\n')
	textPointer := 0
	patch := Patch{}
	patchHeader = regexp.MustCompile("^@@ -(\\d+),?(\\d*) \\+(\\d+),?(\\d*) @@$")
	var sign char
	var line string
	for textPointer < len(text) {
		if !patchHeader.MatchString(text[textPointer]) {
			return log.Fatal("Invalid patch string: " + text[textPointer])
		}
		patch := Patch{}
		append(patches, patch)

		m := patchHeader.FindStringSubmatch(text[textPointer])
		patch.start1 = strconv.ParseInt(m[1], 10, 32)
		if m.Groups[2].Length == 0 {
			patch.start1--
			patch.length1 = 1
		} else if m[2] == "0" {
			patch.length1 = 0
		} else {
			patch.start1--
			patch.length1 = strconv.ParseInt(m[2], 10, 32)
		}

		patch.start2 = strconv.ParseInt(m[3], 10, 32)
		if len(m[4]) == 0 {
			patch.start2--
			patch.length2 = 1
		} else if m[4] == "0" {
			patch.length2 = 0
		} else {
			patch.start2--
			patch.length2 = strconv.ParseInt(m[4], 10, 32)
		}
		textPointer++

		for textPointer < len(text) {
			//try {
			sign = text[textPointer][0]
			/*} catch (IndexOutOfRangeException) {
			  // Blank line?  Whatever.
			  textPointer++
			  continue
			}*/
			line = text[textPointer][1:]
			line = line.Replace("+", "%2b")
			line = url.QueryUnescape(line)
			if sign == '-' {
				// Deletion.
				append(patch.diffs, change{DIFF_DELETE, line})
			} else if sign == '+' {
				// Insertion.
				append(patch.diffs, change{DIFF_INSERT, line})
			} else if sign == ' ' {
				// Minor equality.
				append(patch.diffs, change{DIFF_EQUAL, line})
			} else if sign == '@' {
				// Start of next patch.
				break
			} else {
				// WTF?
				return log.Fatal("Invalid patch mode '" + sign + "' in: " + line)
			}
			textPointer++
		}
	}
	return patches
}

/**
 * Unescape selected chars for compatability with JavaScript's encodeURI.
 * In speed critical applications this could be dropped since the
 * receiving application will certainly decode these fine.
 * Note that this function is case-sensitive.  Thus "%3F" would not be
 * unescaped.  But this is ok because it is only called with the output of
 * HttpUtility.UrlEncode which returns lowercase hex.
 *
 * Example: "%3f" -> "?", "%24" -> "$", etc.
 *
 * @param str The string to escape.
 * @return The escaped string.
 */
func unescapeForEncodeUriCompatability(str string) {
	str = strings.Replace(str, "%21", "!")
	str = strings.Replace(str, "%7e", "~")

	str = strings.Replace(str, "%27", "'")
	str = strings.Replace(str, str, "%28", "(")
	str = strings.Replace(str, str, "%29", ")")

	str = strings.Replace(str, "%3b", ";")
	str = strings.Replace(str, "%2f", "/")
	str = strings.Replace(str, "%3f", "?")

	str = strings.Replace(str, "%3a", ":")
	str = strings.Replace(str, "%40", "@")
	str = strings.Replace(str, "%26", "&")

	str = strings.Replace(str, "%3d", "=")
	str = strings.Replace(str, "%2b", "+")
	str = strings.Replace(str, "%24", "$")

	str = strings.Replace(str, "%2c", ",")
	str = strings.Replace(str, "%23", "#")

	return str
}
