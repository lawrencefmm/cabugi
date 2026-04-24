package spike

import "testing"

func TestOutputsMatch(t *testing.T) {
	cases := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{name: "exact match", actual: "42\n", expected: "42\n", want: true},
		{name: "normalizes line endings and trailing spaces", actual: "42\r\n", expected: "42\n", want: true},
		{name: "detects different answers", actual: "41\n", expected: "42\n", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := outputsMatch(tc.actual, tc.expected); got != tc.want {
				t.Fatalf("outputsMatch(%q, %q) = %t, want %t", tc.actual, tc.expected, got, tc.want)
			}
		})
	}
}

func TestScenarioByName(t *testing.T) {
	scenario, ok := ScenarioByName("python-accepted")
	if !ok {
		t.Fatal("expected python-accepted scenario to exist")
	}

	if scenario.ExpectedVerdict != VerdictAccepted {
		t.Fatalf("python-accepted expected verdict = %q, want %q", scenario.ExpectedVerdict, VerdictAccepted)
	}
}
