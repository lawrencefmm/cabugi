package starterproblems

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type ProblemSeed struct {
	Slug                   string
	Title                  string
	StatementMarkdown      string
	InputMarkdown          string
	OutputMarkdown         string
	ConstraintsMarkdown    string
	NotesMarkdown          string
	TimeLimitMs            int
	MemoryLimitMb          int
	HiddenTestBundleKey    string
	HiddenTestBundleSHA256 string
	HiddenTestBundleJSON   []byte
}

type bundle struct {
	Cases []testCase `json:"cases"`
}

type testCase struct {
	Input          string
	ExpectedOutput string
}

func StarterProblems() ([]ProblemSeed, error) {
	plusBundle, plusSHA, err := buildBundleJSON([]testCase{
		{Input: "1 2\n", ExpectedOutput: "3\n"},
		{Input: "10 -4\n", ExpectedOutput: "6\n"},
		{Input: "0 0\n", ExpectedOutput: "0\n"},
		{Input: "999999999 1\n", ExpectedOutput: "1000000000\n"},
		{Input: "-1000000000 7\n", ExpectedOutput: "-999999993\n"},
	})
	if err != nil {
		return nil, err
	}

	reverseBundle, reverseSHA, err := buildBundleJSON([]testCase{
		{Input: "cabugi\n", ExpectedOutput: "igubac\n"},
		{Input: "level\n", ExpectedOutput: "level\n"},
		{Input: "competitive\n", ExpectedOutput: "evititepmoc\n"},
		{Input: strings.Repeat("a", 32) + "\n", ExpectedOutput: strings.Repeat("a", 32) + "\n"},
		{Input: "algorithm\n", ExpectedOutput: "mhtirogla\n"},
	})
	if err != nil {
		return nil, err
	}

	return []ProblemSeed{
		{
			Slug:                   "a-plus-b",
			Title:                  "A + B",
			StatementMarkdown:      "Given two integers `a` and `b`, print their sum.",
			InputMarkdown:          "The input contains two space-separated integers `a` and `b`.",
			OutputMarkdown:         "Print one integer: `a + b`.",
			ConstraintsMarkdown:    "`-10^9 <= a, b <= 10^9`",
			NotesMarkdown:          "Careful handling whitespace is enough for this starter problem.",
			TimeLimitMs:            1000,
			MemoryLimitMb:          256,
			HiddenTestBundleKey:    "starter/a-plus-b/v1.json",
			HiddenTestBundleSHA256: plusSHA,
			HiddenTestBundleJSON:   plusBundle,
		},
		{
			Slug:                   "reverse-string",
			Title:                  "Reverse String",
			StatementMarkdown:      "Given a lowercase string `s`, print the characters of `s` in reverse order.",
			InputMarkdown:          "The input contains one lowercase string `s` without spaces.",
			OutputMarkdown:         "Print the reversed string.",
			ConstraintsMarkdown:    "`1 <= |s| <= 1000`",
			NotesMarkdown:          "This is a starter warm-up for string handling and indexing.",
			TimeLimitMs:            1000,
			MemoryLimitMb:          256,
			HiddenTestBundleKey:    "starter/reverse-string/v1.json",
			HiddenTestBundleSHA256: reverseSHA,
			HiddenTestBundleJSON:   reverseBundle,
		},
	}, nil
}

func buildBundleJSON(cases []testCase) ([]byte, string, error) {
	contents, err := json.Marshal(bundle{Cases: cases})
	if err != nil {
		return nil, "", fmt.Errorf("marshal hidden test bundle: %w", err)
	}

	sum := sha256.Sum256(contents)
	return contents, hex.EncodeToString(sum[:]), nil
}
