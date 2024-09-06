package diffmatchpatch

import (
	"fmt"
	"strings"
)

// Unified computes the differences between text1 and text2 and formats the differences in the "unified diff" format.
// Optionally pass UnifiedOption to set the new/old labels and context lines.
func (dmp *DiffMatchPatch) Unified(text1, text2 string, opts ...UnifiedOption) string {
	options := newUnifiedOptions(opts)

	text1Enc, text2Enc, lines := dmp.DiffLinesToChars(text1, text2)

	diffs := dmp.DiffMain(text1Enc, text2Enc, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)

	unified := toUnified(diffs, options)

	return unified.String()
}

// DiffUnified formats the diffs slice in the "unified diff" format.
// Optionally pass UnifiedOption to set the new/old labels and context lines.
func (dmp *DiffMatchPatch) DiffUnified(diffs []Diff, opts ...UnifiedOption) string {
	options := newUnifiedOptions(opts)

	u := toUnified(diffs, options)

	return u.String()
}

// toUnified takes a []Diff slice and converts into into a unified struct, which
// can then be used to produce the unified diff output using its String()
// method.
func toUnified(diffs []Diff, opts unifiedOptions) unified {
	maxCtx := opts.contextLines * 2
	u := unified{
		label1: opts.text1Label,
		label2: opts.text2Label,
	}

	if isEqual(diffs) {
		return u
	}

	diffs = diffLinewise(diffs)

	var (
		h *hunk

		lineNo1 int
		lineNo2 int
		context []Diff
	)
	for _, diff := range diffs {
		switch diff.Type {
		case DiffDelete:
			lineNo1++
		case DiffInsert:
			lineNo2++
		case DiffEqual:
			lineNo1++
			lineNo2++
		}

		if diff.Type == DiffEqual {
			context = append(context, diff)
			continue
		}

		// close previous hunk
		if h != nil && len(context) > maxCtx {
			cl := len(context)
			if cl > opts.contextLines {
				cl = opts.contextLines
			}

			h.diffs = append(h.diffs, context[:cl]...)

			u.hunks = append(u.hunks, *h)
			h = nil
		}

		// start new hunk
		if h == nil {
			cl := len(context)
			if cl > opts.contextLines {
				cl = opts.contextLines
			}

			l1 := lineNo1 - cl
			l2 := lineNo2 - cl

			// When starting a new hunk, the line number for lineNo1 XOR lineNo2
			// as already been advanced, but not the other. Account for that in
			// l1 or l2.
			switch diff.Type {
			case DiffDelete:
				l2++
			case DiffInsert:
				l1++
			}

			h = &hunk{
				fromLine: l1,
				toLine:   l2,
				diffs:    context[len(context)-cl:],
			}

			context = nil
		}

		h.diffs = append(h.diffs, context...)
		context = nil

		h.diffs = append(h.diffs, diff)
	}

	// close last hunk
	if h != nil {
		cl := len(context)
		if cl > opts.contextLines {
			cl = opts.contextLines
		}

		h.diffs = append(h.diffs, context[:cl]...)

		u.hunks = append(u.hunks, *h)
		h = nil
	}

	return u
}

func isEqual(diffs []Diff) bool {
	for _, diff := range diffs {
		if diff.Type != DiffEqual {
			return false
		}
	}

	return true
}

// diffLinewise splits and merged diffs so that each individual diff represents one line, including the final newline character.
func diffLinewise(diffs []Diff) []Diff {
	var (
		ret          []Diff
		line1, line2 string
	)

	diffs = diffCleanupNewline(diffs)

	add := func(d Diff) {
		switch d.Type {
		case DiffDelete:
			line1 = line1 + d.Text
		case DiffInsert:
			line2 = line2 + d.Text
		default: // equal
			line1 = line1 + d.Text
			line2 = line2 + d.Text
		}

		if strings.HasSuffix(line1, "\n") && line1 == line2 {
			ret = append(ret, Diff{
				Type: DiffEqual,
				Text: line1,
			})

			line1, line2 = "", ""
		}

		if strings.HasSuffix(line1, "\n") {
			ret = append(ret, Diff{
				Type: DiffDelete,
				Text: line1,
			})

			line1 = ""
		}

		if strings.HasSuffix(line2, "\n") {
			ret = append(ret, Diff{
				Type: DiffInsert,
				Text: line2,
			})

			line2 = ""
		}
	}

	for _, diff := range diffs {
		for _, segment := range strings.SplitAfter(diff.Text, "\n") {
			add(Diff{
				Type: diff.Type,
				Text: segment,
			})
		}
	}

	// line1 and/or line2 may be non-empty if there is no newline at the end of file.
	if line1 != "" && line1 == line2 {
		ret = append(ret, Diff{
			Type: DiffEqual,
			Text: line1,
		})

		line1, line2 = "", ""
	}

	if line1 != "" {
		ret = append(ret, Diff{
			Type: DiffDelete,
			Text: line1,
		})

		line1 = ""
	}

	if line2 != "" {
		ret = append(ret, Diff{
			Type: DiffInsert,
			Text: line2,
		})

		line2 = ""
	}

	return reorderDeletionsFirst(ret)
}

// diffCleanupNewline looks for single edits surrounded on both sides by equalities which can be shifted sideways to align on newlines.
func diffCleanupNewline(diffs []Diff) []Diff {
	var ret []Diff

	for i := 0; i < len(diffs); i++ {
		if i < len(diffs)-2 && diffs[i].Type == DiffEqual && diffs[i+1].Type != DiffEqual && diffs[i+2].Type == DiffEqual {
			common := prefixWithNewline(diffs[i+1].Text, diffs[i+2].Text)

			// Convert ["=<equal>", "±<common\n><change>", "=<common\n><equal>"]
			// to ["=<equal><common\n>", "±<change><common\n>", "=<equal>"]
			if common != "" {
				ret = append(ret,
					Diff{
						Type: DiffEqual,
						Text: diffs[i].Text + common,
					},
					Diff{
						Type: diffs[i+1].Type,
						Text: strings.TrimPrefix(diffs[i+1].Text, common) + common,
					},
					Diff{
						Type: DiffEqual,
						Text: strings.TrimPrefix(diffs[i+2].Text, common),
					},
				)

				i += 2
				continue
			}
		}

		ret = append(ret, diffs[i])
	}

	return ret
}

// prefixWithNewline returns the longest common prefix between text1 and text2, up to and including a newline character.
// If text1 and text2 do not have a common prefix, or the common prefix does not include a newline character, the empty string is returned.
func prefixWithNewline(text1, text2 string) string {
	prefix := New().DiffCommonPrefix(text1, text2)

	index := strings.LastIndex(text1[:prefix], "\n")
	if index != -1 {
		return text1[:index+1]
	}

	return ""
}

// reorderDeletionsFirst reorders changes so that deletions come before insertions, without crossing an equality boundary.
func reorderDeletionsFirst(diffs []Diff) []Diff {
	var (
		ret        []Diff
		deletions  []Diff
		insertions []Diff
	)

	for _, diff := range diffs {
		switch diff.Type {
		case DiffDelete:
			deletions = append(deletions, diff)
		case DiffInsert:
			insertions = append(insertions, diff)
		case DiffEqual:
			ret = append(ret, deletions...)
			deletions = nil

			ret = append(ret, insertions...)
			insertions = nil

			ret = append(ret, diff)
		}
	}

	ret = append(ret, deletions...)
	ret = append(ret, insertions...)

	return ret
}

// unified represents modifications in a form conducive to printing a unified diff.
type unified struct {
	label1, label2 string
	hunks          []hunk
}

// hunk is a list of nearby changes, deperated by at most 2*contextLines lines.
type hunk struct {
	// The line in the original source where the hunk starts.
	fromLine int
	// The line in the original source where the hunk finishes.
	toLine int
	// List of modifications. Each Diff represents one deleted, inserted, or equal line.
	diffs []Diff
}

// numLines returns the number of lines in the hunk for text1 and text2.
func (h hunk) numLines() (n1, n2 int) {
	for _, diff := range h.diffs {
		switch diff.Type {
		case DiffDelete:
			n1++
		case DiffInsert:
			n2++
		case DiffEqual:
			n1++
			n2++
		}
	}

	return n1, n2
}

func (h hunk) String() string {
	var b strings.Builder

	fmt.Fprint(&b, "@@")

	numLines1, numLines2 := h.numLines()

	switch {
	case numLines1 > 1:
		fmt.Fprintf(&b, " -%d,%d", h.fromLine, numLines1)
	case h.fromLine == 1 && numLines1 == 0:
		// Mimic GNU diff -u behavior when adding to empty file.
		fmt.Fprintf(&b, " -0,0")
	default:
		fmt.Fprintf(&b, " -%d", h.fromLine)
	}

	switch {
	case numLines2 > 1:
		fmt.Fprintf(&b, " +%d,%d", h.toLine, numLines2)
	case h.toLine == 1 && numLines2 == 0:
		// Mimic GNU diff -u behavior when adding to empty file.
		fmt.Fprintf(&b, " +0,0")
	default:
		fmt.Fprintf(&b, " +%d", h.toLine)
	}

	fmt.Fprint(&b, " @@\n")

	for _, diff := range h.diffs {
		switch diff.Type {
		case DiffDelete:
			fmt.Fprintf(&b, "-%s", diff.Text)
		case DiffInsert:
			fmt.Fprintf(&b, "+%s", diff.Text)
		default:
			fmt.Fprintf(&b, " %s", diff.Text)
		}
		if !strings.HasSuffix(diff.Text, "\n") {
			fmt.Fprintf(&b, "\n\\ No newline at end of file\n")
		}
	}

	return b.String()
}

// String converts a unified diff to the standard textual form for that diff.
// The output of this function can be passed to tools like patch.
func (u unified) String() string {
	if len(u.hunks) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", u.label1)
	fmt.Fprintf(&b, "+++ %s\n", u.label2)
	for _, hunk := range u.hunks {
		fmt.Fprint(&b, hunk)
	}

	return b.String()
}

// DefaultContextLines is the number of unchanged lines of surrounding
// context displayed by Unified.
const DefaultContextLines = 3

// UnifiedOption is an option for DiffUnified().
type UnifiedOption func(*unifiedOptions)

type unifiedOptions struct {
	contextLines int
	text1Label   string
	text2Label   string
}

func newUnifiedOptions(opts []UnifiedOption) unifiedOptions {
	ret := unifiedOptions{
		contextLines: DefaultContextLines,
		text1Label:   "text1",
		text2Label:   "text2",
	}

	for _, o := range opts {
		o(&ret)
	}

	return ret
}

// UnifiedContextLines sets the number of unchanged lines of surrounding context
// printed. Defaults to DefaultContextLines.
func UnifiedContextLines(lines int) UnifiedOption {
	if lines <= 0 {
		lines = DefaultContextLines
	}

	return func(o *unifiedOptions) {
		o.contextLines = lines
	}
}

// UnifiedLabels sets the labels for the old and new files. Defaults to "text1" and "text2".
func UnifiedLabels(oldLabel, newLabel string) UnifiedOption {
	return func(o *unifiedOptions) {
		o.text1Label = oldLabel
		o.text2Label = newLabel
	}
}
