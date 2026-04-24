package sandbox

import (
	"slices"
	"testing"
)

func TestNewRunnerUsesPinnedImages(t *testing.T) {
	runner := NewRunner()

	if runner.cppImage != "gcc:14.2.0" {
		t.Fatalf("cppImage = %q, want %q", runner.cppImage, "gcc:14.2.0")
	}
	if runner.pythonImage != "python:3.13.0-alpine3.20" {
		t.Fatalf("pythonImage = %q, want %q", runner.pythonImage, "python:3.13.0-alpine3.20")
	}
	if runner.nonRootUser != "65534:65534" {
		t.Fatalf("nonRootUser = %q, want %q", runner.nonRootUser, "65534:65534")
	}
}

func TestContainerArgsIncludeHardeningFlags(t *testing.T) {
	runner := NewRunner()
	args := runner.containerArgs("/tmp/workspace", "judge-sandbox-1", runner.cppImage, []string{"sh", "-lc", "echo ok"})

	for _, expected := range []string{
		"--pull", "never",
		"--network", "none",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", "65534:65534",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"--tmpfs", "/run:rw,noexec,nosuid,size=16m",
		"--tmpfs", "/var/tmp:rw,noexec,nosuid,size=64m",
		"--env", "HOME=/tmp",
		"--env", "PYTHONDONTWRITEBYTECODE=1",
		"--mount", "type=bind,src=/tmp/workspace,dst=/workspace",
	} {
		if !slices.Contains(args, expected) {
			t.Fatalf("container args = %#v, want %q to be present", args, expected)
		}
	}
}
