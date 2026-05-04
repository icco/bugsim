// Package engine materializes pack workspaces and orchestrates Docker-isolated
// test runs against them.
package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/icco/bugsim/internal/pack"
	"github.com/icco/bugsim/internal/runner"
)

// MaterializeWorkspace copies skeleton/ then hidden_tests/ into dstDir.
// File-name collisions between the two are reported as an error so pack
// authors notice unintended overlap.
func MaterializeWorkspace(p *pack.Pack, dstDir string) error {
	if p.Manifest.Track != pack.TrackImplement {
		return errors.New("only implement packs have a workspace")
	}
	if err := os.RemoveAll(dstDir); err != nil {
		return fmt.Errorf("reset workspace: %w", err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	if err := os.CopyFS(dstDir, os.DirFS(filepath.Join(p.Dir, "skeleton"))); err != nil {
		return fmt.Errorf("copy skeleton: %w", err)
	}
	if err := os.CopyFS(dstDir, os.DirFS(filepath.Join(p.Dir, "hidden_tests"))); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("hidden_tests collides with skeleton (rename to avoid overlap): %w", err)
		}
		return fmt.Errorf("copy hidden_tests: %w", err)
	}
	return nil
}

// TestResult captures subprocess output from the runner's test command.
type TestResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Limits configure the resource caps applied to each container.
type Limits struct {
	Memory    string
	CPUs      string
	PIDsLimit int
}

// DefaultLimits returns the default per-run container resource caps.
func DefaultLimits() Limits {
	return Limits{Memory: "512m", CPUs: "2", PIDsLimit: 256}
}

// dockerBin is overridable for tests.
var dockerBin = "docker"

// EnsureDocker verifies the docker CLI is callable.
func EnsureDocker(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, dockerBin, "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is required for bugsim runs (`docker version` failed): %w", err)
	}
	return nil
}

// RunTests executes the runner's test command inside a Docker container with
// workDir bind-mounted at /workspace.
func RunTests(ctx context.Context, workDir string, rdef *runner.Definition, lim Limits) (*TestResult, error) {
	if len(rdef.Commands["test"]) == 0 {
		return nil, errors.New("runner has empty test command")
	}
	if rdef.Image == "" {
		return nil, errors.New("runner has no image")
	}

	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("workspace abs: %w", err)
	}

	dockerArgs := slices.Concat(
		[]string{
			"run", "--rm",
			"--network", rdef.Network,
			"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
			"-v", absWork + ":/workspace",
			"-w", "/workspace",
			"-e", "HOME=/tmp",
		},
		limitArgs(lim),
		envArgs(rdef.Env),
		[]string{rdef.Image},
		rdef.Commands["test"],
	)

	cmd := exec.CommandContext(ctx, dockerBin, dockerArgs...)
	cmd.WaitDelay = 5 * time.Second

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	res := &TestResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
	return res, nil
}

func limitArgs(lim Limits) []string {
	var args []string
	if lim.Memory != "" {
		args = append(args, "--memory", lim.Memory)
	}
	if lim.CPUs != "" {
		args = append(args, "--cpus", lim.CPUs)
	}
	if lim.PIDsLimit > 0 {
		args = append(args, "--pids-limit", strconv.Itoa(lim.PIDsLimit))
	}
	return args
}

func envArgs(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	args := make([]string, 0, 2*len(env))
	for _, k := range keys {
		args = append(args, "-e", k+"="+env[k])
	}
	return args
}
