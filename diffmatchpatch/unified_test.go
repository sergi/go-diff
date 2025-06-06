package diffmatchpatch_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestDiffUnified(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		text1 string
		text2 string
		want  string
	}{
		{
			name:  "No changes",
			text1: "Hello, world!\n",
			text2: "Hello, world!\n",
			want:  "",
		},
		{
			name:  "Insertion at beginning",
			text1: "Hello, world!\n",
			text2: "New line\nHello, world!\n",
			want:  "--- text1\n+++ text2\n@@ -1 +1,2 @@\n+New line\n Hello, world!\n",
		},
		{
			name:  "Insertion at end",
			text1: "Hello, world!\n",
			text2: "Hello, world!\nNew line\n",
			want:  "--- text1\n+++ text2\n@@ -1 +1,2 @@\n Hello, world!\n+New line\n",
		},
		{
			name:  "Insertion middle",
			text1: "Hello, world!\nHello, world!\n",
			text2: "Hello, world!\nNew line\nHello, world!\n",
			want:  "--- text1\n+++ text2\n@@ -1,2 +1,3 @@\n Hello, world!\n+New line\n Hello, world!\n",
		},
		{
			name:  "Removal at beginning",
			text1: "Old line\nHello, world!\n",
			text2: "Hello, world!\n",
			want:  "--- text1\n+++ text2\n@@ -1,2 +1 @@\n-Old line\n Hello, world!\n",
		},
		{
			name:  "Removal at end",
			text1: "Hello, world!\nOld line\n",
			text2: "Hello, world!\n",
			want:  "--- text1\n+++ text2\n@@ -1,2 +1 @@\n Hello, world!\n-Old line\n",
		},
		{
			name:  "Removal middle",
			text1: "Hello, world!\nOld line\nHello, world!\n",
			text2: "Hello, world!\nHello, world!\n",
			want:  "--- text1\n+++ text2\n@@ -1,3 +1,2 @@\n Hello, world!\n-Old line\n Hello, world!\n",
		},
		{
			name:  "Replacement",
			text1: "Prefix\nHello, world!\nSuffix\n",
			text2: "Prefix\nHello, Golang!\nSuffix\n",
			want:  "--- text1\n+++ text2\n@@ -1,3 +1,3 @@\n Prefix\n-Hello, world!\n+Hello, Golang!\n Suffix\n",
		},
		{
			name:  "Insertion",
			text1: makeContext(10, 0),
			text2: makeContext(5, 0) + "INSERTION\n" + makeContext(5, 5),
			want:  "--- text1\n+++ text2\n@@ -3,6 +3,7 @@\n context2\n context3\n context4\n+INSERTION\n context5\n context6\n context7\n",
		},
		{
			name:  "Multiple hunks",
			text1: makeContext(20, 0),
			text2: makeContext(5, 0) + "INSERTION1\n" + makeContext(10, 5) + "INSERTION2\n" + makeContext(5, 15),
			want: `--- text1
+++ text2
@@ -3,6 +3,7 @@
 context2
 context3
 context4
+INSERTION1
 context5
 context6
 context7
@@ -13,6 +14,7 @@
 context12
 context13
 context14
+INSERTION2
 context15
 context16
 context17
`,
		},
		{
			name:  "Merge hunk with <= 5 lines of context",
			text1: makeContext(15, 0),
			text2: makeContext(5, 0) + "INSERTION1\n" + makeContext(5, 5) + "INSERTION2\n" + makeContext(5, 10),
			want: `--- text1
+++ text2
@@ -3,11 +3,13 @@
 context2
 context3
 context4
+INSERTION1
 context5
 context6
 context7
 context8
 context9
+INSERTION2
 context10
 context11
 context12
`,
		},
		{
			name:  "Insert without newline",
			text1: "context1",
			text2: "context1\nnew line",
			want: `--- text1
+++ text2
@@ -1 +1,2 @@
-context1
\ No newline at end of file
+context1
+new line
\ No newline at end of file
`,
		},
		{
			name:  "Removal without newline",
			text1: "context1\nold line",
			text2: "context1",
			want: `--- text1
+++ text2
@@ -1,2 +1 @@
-context1
-old line
\ No newline at end of file
+context1
\ No newline at end of file
`,
		},
		{
			name:  "context without newline",
			text1: "context0\nold1\ncontext1",
			text2: "context0\nnew1\ncontext1",
			want: `--- text1
+++ text2
@@ -1,3 +1,3 @@
 context0
-old1
+new1
 context1
\ No newline at end of file
`,
		},
		{
			name:  "Replace multiple subsequent lines",
			text1: makeContext(5, 0) + "old1\nold2\nold3\n" + makeContext(5, 5),
			text2: makeContext(5, 0) + "new1\nnew2\nnew3\n" + makeContext(5, 5),
			want: `--- text1
+++ text2
@@ -3,9 +3,9 @@
 context2
 context3
 context4
-old1
-old2
-old3
+new1
+new2
+new3
 context5
 context6
 context7
`,
		},
		{
			name:  "empty text1",
			text1: "",
			text2: "new1\n",
			want: `--- text1
+++ text2
@@ -0,0 +1 @@
+new1
`,
		},
		{
			name:  "empty text2",
			text1: "old1\n",
			text2: "",
			want: `--- text1
+++ text2
@@ -1 +0,0 @@
-old1
`,
		},
	}

	for _, tc := range cases {
		// Un-alias tc for compatibility with Go <1.22.
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dmp := diffmatchpatch.New()

			got := dmp.Unified(tc.text1, tc.text2, diffmatchpatch.UnifiedLabels("text1", "text2"))

			t.Logf("dmp.Unified() =\n%s", got)

			if got != tc.want {
				t.Errorf("Unified() output differs (-want/+got):\n%s", cmp.Diff(tc.want, got))
			}

			// DiffLinesToChars / DiffCharsToLines is not required for correct results.
			diffs := dmp.DiffMain(tc.text1, tc.text2, false)

			got = dmp.DiffUnified(diffs, diffmatchpatch.UnifiedLabels("text1", "text2"), diffmatchpatch.UnifiedContextLines(3))
			if got != tc.want {
				t.Errorf("DiffUnified() output differs (-want/+got):\n%s", cmp.Diff(tc.want, got))
			}

		})
	}
}

func makeContext(n, start int) string {
	var b strings.Builder

	for i := start; i < start+n; i++ {
		fmt.Fprintf(&b, "context%d\n", i)
	}

	return b.String()
}

func ExampleDiffMatchPatch_DiffUnified() {
	text1 := "Prefix\nHello, world!\nSuffix\n"
	text2 := "Prefix\nHello, Golang!\nSuffix\n"

	dmp := diffmatchpatch.New()

	// Pre-process the inputs so that each codepoint in text[12]End represents one line.
	text1Enc, text2Enc, lines := dmp.DiffLinesToChars(text1, text2)

	// Run the diff algorithm on the preprocessed inputs.
	diffs := dmp.DiffMain(text1Enc, text2Enc, false)

	// Expand the diffs back into the full lines they represent.
	diffs = dmp.DiffCharsToLines(diffs, lines)

	// Format as unified diff.
	unifiedDiff := dmp.DiffUnified(diffs,
		diffmatchpatch.UnifiedLabels("old.txt", "new.txt"),
		diffmatchpatch.UnifiedContextLines(3))

	fmt.Print(unifiedDiff)
	// Output:
	// --- old.txt
	// +++ new.txt
	// @@ -1,3 +1,3 @@
	//  Prefix
	// -Hello, world!
	// +Hello, Golang!
	//  Suffix
}
