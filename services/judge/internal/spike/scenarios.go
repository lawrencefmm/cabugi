package spike

import (
	"path/filepath"
	"runtime"
	"time"
)

func DefaultScenarios() []Scenario {
	return []Scenario{
		{
			Name:            "cpp17-accepted",
			Language:        LanguageCPP17,
			SourcePath:      fixturePath("testdata", "spike", "cpp17", "accepted", "main.cpp"),
			TimeLimit:       2 * time.Second,
			Cases:           []TestCase{{Input: "21\n", ExpectedOutput: "42\n"}},
			ExpectedVerdict: VerdictAccepted,
		},
		{
			Name:            "cpp17-wrong-answer",
			Language:        LanguageCPP17,
			SourcePath:      fixturePath("testdata", "spike", "cpp17", "wrong_answer", "main.cpp"),
			TimeLimit:       2 * time.Second,
			Cases:           []TestCase{{Input: "21\n", ExpectedOutput: "42\n"}},
			ExpectedVerdict: VerdictWrongAnswer,
		},
		{
			Name:            "cpp17-compile-error",
			Language:        LanguageCPP17,
			SourcePath:      fixturePath("testdata", "spike", "cpp17", "compile_error", "main.cpp"),
			TimeLimit:       2 * time.Second,
			Cases:           []TestCase{{Input: "21\n", ExpectedOutput: "42\n"}},
			ExpectedVerdict: VerdictCompileError,
		},
		{
			Name:            "cpp17-time-limit-exceeded",
			Language:        LanguageCPP17,
			SourcePath:      fixturePath("testdata", "spike", "cpp17", "time_limit_exceeded", "main.cpp"),
			TimeLimit:       500 * time.Millisecond,
			Cases:           []TestCase{{Input: "21\n", ExpectedOutput: "42\n"}},
			ExpectedVerdict: VerdictTimeLimitExceeded,
		},
		{
			Name:            "python-accepted",
			Language:        LanguagePython,
			SourcePath:      fixturePath("testdata", "spike", "python", "accepted", "main.py"),
			TimeLimit:       2 * time.Second,
			Cases:           []TestCase{{Input: "21\n", ExpectedOutput: "42\n"}},
			ExpectedVerdict: VerdictAccepted,
		},
	}
}

func ScenarioByName(name string) (Scenario, bool) {
	for _, scenario := range DefaultScenarios() {
		if scenario.Name == name {
			return scenario, true
		}
	}

	return Scenario{}, false
}

func fixturePath(parts ...string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))

	allParts := append([]string{moduleRoot}, parts...)
	return filepath.Join(allParts...)
}
