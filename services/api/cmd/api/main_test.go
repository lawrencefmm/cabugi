package main

import "testing"

func TestFailStartupIfRequired(t *testing.T) {
	if err := failStartupIfRequired(false, false, "database"); err != nil {
		t.Fatalf("failStartupIfRequired() optional dependency error = %v, want nil", err)
	}
	if err := failStartupIfRequired(true, true, "database"); err != nil {
		t.Fatalf("failStartupIfRequired() ready dependency error = %v, want nil", err)
	}
	if err := failStartupIfRequired(true, false, "database"); err == nil {
		t.Fatal("failStartupIfRequired() should fail when a required dependency is not ready")
	}
}
