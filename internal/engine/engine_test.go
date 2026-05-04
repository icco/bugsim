package engine_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
