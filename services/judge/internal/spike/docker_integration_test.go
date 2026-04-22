package spike

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestRunnerDefaultScenarios(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not available")
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker daemon is not available")
	}

	runner := NewRunner()
	for _, scenario := range DefaultScenarios() {
		scenario := scenario
		t.Run(scenario.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			result, err := runner.EvaluateScenario(ctx, scenario)
			if err != nil {
				t.Fatalf("EvaluateScenario() error = %v", err)
			}

			if result.Verdict != scenario.ExpectedVerdict {
				t.Fatalf("scenario %s verdict = %q, want %q", scenario.Name, result.Verdict, scenario.ExpectedVerdict)
			}
		})
	}
}
