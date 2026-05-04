package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/icco/bugsim/internal/fsutil"
	"github.com/icco/bugsim/internal/pack"
	"github.com/icco/bugsim/internal/runner"
)

// MaterializeWorkspace copies skeleton/ and hidden_tests/ into dstDir.
// hidden_tests/ is applied after skeleton/ (overwrites on name collisions).
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
	sk := filepath.Join(p.Dir, "skeleton")
	ht := filepath.Join(p.Dir, "hidden_tests")
	if err := fsutil.CopyTree(dstDir, sk); err != nil {
		return fmt.Errorf("copy skeleton: %w", err)
	}
	if err := fsutil.CopyTree(dstDir, ht); err != nil {
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
	Memory    string // e.g. "512m"
	CPUs      string // e.g. "2"
	PIDsLimit int    // e.g. 256
}

// DefaultLimits returns the default per-run container resource caps.
func DefaultLimits() Limits {
	return Limits{Memory: "512m", CPUs: "2", PIDsLimit: 256}
}

// dockerBin is overridable for tests.
var dockerBin = "docker"

// EnsureDocker verifies the docker CLI is callable. Returns a friendly error if not.
func EnsureDocker(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, dockerBin, "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is required for bugsim runs (`docker version` failed): %w", err)
	}
	return nil
}

// RunTests executes the runner's test command inside a Docker container with workDir
// bind-mounted at /workspace. The container is removed on exit.
func RunTests(ctx context.Context, workDir string, rdef *runner.Definition, lim Limits) (*TestResult, error) {
	argv := append([]string(nil), rdef.Commands["test"]...)
	if len(argv) == 0 {
		return nil, errors.New("runner has empty test command")
	}
	if rdef.Image == "" {
		return nil, errors.New("runner has no image")
	}

	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("workspace abs: %w", err)
	}

	dockerArgs := []string{
		"run", "--rm",
		"--network", rdef.Network,
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"-v", absWork + ":/workspace",
		"-w", "/workspace",
		"-e", "HOME=/tmp",
	}
	if lim.Memory != "" {
		dockerArgs = append(dockerArgs, "--memory", lim.Memory)
	}
	if lim.CPUs != "" {
		dockerArgs = append(dockerArgs, "--cpus", lim.CPUs)
	}
	if lim.PIDsLimit > 0 {
		dockerArgs = append(dockerArgs, "--pids-limit", strconv.Itoa(lim.PIDsLimit))
	}
	for k, v := range rdef.Env {
		dockerArgs = append(dockerArgs, "-e", k+"="+v)
	}
	dockerArgs = append(dockerArgs, rdef.Image)
	dockerArgs = append(dockerArgs, argv...)

	cmd := exec.CommandContext(ctx, dockerBin, dockerArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	res := &TestResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
	res.ExitCode = 0
	return res, nil
}
