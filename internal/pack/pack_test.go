package pack_test

import (
	"os"
	"path/filepath"
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
