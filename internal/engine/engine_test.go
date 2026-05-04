package engine_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icco/bugsim/internal/engine"
	"github.com/icco/bugsim/internal/pack"
	"github.com/icco/bugsim/internal/runner"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available in PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := engine.EnsureDocker(ctx); err != nil {
		t.Skipf("docker not usable: %v", err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newGoPack(t *testing.T, body string) *pack.Pack {
	t.Helper()
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
	writeFile(t, filepath.Join(dir, "skeleton", "go.mod"), "module example.com/demo\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "skeleton", "demo.go"), body)
	writeFile(t, filepath.Join(dir, "hidden_tests", "demo_test.go"), `package demo

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 3); got != 5 {
		t.Fatalf("Add(2,3)=%d", got)
	}
}
`)
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load pack: %v", err)
	}
	return p
}

func TestMaterializeWorkspace(t *testing.T) {
	p := newGoPack(t, "package demo\n\nfunc Add(a, b int) int { return a + b }\n")
	work := filepath.Join(t.TempDir(), "work")
	if err := engine.MaterializeWorkspace(p, work); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	for _, rel := range []string{"go.mod", "demo.go", "demo_test.go"} {
		if _, err := os.Stat(filepath.Join(work, rel)); err != nil {
			t.Fatalf("missing %s in workspace: %v", rel, err)
		}
	}
}

func TestRunTestsPass(t *testing.T) {
	requireDocker(t)
	p := newGoPack(t, "package demo\n\nfunc Add(a, b int) int { return a + b }\n")
	work := filepath.Join(t.TempDir(), "work")
	if err := engine.MaterializeWorkspace(p, work); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	rdef, err := runner.Load(p.Manifest.Runner)
	if err != nil {
		t.Fatalf("runner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := engine.RunTests(ctx, work, rdef, engine.DefaultLimits())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected pass, got exit=%d stderr=%s", res.ExitCode, res.Stderr)
	}
}

func TestMaterializeBugReviewErrors(t *testing.T) {
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
	writeFile(t, filepath.Join(dir, "bug", "x.txt"), "x")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	work := filepath.Join(t.TempDir(), "work")
	if err := engine.MaterializeWorkspace(p, work); err == nil {
		t.Fatal("expected error materialising a bug_review pack")
	}
}

func TestMaterializeMissingSkeleton(t *testing.T) {
	dir := t.TempDir()
	// Bypass pack.Load so we can construct a pack pointing at a directory
	// that lacks skeleton/ on disk and still try to materialise it.
	p := &pack.Pack{
		Dir: dir,
		Manifest: pack.Manifest{
			PackFormatVersion: 1,
			ID:                "demo",
			Title:             "Demo",
			Track:             pack.TrackImplement,
			Runner:            "go",
			Difficulty:        pack.DifficultyEasy,
		},
	}
	work := filepath.Join(t.TempDir(), "work")
	if err := engine.MaterializeWorkspace(p, work); err == nil {
		t.Fatal("expected error: skeleton/ is missing on disk")
	}
}

func TestMaterializeReplacesExistingWorkspace(t *testing.T) {
	p := newGoPack(t, "package demo\n\nfunc Add(a, b int) int { return a + b }\n")
	work := filepath.Join(t.TempDir(), "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(work, "stale.txt")
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := engine.MaterializeWorkspace(p, work); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale file should have been removed, got err=%v", err)
	}
}

func TestMaterializeCollisionDetected(t *testing.T) {
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
	writeFile(t, filepath.Join(dir, "skeleton", "go.mod"), "module example.com/demo\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "skeleton", "demo.go"), "package demo\n")
	// Same path appears in both skeleton/ and hidden_tests/, which the engine
	// must reject so authors notice the overlap.
	writeFile(t, filepath.Join(dir, "hidden_tests", "demo.go"), "package demo\n")
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	work := filepath.Join(t.TempDir(), "work")
	err = engine.MaterializeWorkspace(p, work)
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "hidden_tests collides with skeleton") {
		t.Fatalf("expected collision message, got: %v", err)
	}
}

func TestDefaultLimits(t *testing.T) {
	lim := engine.DefaultLimits()
	if lim.Memory == "" || lim.CPUs == "" || lim.PIDsLimit <= 0 {
		t.Fatalf("default limits should be set, got %+v", lim)
	}
}

func TestRunTestsRejectsEmptyImage(t *testing.T) {
	rdef := &runner.Definition{
		ID:       "broken",
		Network:  "none",
		Commands: map[string][]string{"test": {"true"}},
	}
	work := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := engine.RunTests(ctx, work, rdef, engine.DefaultLimits()); err == nil {
		t.Fatal("expected error: empty image")
	}
}

func TestRunTestsRejectsEmptyCommand(t *testing.T) {
	rdef := &runner.Definition{
		ID:       "broken",
		Image:    "alpine:3",
		Network:  "none",
		Commands: map[string][]string{"test": {}},
	}
	work := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := engine.RunTests(ctx, work, rdef, engine.DefaultLimits()); err == nil {
		t.Fatal("expected error: empty command")
	}
}

func TestRunTestsFail(t *testing.T) {
	requireDocker(t)
	p := newGoPack(t, "package demo\n\nfunc Add(a, b int) int { return a - b }\n")
	work := filepath.Join(t.TempDir(), "work")
	if err := engine.MaterializeWorkspace(p, work); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	rdef, err := runner.Load(p.Manifest.Runner)
	if err != nil {
		t.Fatalf("runner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := engine.RunTests(ctx, work, rdef, engine.DefaultLimits())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ExitCode == 0 {
		t.Fatalf("expected fail, got pass: %s", res.Stdout)
	}
}
