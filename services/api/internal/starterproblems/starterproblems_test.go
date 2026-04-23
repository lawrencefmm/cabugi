package starterproblems

import "testing"

func TestStarterProblemsReturnsUniquePublishedSeeds(t *testing.T) {
	seeds, err := StarterProblems()
	if err != nil {
		t.Fatalf("StarterProblems() error = %v", err)
	}
	if len(seeds) < 2 {
		t.Fatalf("StarterProblems() returned %d seeds, want at least 2", len(seeds))
	}

	seen := make(map[string]struct{}, len(seeds))
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
		seen[seed.Slug] = struct{}{}
	}
}
