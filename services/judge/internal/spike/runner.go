package spike

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	cppImage       string
	pythonImage    string
	compileTimeout time.Duration
	memoryLimit    string
	cpuLimit       string
}

func NewRunner() Runner {
	return Runner{
		cppImage:       "gcc:14",
		pythonImage:    "python:3.13-alpine",
		compileTimeout: 30 * time.Second,
		memoryLimit:    "512m",
		cpuLimit:       "1",
	}
}

func (r Runner) EvaluateScenario(ctx context.Context, scenario Scenario) (Result, error) {
	source, err := os.ReadFile(scenario.SourcePath)
	if err != nil {
		return Result{}, fmt.Errorf("read scenario source: %w", err)
	}

	return r.Evaluate(ctx, Request{
		Language:  scenario.Language,
		Source:    string(source),
		TimeLimit: scenario.TimeLimit,
		Cases:     scenario.Cases,
	})
}

func (r Runner) Evaluate(ctx context.Context, request Request) (Result, error) {
	workspace, err := os.MkdirTemp("", "judge-spike-*")
	if err != nil {
		return Result{}, fmt.Errorf("create workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	if err := os.Chmod(workspace, 0o777); err != nil {
		return Result{}, fmt.Errorf("set workspace permissions: %w", err)
	}

	entrypoint, err := writeSource(workspace, request.Language, request.Source)
	if err != nil {
		return Result{}, err
	}

	executablePath := entrypoint
	if request.Language == LanguageCPP17 {
		if err := r.ensureImageAvailable(ctx, r.cppImage); err != nil {
			return Result{}, err
		}

		compileOutput, compileErr := r.compileCPP(ctx, workspace, entrypoint)
		if compileErr != nil {
			return Result{Verdict: VerdictCompileError, CompileOutput: compileOutput}, nil
		}
		executablePath = filepath.Join(workspace, "program")
	} else {
		if err := r.ensureImageAvailable(ctx, r.pythonImage); err != nil {
			return Result{}, err
		}
	}

	result := Result{Verdict: VerdictAccepted}
	for _, testCase := range request.Cases {
		caseResult, err := r.runCase(ctx, request, workspace, executablePath, testCase)
		if err != nil {
			return Result{}, err
		}

		result.CaseResults = append(result.CaseResults, caseResult)
		if caseResult.Verdict != VerdictAccepted {
			result.Verdict = caseResult.Verdict
			return result, nil
		}
	}

	return result, nil
}

func writeSource(workspace string, language Language, source string) (string, error) {
	filename := "main.py"
	if language == LanguageCPP17 {
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

func (r Runner) ensureImageAvailable(ctx context.Context, image string) error {
	inspect := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if err := inspect.Run(); err == nil {
		return nil
	}

	pull := exec.CommandContext(ctx, "docker", "pull", image)
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("pull image %s: %w", image, err)
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

func (r Runner) runCase(ctx context.Context, request Request, workspace string, executablePath string, testCase TestCase) (CaseResult, error) {
	runCtx, cancel := context.WithTimeout(ctx, request.TimeLimit)
	defer cancel()

	command := []string{"python", filepath.Base(executablePath)}
	image := r.pythonImage
	if request.Language == LanguageCPP17 {
		command = []string{"sh", "-lc", fmt.Sprintf("./%s", filepath.Base(executablePath))}
		image = r.cppImage
	}

	start := time.Now()
	stdout, stderr, err := r.runContainer(runCtx, workspace, image, command, testCase.Input)
	duration := time.Since(start)

	if runCtx.Err() == context.DeadlineExceeded {
		return CaseResult{Verdict: VerdictTimeLimitExceeded, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
	}

	if err != nil {
		return CaseResult{Verdict: VerdictRuntimeError, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
	}

	if !outputsMatch(stdout, testCase.ExpectedOutput) {
		return CaseResult{Verdict: VerdictWrongAnswer, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
	}

	return CaseResult{Verdict: VerdictAccepted, Stdout: stdout, Stderr: stderr, Duration: duration}, nil
}

func (r Runner) runContainer(ctx context.Context, workspace string, image string, command []string, stdin string) (string, string, error) {
	containerName := fmt.Sprintf("judge-spike-%d", time.Now().UnixNano())
	args := []string{
		"run",
		"--rm",
		"-i",
		"--name", containerName,
		"--network", "none",
		"--cpus", r.cpuLimit,
		"--memory", r.memoryLimit,
		"--pids-limit", "64",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"-v", fmt.Sprintf("%s:/workspace", workspace),
		"-w", "/workspace",
		image,
	}
	args = append(args, command...)

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

func outputsMatch(actual string, expected string) bool {
	normalize := func(value string) string {
		value = strings.ReplaceAll(value, "\r\n", "\n")
		return strings.TrimSpace(value)
	}

	return normalize(actual) == normalize(expected)
}
