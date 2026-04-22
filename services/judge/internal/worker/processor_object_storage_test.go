package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	judgeconfig "github.com/lawrencefmm/cabugi/services/judge/internal/config"
	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

func TestProcessOneCompletesSubmissionUsingObjectStoredBundle(t *testing.T) {
	contents := []byte(`{"cases":[{"Input":"21\n","ExpectedOutput":"42\n"}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(contents)
	}))
	defer server.Close()

	loader, err := NewS3BundleLoader(judgeconfig.ObjectStorageConfig{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "hidden-tests",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("NewS3BundleLoader() error = %v", err)
	}

	store := &stubStore{job: SubmissionJob{
		SubmissionID: "submission-id",
		Language:     "cpp17",
		SourceCode:   "int main() {}",
		BundleKey:    "bundles/two-sum.json",
		BundleSHA256: sha256Hex(contents),
		TimeLimit:    time.Second,
	}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictAccepted, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictAccepted, Duration: 25 * time.Millisecond}}}}
	processor := NewProcessor(store, loader, runner)

	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessOne() should report that a job was processed")
	}
	if store.completedStatus != "accepted" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "accepted")
	}
	if len(runner.request.Cases) != 1 || runner.request.Cases[0].ExpectedOutput != "42\n" {
		t.Fatalf("runner received unexpected test cases: %#v", runner.request.Cases)
	}
}
