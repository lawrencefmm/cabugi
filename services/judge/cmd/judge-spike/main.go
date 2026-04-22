package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

func main() {
	scenarioName := flag.String("scenario", "", "run a single named scenario")
	runAll := flag.Bool("all", false, "run all built-in scenarios")
	flag.Parse()

	runner := spike.NewRunner()

	if *runAll || *scenarioName == "" {
		if err := runScenarios(runner, spike.DefaultScenarios()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	scenario, ok := spike.ScenarioByName(*scenarioName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown scenario %q\n", *scenarioName)
		os.Exit(1)
	}

	if err := runScenarios(runner, []spike.Scenario{scenario}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runScenarios(runner spike.Runner, scenarios []spike.Scenario) error {
	for _, scenario := range scenarios {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		result, err := runner.EvaluateScenario(ctx, scenario)
		cancel()
		if err != nil {
			return fmt.Errorf("%s: %w", scenario.Name, err)
		}

		fmt.Printf("%s: %s\n", scenario.Name, result.Verdict)
		if result.Verdict != scenario.ExpectedVerdict {
			return fmt.Errorf("scenario %s returned %s, want %s", scenario.Name, result.Verdict, scenario.ExpectedVerdict)
		}
	}

	fmt.Println("all scenarios passed")
	return nil
}
