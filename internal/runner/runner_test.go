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
