// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's Diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

package diffmatchpatch_test

import (
	"fmt"
	"log"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Compare two texts and print the differences.
func ExampleDiffMatchPatch_DiffMain() {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain("Lorem ipsum dolor.", "Lorem dolor sit amet.", false)

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			fmt.Print("[+]", diff.Text)
		case diffmatchpatch.DiffDelete:
			fmt.Print("[-]", diff.Text)
		case diffmatchpatch.DiffEqual:
			fmt.Print("[=]", diff.Text)
		}
	}

	// Output: [=]Lorem [-]ipsum [=]dolor[+] sit amet[=].
}

// The diffs are plain data; the source and target texts can be
// reconstructed from them without any pretty printing.
func ExampleDiffMatchPatch_DiffText1() {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain("Hello world.", "Hello goodbye world.", false)

	fmt.Println("source:", dmp.DiffText1(diffs))
	fmt.Println("target:", dmp.DiffText2(diffs))

	// Output:
	// source: Hello world.
	// target: Hello goodbye world.
}

// Diff two texts line by line by reducing each line to a character,
// diffing the reduced texts and expanding the result back to lines.
func ExampleDiffMatchPatch_DiffLinesToChars() {
	dmp := diffmatchpatch.New()

	text1 := "foo\nbar\nbaz\n"
	text2 := "foo\nbaz\nqux\n"

	chars1, chars2, lines := dmp.DiffLinesToChars(text1, text2)
	diffs := dmp.DiffMain(chars1, chars2, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)

	for _, diff := range diffs {
		fmt.Printf("%s %q\n", diff.Type, diff.Text)
	}

	// Output:
	// Equal "foo\n"
	// Delete "bar\n"
	// Equal "baz\n"
	// Insert "qux\n"
}

// Make a patch, convert it to text, and apply that text to the
// original later, possibly elsewhere.
func ExampleDiffMatchPatch_PatchMake() {
	dmp := diffmatchpatch.New()

	text1 := "hello world"
	text2 := "goodbye world"

	patchText := dmp.PatchToText(dmp.PatchMake(text1, text2))
	fmt.Print(patchText)

	patches, err := dmp.PatchFromText(patchText)
	if err != nil {
		log.Fatal(err)
	}
	result, applied := dmp.PatchApply(patches, text1)

	fmt.Println(applied)
	fmt.Println(result)

	// Output:
	// @@ -1,9 +1,11 @@
	// -hello
	// +goodbye
	//   wor
	// [true]
	// goodbye world
}

// Locate a pattern in the text, exactly or fuzzily.
func ExampleDiffMatchPatch_MatchMain() {
	dmp := diffmatchpatch.New()

	text := "The quick brown fox jumps over the lazy dog."

	fmt.Println(dmp.MatchMain(text, "quick brown", 0))
	fmt.Println(dmp.MatchMain(text, "quick grown", 0)) // typo, still found

	// Output:
	// 4
	// 4
}

// Cleanup shifts edits to word boundaries: the raw diff on the left
// splits words, the cleaned up diff on the right edits whole words.
func ExampleDiffMatchPatch_DiffCleanupSemantic() {
	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain("and one two.", "and one one.", false)

	for _, diff := range diffs {
		fmt.Printf("%s %q\n", diff.Type, diff.Text)
	}
	fmt.Println("---")
	for _, diff := range dmp.DiffCleanupSemantic(diffs) {
		fmt.Printf("%s %q\n", diff.Type, diff.Text)
	}

	// Output:
	// Equal "and one "
	// Delete "tw"
	// Equal "o"
	// Insert "ne"
	// Equal "."
	// ---
	// Equal "and one "
	// Delete "two"
	// Insert "one"
	// Equal "."
}
