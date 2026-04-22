package spike

import "time"

type Language string

const (
	LanguageCPP17  Language = "cpp17"
	LanguagePython Language = "python"
)

type Verdict string

const (
	VerdictAccepted          Verdict = "Accepted"
	VerdictWrongAnswer       Verdict = "Wrong Answer"
	VerdictCompileError      Verdict = "Compile Error"
	VerdictRuntimeError      Verdict = "Runtime Error"
	VerdictTimeLimitExceeded Verdict = "Time Limit Exceeded"
)

type TestCase struct {
	Input          string
	ExpectedOutput string
}

type Scenario struct {
	Name            string
	Language        Language
	SourcePath      string
	TimeLimit       time.Duration
	Cases           []TestCase
	ExpectedVerdict Verdict
}

type Request struct {
	Language  Language
	Source    string
	TimeLimit time.Duration
	Cases     []TestCase
}

type CaseResult struct {
	Verdict  Verdict
	Stdout   string
	Stderr   string
	Duration time.Duration
}

type Result struct {
	Verdict       Verdict
	CompileOutput string
	CaseResults   []CaseResult
}
