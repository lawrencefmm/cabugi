package starterproblems

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestStarterProblemsReturnsUniquePublishedSeeds(t *testing.T) {
	seeds, err := StarterProblems()
	if err != nil {
		t.Fatalf("StarterProblems() error = %v", err)
	}
	if len(seeds) < 5 {
		t.Fatalf("StarterProblems() returned %d seeds, want at least 5", len(seeds))
	}

	seen := make(map[string]struct{}, len(seeds))
	expectedSlugs := map[string]struct{}{
		"a-plus-b":          {},
		"reverse-string":    {},
		"count-positives":   {},
		"palindrome-check":  {},
		"running-sum":       {},
	}
	for _, seed := range seeds {
		if seed.Slug == "" || seed.Title == "" {
			t.Fatalf("StarterProblems() returned incomplete seed: %#v", seed)
		}
		if seed.HiddenTestBundleKey == "" || seed.HiddenTestBundleSHA256 == "" || len(seed.HiddenTestBundleJSON) == 0 {
			t.Fatalf("StarterProblems() returned incomplete hidden bundle metadata: %#v", seed)
		}
		if _, ok := seen[seed.Slug]; ok {
			t.Fatalf("StarterProblems() returned duplicate slug %q", seed.Slug)
		}
		if _, ok := expectedSlugs[seed.Slug]; !ok {
			t.Fatalf("StarterProblems() returned unexpected slug %q", seed.Slug)
		}

		var hiddenBundle bundle
		if err := json.Unmarshal(seed.HiddenTestBundleJSON, &hiddenBundle); err != nil {
			t.Fatalf("StarterProblems() returned invalid hidden bundle JSON for %q: %v", seed.Slug, err)
		}
		if len(hiddenBundle.Cases) == 0 {
			t.Fatalf("StarterProblems() returned empty hidden bundle for %q", seed.Slug)
		}

		sum := sha256.Sum256(seed.HiddenTestBundleJSON)
		if got := hex.EncodeToString(sum[:]); got != seed.HiddenTestBundleSHA256 {
			t.Fatalf("StarterProblems() returned checksum %q for %q, want %q", seed.HiddenTestBundleSHA256, seed.Slug, got)
		}

		seen[seed.Slug] = struct{}{}
	}
}
