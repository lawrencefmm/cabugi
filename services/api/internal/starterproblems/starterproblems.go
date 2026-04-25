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

	countPositivesBundle, countPositivesSHA, err := buildBundleJSON([]testCase{
		{Input: "5\n1 -2 3 0 4\n", ExpectedOutput: "3\n"},
		{Input: "4\n-1 -2 -3 -4\n", ExpectedOutput: "0\n"},
		{Input: "6\n0 0 0 0 0 0\n", ExpectedOutput: "0\n"},
		{Input: "1\n7\n", ExpectedOutput: "1\n"},
		{Input: "7\n-5 1000000000 1 -1 2 -2 3\n", ExpectedOutput: "4\n"},
	})
	if err != nil {
		return nil, err
	}

	palindromeBundle, palindromeSHA, err := buildBundleJSON([]testCase{
		{Input: "level\n", ExpectedOutput: "YES\n"},
		{Input: "cabugi\n", ExpectedOutput: "NO\n"},
		{Input: "a\n", ExpectedOutput: "YES\n"},
		{Input: "abccba\n", ExpectedOutput: "YES\n"},
		{Input: "abca\n", ExpectedOutput: "NO\n"},
	})
	if err != nil {
		return nil, err
	}

	runningSumBundle, runningSumSHA, err := buildBundleJSON([]testCase{
		{Input: "5\n1 2 3 4 5\n", ExpectedOutput: "1 3 6 10 15\n"},
		{Input: "4\n5 -2 7 0\n", ExpectedOutput: "5 3 10 10\n"},
		{Input: "1\n9\n", ExpectedOutput: "9\n"},
		{Input: "3\n0 0 0\n", ExpectedOutput: "0 0 0\n"},
		{Input: "6\n1 -1 1 -1 1 -1\n", ExpectedOutput: "1 0 1 0 1 0\n"},
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
		{
			Slug:                   "count-positives",
			Title:                  "Count Positives",
			StatementMarkdown:      "Given `n` integers, print how many of them are strictly greater than zero.",
			InputMarkdown:          "The first line contains one integer `n`. The second line contains `n` space-separated integers.",
			OutputMarkdown:         "Print one integer: the number of positive values in the list.",
			ConstraintsMarkdown:    "`1 <= n <= 2 * 10^5`, `-10^9 <= a_i <= 10^9`",
			NotesMarkdown:          "This starter problem focuses on loops, counting, and careful treatment of zeros.",
			TimeLimitMs:            1000,
			MemoryLimitMb:          256,
			HiddenTestBundleKey:    "starter/count-positives/v1.json",
			HiddenTestBundleSHA256: countPositivesSHA,
			HiddenTestBundleJSON:   countPositivesBundle,
		},
		{
			Slug:                   "palindrome-check",
			Title:                  "Palindrome Check",
			StatementMarkdown:      "Given a lowercase string `s`, print `YES` if `s` reads the same forward and backward. Otherwise print `NO`.",
			InputMarkdown:          "The input contains one lowercase string `s` without spaces.",
			OutputMarkdown:         "Print `YES` when the string is a palindrome and `NO` otherwise.",
			ConstraintsMarkdown:    "`1 <= |s| <= 10^5`",
			NotesMarkdown:          "This warm-up introduces two-pointer style reasoning and exact output formatting.",
			TimeLimitMs:            1000,
			MemoryLimitMb:          256,
			HiddenTestBundleKey:    "starter/palindrome-check/v1.json",
			HiddenTestBundleSHA256: palindromeSHA,
			HiddenTestBundleJSON:   palindromeBundle,
		},
		{
			Slug:                   "running-sum",
			Title:                  "Running Sum",
			StatementMarkdown:      "Given `n` integers, print the prefix sums of the sequence.",
			InputMarkdown:          "The first line contains one integer `n`. The second line contains `n` space-separated integers.",
			OutputMarkdown:         "Print `n` space-separated integers where the `i`-th value is the sum of the first `i` elements.",
			ConstraintsMarkdown:    "`1 <= n <= 2 * 10^5`, `-10^9 <= a_i <= 10^9`",
			NotesMarkdown:          "This is a slightly longer starter problem that checks accumulation and multi-value output formatting.",
			TimeLimitMs:            1000,
			MemoryLimitMb:          256,
			HiddenTestBundleKey:    "starter/running-sum/v1.json",
			HiddenTestBundleSHA256: runningSumSHA,
			HiddenTestBundleJSON:   runningSumBundle,
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
