package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

type Runner struct {
	cppImage       string
	pythonImage    string
	compileTimeout time.Duration
	memoryLimit    string
	cpuLimit       string
	nonRootUser    string
}

func NewRunner() Runner {
	return Runner{
		cppImage:       "gcc:14.2.0",
		pythonImage:    "python:3.13.0-alpine3.20",
		compileTimeout: 30 * time.Second,
		memoryLimit:    "512m",
		cpuLimit:       "1",
		nonRootUser:    "65534:65534",
	}
}

func (r Runner) Evaluate(ctx context.Context, request spike.Request) (spike.Result, error) {
	workspace, err := os.MkdirTemp("", "judge-sandbox-*")
	if err != nil {
		return spike.Result{}, fmt.Errorf("create workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	if err := os.Chmod(workspace, 0o777); err != nil {
		return spike.Result{}, fmt.Errorf("set workspace permissions: %w", err)
	}

	entrypoint, err := writeSource(workspace, request.Language, request.Source)
	if err != nil {
		return spike.Result{}, err
	}

	executablePath := entrypoint
	if request.Language == spike.LanguageCPP17 {
		if err := r.ensureImagePrepared(ctx, r.cppImage); err != nil {
			return spike.Result{}, err
		}

		compileOutput, compileErr := r.compileCPP(ctx, workspace, entrypoint)
		if compileErr != nil {
			return spike.Result{Verdict: spike.VerdictCompileError, CompileOutput: compileOutput}, nil
		}
		executablePath = filepath.Join(workspace, "program")
	} else {
		if err := r.ensureImagePrepared(ctx, r.pythonImage); err != nil {
			return spike.Result{}, err
		}
	}

	result := spike.Result{Verdict: spike.VerdictAccepted}
	for _, testCase := range request.Cases {
		caseResult, err := r.runCase(ctx, request, workspace, executablePath, testCase)
		if err != nil {
			return spike.Result{}, err
		}

		result.CaseResults = append(result.CaseResults, caseResult)
		if caseResult.Verdict != spike.VerdictAccepted {
			result.Verdict = caseResult.Verdict
			return result, nil
		}
	}

	return result, nil
}

func writeSource(workspace string, language spike.Language, source string) (string, error) {
	filename := "main.py"
	if language == spike.LanguageCPP17 {
		filename = "main.cpp"
	}

	path := filepath.Join(workspace, filename)
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		return "", fmt.Errorf("write source file: %w", err)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		return "", fmt.Errorf("set source permissions: %w", err)
	}

	return path, nil
}

func (r Runner) ensureImagePrepared(ctx context.Context, image string) error {
	inspect := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if err := inspect.Run(); err != nil {
		return fmt.Errorf("required judge image %s is not available; prepare it before running the worker: %w", image, err)
	}

	return nil
}

func (r Runner) compileCPP(ctx context.Context, workspace string, sourcePath string) (string, error) {
	compileCtx, cancel := context.WithTimeout(ctx, r.compileTimeout)
	defer cancel()

	stdout, stderr, err := r.runContainer(compileCtx, workspace, r.cppImage, []string{"sh", "-lc", fmt.Sprintf("g++ -std=c++17 -O2 -pipe -o program %s", filepath.Base(sourcePath))}, "")
	output := strings.TrimSpace(strings.Join([]string{stdout, stderr}, "\n"))
	if err != nil {
		return output, err
	}

	return output, nil
}

func (r Runner) runCase(ctx context.Context, request spike.Request, workspace string, executablePath string, testCase spike.TestCase) (spike.CaseResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, request.TimeLimit)
	defer cancel()

	command := []string{"python", filepath.Base(executablePath)}
	image := r.pythonImage
	if request.Language == spike.LanguageCPP17 {
		command = []string{"sh", "-lc", fmt.Sprintf("./%s", filepath.Base(executablePath))}
		image = r.cppImage
	}

	start := time.Now()
	stdout, stderr, err := r.runContainer(runCtx, workspace, image, command, testCase.Input)
	duration := time.Since(start)

	if runCtx.Err() == context.DeadlineExceeded {
		return spike.CaseResult{Verdict: spike.VerdictTimeLimitExceeded, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
	}

	if err != nil {
		return spike.CaseResult{Verdict: spike.VerdictRuntimeError, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
	}

	if !outputsMatch(stdout, testCase.ExpectedOutput) {
		return spike.CaseResult{Verdict: spike.VerdictWrongAnswer, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
	}

	return spike.CaseResult{Verdict: spike.VerdictAccepted, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
}

func (r Runner) runContainer(ctx context.Context, workspace string, image string, command []string, stdin string) (string, string, error) {
	containerName := fmt.Sprintf("judge-sandbox-%d", time.Now().UnixNano())
	args := r.containerArgs(workspace, containerName, image, command)

	cmd := exec.CommandContext(ctx, "docker", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		_ = exec.Command("docker", "rm", "-f", containerName).Run()
	}

	return stdout.String(), stderr.String(), err
}

func (r Runner) containerArgs(workspace string, containerName string, image string, command []string) []string {
	args := []string{
		"run",
		"--rm",
		"--pull", "never",
		"-i",
		"--name", containerName,
		"--network", "none",
		"--cpus", r.cpuLimit,
		"--memory", r.memoryLimit,
		"--pids-limit", "64",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", r.nonRootUser,
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"--tmpfs", "/run:rw,noexec,nosuid,size=16m",
		"--tmpfs", "/var/tmp:rw,noexec,nosuid,size=64m",
		"--env", "HOME=/tmp",
		"--env", "PYTHONDONTWRITEBYTECODE=1",
		"--mount", fmt.Sprintf("type=bind,src=%s,dst=/workspace,rw", workspace),
		"-w", "/workspace",
		image,
	}
	args = append(args, command...)
	return args
}

func outputsMatch(actual string, expected string) bool {
	normalize := func(value string) string {
		value = strings.ReplaceAll(value, "\r\n", "\n")
		return strings.TrimSpace(value)
	}

	return normalize(actual) == normalize(expected)
}
