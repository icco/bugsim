package runner_test

import (
	"strings"
	"testing"

	"github.com/icco/bugsim/internal/runner"
	"github.com/icco/bugsim/runners"
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
	if _, err := runner.Load("   "); err == nil {
		t.Fatal("expected error: whitespace id")
	}
}

func TestLoadGoRunnerEnvKeys(t *testing.T) {
	// GOCACHE/GOMODCACHE must point inside the writable workspace tmpdir
	// or the runner can't cache builds across runs of the same session.
	d, err := runner.Load("go")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, key := range []string{"GOCACHE", "GOMODCACHE"} {
		v, ok := d.Env[key]
		if !ok {
			t.Fatalf("expected env[%q] to be set, got %#v", key, d.Env)
		}
		if !strings.HasPrefix(v, "/tmp/") {
			t.Fatalf("env[%q] = %q, want /tmp/* path", key, v)
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
	if !strings.HasPrefix(d.Image, "node:") {
		t.Fatalf("image = %q, want a node image", d.Image)
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

// TestEmbeddedRunnersAllPassSchema walks the embedded runners/*.json
// files and checks each one against the runner schema. This catches
// future contributors who add a runner with a missing image, mismatched
// id, or unsupported network value — without needing a per-runner test.
func TestEmbeddedRunnersAllPassSchema(t *testing.T) {
	entries, err := runners.Files.ReadDir(".")
	if err != nil {
		t.Fatalf("read embed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no embedded runners found — embed.go is broken")
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		t.Run(id, func(t *testing.T) {
			d, err := runner.Load(id)
			if err != nil {
				t.Fatalf("load %s: %v", id, err)
			}
			if d.ID != id {
				t.Fatalf("id %q != filename stem %q", d.ID, id)
			}
			if d.Image == "" {
				t.Fatalf("%s: image is empty", id)
			}
			if len(d.Commands["test"]) == 0 {
				t.Fatalf("%s: commands.test is empty", id)
			}
			switch d.Network {
			case "none", "bridge":
			default:
				t.Fatalf("%s: network = %q, want none or bridge", id, d.Network)
			}
		})
	}
}
