package runner_test

import (
	"testing"

	"github.com/icco/bugsim/internal/runner"
)

func TestLoadGoRunner(t *testing.T) {
	d, err := runner.Load("go")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if d.ID != "go" {
		t.Fatalf("id = %q", d.ID)
	}
	if d.Image == "" {
		t.Fatal("image is empty")
	}
	if d.Network != "none" {
		t.Fatalf("network = %q, want none default", d.Network)
	}
	if got := d.Commands["test"]; len(got) == 0 || got[0] != "go" {
		t.Fatalf("test cmd = %v", got)
	}
}

func TestLoadUnknownRunner(t *testing.T) {
	if _, err := runner.Load("does-not-exist"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadEmptyID(t *testing.T) {
	if _, err := runner.Load(""); err == nil {
		t.Fatal("expected error: empty id")
	}
	if _, err := runner.Load("  "); err == nil {
		t.Fatal("expected error: whitespace id")
	}
}

func TestLoadGoRunnerEnvKeys(t *testing.T) {
	d, err := runner.Load("go")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, key := range []string{"GOCACHE", "GOMODCACHE"} {
		if _, ok := d.Env[key]; !ok {
			t.Fatalf("expected env[%q] to be set, got %#v", key, d.Env)
		}
	}
}

func TestLoadTypescriptRunner(t *testing.T) {
	d, err := runner.Load("typescript")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if d.ID != "typescript" {
		t.Fatalf("id = %q, want typescript", d.ID)
	}
	if d.Image == "" {
		t.Fatal("typescript runner must declare an image")
	}
	if d.Network != "none" {
		t.Fatalf("typescript runner network = %q, want none default", d.Network)
	}
}

func TestLoadTypescriptRunnerCommand(t *testing.T) {
	d, err := runner.Load("typescript")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cmd := d.Commands["test"]
	if len(cmd) == 0 || cmd[0] != "node" {
		t.Fatalf("typescript test cmd should start with node, got %v", cmd)
	}
	hasTestFlag := false
	for _, arg := range cmd {
		if arg == "--test" {
			hasTestFlag = true
		}
	}
	if !hasTestFlag {
		t.Fatalf("typescript test cmd should include --test, got %v", cmd)
	}
}

func TestLoadTypescriptRunnerEnv(t *testing.T) {
	d, err := runner.Load("typescript")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := d.Env["NODE_NO_WARNINGS"]; got != "1" {
		t.Fatalf("NODE_NO_WARNINGS = %q, want \"1\"", got)
	}
}
