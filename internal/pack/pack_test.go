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
	writeFile(t, filepath.Join(dir, "manifest.yaml"), "id: [unterminated\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: invalid YAML")
	}
}

func TestLoadMissingProblemMD(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: problem.md missing")
	}
}

func TestLoadEmptyTitle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: ""
track: implement
runner: go
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "x")
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: empty title")
	}
}

func TestLoadInvalidTrack(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: nonsense
runner: go
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "x")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: invalid track")
	}
}

func TestLoadInvalidDifficulty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
runner: go
difficulty: extreme
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "x")
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: invalid difficulty")
	}
}

func TestLoadImplementMissingHiddenTests(t *testing.T) {
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
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: hidden_tests/ missing")
	}
}

func TestLoadImplementMissingRunner(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo
title: Demo
track: implement
difficulty: easy
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	writeFile(t, filepath.Join(dir, "skeleton", "x.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "hidden_tests", "x_test.go"), "package x\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: implement requires runner")
	}
}

func TestLoadBugReviewMissingBug(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo-bug
title: Demo
track: bug_review
difficulty: easy
bug_review:
  prompt: pick one
  choices:
    - id: a
      label: a
      correct: true
    - id: b
      label: b
      correct: false
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: bug/ missing")
	}
}

func TestLoadBugReviewMissingPrompt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
id: demo-bug
title: Demo
track: bug_review
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
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	writeFile(t, filepath.Join(dir, "bug", "x.txt"), "data\n")
	if _, err := pack.Load(dir); err == nil {
		t.Fatal("expected error: empty prompt")
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

func TestReadBugCorpusIncludesTSFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "manifest.yaml"), `
pack_format_version: 1
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
`)
	writeFile(t, filepath.Join(dir, "problem.md"), "demo")
	writeFile(t, filepath.Join(dir, "bug", "notes.md"), "## summary\n")
	writeFile(t, filepath.Join(dir, "bug", "uploader.ts"), "export const ok = true;\n")
	writeFile(t, filepath.Join(dir, "bug", "ignored.png"), "binary")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	corpus, err := p.ReadBugCorpus()
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	if !strings.Contains(corpus, "uploader.ts") || !strings.Contains(corpus, "export const ok") {
		t.Fatalf("expected .ts content in corpus, got:\n%s", corpus)
	}
	if strings.Contains(corpus, "ignored.png") {
		t.Fatalf("did not expect .png in corpus, got:\n%s", corpus)
	}
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
