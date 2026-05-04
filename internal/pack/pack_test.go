package pack_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icco/bugsim/internal/pack"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newImplementPack(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
}

func TestLoadImplement(t *testing.T) {
	dir := t.TempDir()
	newImplementPack(t, dir)
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.Manifest.ID != "demo" {
		t.Fatalf("id = %q", p.Manifest.ID)
	}
	if p.Manifest.Track != pack.TrackImplement {
		t.Fatalf("track = %q", p.Manifest.Track)
	}
}

func TestLoadImplementMissingSkeleton(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error for missing skeleton/")
	}
}

func TestLoadBugReview(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo-bug
title: Demo Bug
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick one
  choices:
    - id: a
      label: choice a
      correct: false
    - id: b
      label: choice b
      correct: true
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	writeFile(t, filepath.Join(dir, "bug", "snippet.go"), "package x\n")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.Manifest.BugReview == nil || len(p.Manifest.BugReview.Choices) != 2 {
		t.Fatalf("bug review parse failed")
	}
}

func TestBugReviewRequiresExactlyOneCorrect(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: bad
title: Bad
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick one
  choices:
    - id: a
      label: a
      correct: true
    - id: b
      label: b
      correct: true
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "x")
	writeFile(t, filepath.Join(dir, "bug", "x.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: more than one correct")
	}
}

func TestUnsupportedFormatVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 99
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: unsupported version")
	}
}

func TestLoadMissingManifest(t *testing.T) {
	dir := t.TempDir()
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: manifest.yaml missing")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	// Deliberately broken YAML — load must surface a parse error instead
	// of panicking, and the error must mention parsing.
	writeFile(t, filepath.Join(dir, "manifest.yaml"), "id: [unterminated\n")
	_, err := pack.Load(dir)
	if err == nil {
		t.Fatal("expected error: invalid YAML")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

// TestManifestValidationErrors is a single table-driven test that covers
// every manifest-validation rule. It replaces a wall of nearly-identical
// "missing X" tests and makes it cheap to add new rules.
func TestManifestValidationErrors(t *testing.T) {
	cases := []struct {
		name      string
		manifest  string
		hasFiles  []string // extra files to drop into the pack dir
		wantSubst string   // a substring expected somewhere in err.Error()
	}{
		{
			name: "missing problem.md",
			manifest: `pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`,
			hasFiles:  []string{"skeleton/x.go", "hidden_tests/x_test.go"},
			wantSubst: "problem.md",
		},
		{
			name: "empty title",
			manifest: `pack_format_version: 1
id: demo
title: ""
track: implement
runner: go
difficulty: easy
`,
			hasFiles:  []string{"problem.md", "skeleton/x.go", "hidden_tests/x_test.go"},
			wantSubst: "title",
		},
		{
			name: "invalid track",
			manifest: `pack_format_version: 1
id: demo
title: Demo
track: nonsense
runner: go
difficulty: easy
`,
			hasFiles:  []string{"problem.md"},
			wantSubst: "track",
		},
		{
			name: "invalid difficulty",
			manifest: `pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: extreme
`,
			hasFiles:  []string{"problem.md", "skeleton/x.go", "hidden_tests/x_test.go"},
			wantSubst: "difficulty",
		},
		{
			name: "implement missing hidden_tests/",
			manifest: `pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`,
			hasFiles:  []string{"problem.md", "skeleton/x.go"},
			wantSubst: "hidden_tests",
		},
		{
			name: "implement missing runner",
			manifest: `pack_format_version: 1
id: demo
title: Demo
track: implement
difficulty: easy
`,
			hasFiles:  []string{"problem.md", "skeleton/x.go", "hidden_tests/x_test.go"},
			wantSubst: "runner",
		},
		{
			name: "bug_review missing bug/",
			manifest: `pack_format_version: 1
id: demo-bug
title: Demo
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick
  choices:
    - id: a
      label: a
      correct: true
    - id: b
      label: b
      correct: false
`,
			hasFiles:  []string{"problem.md"},
			wantSubst: "bug",
		},
		{
			name: "bug_review empty prompt",
			manifest: `pack_format_version: 1
id: demo-bug
title: Demo
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: ""
  choices:
    - id: a
      label: a
      correct: true
    - id: b
      label: b
      correct: false
`,
			hasFiles:  []string{"problem.md", "bug/x.txt"},
			wantSubst: "prompt",
		},
		{
			name: "bug_review with too few choices",
			manifest: `pack_format_version: 1
id: demo-bug
title: Demo
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick
  choices:
    - id: a
      label: a
      correct: true
`,
			hasFiles:  []string{"problem.md", "bug/x.txt"},
			wantSubst: "at least 2",
		},
		{
			name: "missing runner on bug_review",
			manifest: `pack_format_version: 1
id: demo-bug
title: Demo
track: bug_review
difficulty: easy
bug_review:
  prompt: pick
  choices:
    - id: a
      label: a
      correct: true
    - id: b
      label: b
      correct: false
`,
			hasFiles:  []string{"problem.md", "bug/x.txt"},
			wantSubst: "runner",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "manifest.yaml"), tc.manifest)
			for _, f := range tc.hasFiles {
				writeFile(t, filepath.Join(dir, f), "x\n")
			}
			_, err := pack.Load(dir)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSubst)
			}
			if !strings.Contains(err.Error(), tc.wantSubst) {
				t.Fatalf("expected error to mention %q, got: %v", tc.wantSubst, err)
			}
		})
	}
}

func TestIsTemplatePack(t *testing.T) {
	if !pack.IsTemplatePack("_template-implement") {
		t.Fatal("_template-implement should be a template")
	}
	if pack.IsTemplatePack("sum-positive") {
		t.Fatal("sum-positive should not be a template")
	}
}

func TestDiscoverEmpty(t *testing.T) {
	root := t.TempDir()
	got, err := pack.Discover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

// TestReadBugCorpusIncludesPolyglotExtensions verifies the corpus reader's
// allow-list across every supported extension. Each row exercises one
// extension; if a future runner needs another (.rs, .py), add it here.
func TestReadBugCorpusIncludesPolyglotExtensions(t *testing.T) {
	want := []struct {
		filename string
		body     string
	}{
		{"notes.md", "# heading"},
		{"context.txt", "ascii"},
		{"snippet.go", "package x"},
		{"data.json", `{"k":1}`},
		{"config.yaml", "k: 1"},
		{"more.yml", "k: 2"},
		{"uploader.ts", "export const ok = 1;"},
		{"react.tsx", "export const C = () => null;"},
		{"helper.js", "module.exports = 1;"},
		{"esm.mjs", "export default 1;"},
		{"cjs.cjs", "module.exports = 1;"},
	}
	dir := newCorpusPack(t, "all-extensions")
	for _, w := range want {
		writeFile(t, filepath.Join(dir, "bug", w.filename), w.body)
	}
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	corpus, err := p.ReadBugCorpus()
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	for _, w := range want {
		t.Run(w.filename, func(t *testing.T) {
			if !strings.Contains(corpus, w.filename) {
				t.Fatalf("missing %q header in corpus", w.filename)
			}
			if !strings.Contains(corpus, w.body) {
				t.Fatalf("missing body of %q in corpus", w.filename)
			}
		})
	}
}

// TestReadBugCorpusExcludesBinary asserts that binary-ish or unknown
// extensions are silently skipped — pasting a screenshot or .pdf into a
// pack must not pollute the markdown view.
func TestReadBugCorpusExcludesBinary(t *testing.T) {
	dir := newCorpusPack(t, "exclude-binary")
	writeFile(t, filepath.Join(dir, "bug", "screenshot.png"), "fake-binary")
	writeFile(t, filepath.Join(dir, "bug", "report.pdf"), "%PDF-1.4")
	writeFile(t, filepath.Join(dir, "bug", "trace.bin"), "\x00\x01\x02")
	writeFile(t, filepath.Join(dir, "bug", "kept.md"), "shown")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	corpus, err := p.ReadBugCorpus()
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	for _, name := range []string{"screenshot.png", "report.pdf", "trace.bin"} {
		if strings.Contains(corpus, name) {
			t.Fatalf("did not expect %q to appear in corpus, got:\n%s", name, corpus)
		}
	}
	if !strings.Contains(corpus, "kept.md") {
		t.Fatalf("expected kept.md to appear in corpus, got:\n%s", corpus)
	}
}

// TestReadBugCorpusReturnsRelativePaths makes sure file headers are
// relative to bug/, not absolute — otherwise the markdown leaks the
// reviewer's working directory.
func TestReadBugCorpusReturnsRelativePaths(t *testing.T) {
	dir := newCorpusPack(t, "relpath")
	writeFile(t, filepath.Join(dir, "bug", "nested", "deep.md"), "deep")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	corpus, err := p.ReadBugCorpus()
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	wantHeader := "nested" + string(filepath.Separator) + "deep.md"
	if !strings.Contains(corpus, wantHeader) {
		t.Fatalf("expected relative header %q in corpus, got:\n%s", wantHeader, corpus)
	}
	if strings.Contains(corpus, dir) {
		t.Fatalf("corpus leaked absolute path %q", dir)
	}
}

func TestPackTagsAndRecommendedMinutesParse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
tags: [go, basics, polyglot]
recommended_minutes: 12
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "x")
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := p.Manifest.Tags; len(got) != 3 || got[2] != "polyglot" {
		t.Fatalf("tags = %v", got)
	}
	if p.Manifest.RecommendedMinutes == nil || *p.Manifest.RecommendedMinutes != 12 {
		t.Fatalf("recommended_minutes = %v", p.Manifest.RecommendedMinutes)
	}
}

// TestDiscoverPropagatesBrokenPack ensures one bad pack in a directory
// fails the whole discovery — this is the right behaviour for the
// `bugsim list` / `verify-pack` workflow because a silent skip would hide
// authoring bugs.
func TestDiscoverPropagatesBrokenPack(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good")
	newImplementPack(t, good)
	bad := filepath.Join(root, "bad")
	writeFile(t, filepath.Join(bad, "manifest.yaml"), "not: [valid yaml\n")
	_, err := pack.Discover(root)
	if err == nil {
		t.Fatal("expected discover to surface the bad pack's error")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Fatalf("expected error to mention the bad pack dir, got: %v", err)
	}
}

// TestPackValidateRejectsProblemMDDirectory catches a tricky bug: if
// problem.md is a directory rather than a file, os.ReadFile returns an
// error but its message ("is a directory") should still propagate.
func TestPackValidateRejectsProblemMDDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`)
	if err := os.MkdirAll(filepath.Join(dir, "problem.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	_, err := pack.Load(dir)
	if err == nil {
		t.Fatal("expected error: problem.md is a directory")
	}
	if !strings.Contains(err.Error(), "problem.md") {
		t.Fatalf("expected error to mention problem.md, got: %v", err)
	}
}

// TestBugReviewRequiresChoiceIDAndLabel makes sure choices missing either
// `id` or `label` are rejected at load time. Without this, an MCQ with
// an unrenderable empty option would slip through to the TUI.
func TestBugReviewRequiresChoiceIDAndLabel(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "empty id",
			body: `pack_format_version: 1
id: bad
title: Bad
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick
  choices:
    - id: ""
      label: a
      correct: true
    - id: b
      label: b
      correct: false
`,
		},
		{
			name: "empty label",
			body: `pack_format_version: 1
id: bad
title: Bad
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick
  choices:
    - id: a
      label: ""
      correct: true
    - id: b
      label: b
      correct: false
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "manifest.yaml"), tc.body)
			writeFile(t, filepath.Join(dir, "problem.md"), "x")
			writeFile(t, filepath.Join(dir, "bug", "x.txt"), "x")
			if _, err := pack.Load(dir); err == nil {
				t.Fatal("expected error: choice missing id/label")
			}
		})
	}
}

// FuzzLoadManifestNeverPanics fuzzes the manifest YAML body and asserts
// that pack.Load either succeeds or returns an error — never panics. This
// is exactly the kind of thing fuzzing buys you for cheap: pack authors
// will paste in malformed YAML, and we don't want bugsim to crash.
func FuzzLoadManifestNeverPanics(f *testing.F) {
	seeds := []string{
		"",
		"id: foo",
		"pack_format_version: 1\nid: x\ntitle: X\ntrack: implement\nrunner: go\ndifficulty: easy\n",
		"pack_format_version: 99\n",
		"!!! not yaml :::",
		"a: [1, 2, 3, [4, [5, [6]]]]",
		"track: \"bug_review\"\nbug_review:\n  prompt: \"\"\n  choices: []\n",
		strings.Repeat("k: v\n", 1000),
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		// problem.md and skeleton/hidden_tests are missing by design — we
		// only care that Load doesn't panic on hostile YAML; an error is
		// fine and expected for nearly every input.
		_, _ = pack.Load(dir)
	})
}

func TestDiscoverIgnoresTemplates(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "good")
	newImplementPack(t, good)
	tmpl := filepath.Join(root, "_template")
	newImplementPack(t, tmpl)
	junk := filepath.Join(root, "junk")
	if err := os.MkdirAll(junk, 0o755); err != nil {
		t.Fatal(err)
	}

	found, err := pack.Discover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(found) != 1 || found[0].Dir != good {
		t.Fatalf("discover returned %+v", found)
	}
}

// newCorpusPack builds a minimal valid bug_review pack directory with the
// given id and returns its path. Tests that exercise ReadBugCorpus drop
// extra files into <dir>/bug/.
func newCorpusPack(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: `+id+`
title: Demo
track: bug_review
runner: go
difficulty: easy
bug_review:
  prompt: pick
  choices:
    - id: a
      label: a
      correct: true
    - id: b
      label: b
      correct: false
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	// At least one file is needed under bug/ so the directory exists.
	writeFile(t, filepath.Join(dir, "bug", ".keep"), "")
	return dir
}
