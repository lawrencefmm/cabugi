package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

type stubStore struct {
	job              SubmissionJob
	claimErr         error
	runningID        string
	completedID      string
	completedStatus  string
	completedResults []CaseResult
}

type stubLoader struct {
	cases       []spike.TestCase
	err         error
	receivedKey string
	receivedSHA string
}

type stubRunner struct {
	request spike.Request
	result  spike.Result
	err     error
}

func (store *stubStore) ClaimNextJob(context.Context) (SubmissionJob, error) {
	return store.job, store.claimErr
}

func (store *stubStore) MarkSubmissionRunning(_ context.Context, submissionID string) error {
	store.runningID = submissionID
	return nil
}

func (store *stubStore) CompleteSubmission(_ context.Context, submissionID string, status string, results []CaseResult) error {
	store.completedID = submissionID
	store.completedStatus = status
	store.completedResults = results
	return nil
}

func (loader *stubLoader) LoadCases(_ context.Context, key string, expectedSHA256 string) ([]spike.TestCase, error) {
	loader.receivedKey = key
	loader.receivedSHA = expectedSHA256
	return loader.cases, loader.err
}

func (runner *stubRunner) Evaluate(_ context.Context, request spike.Request) (spike.Result, error) {
	runner.request = request
	return runner.result, runner.err
}

func TestProcessOneReturnsFalseWhenNoJobsExist(t *testing.T) {
	processor := NewProcessor(&stubStore{claimErr: ErrNoJobs}, &stubLoader{}, &stubRunner{})
	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if processed {
		t.Fatal("ProcessOne() should report no work when the queue is empty")
	}
}

func TestProcessOneClaimsRunsAndCompletesSubmission(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", Language: "cpp17", SourceCode: "int main() {}", BundleKey: "bundles/two-sum.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictAccepted, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictAccepted, Duration: 25 * time.Millisecond}}}}
	loader := &stubLoader{cases: []spike.TestCase{{Input: "21\n", ExpectedOutput: "42\n"}}}
	processor := NewProcessor(store, loader, runner)

	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessOne() should report that a job was processed")
	}
	if store.runningID != "submission-id" {
		t.Fatalf("MarkSubmissionRunning() received %q, want %q", store.runningID, "submission-id")
	}
	if store.completedStatus != "accepted" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "accepted")
	}
	if len(store.completedResults) != 1 || store.completedResults[0].Verdict != "accepted" {
		t.Fatalf("CompleteSubmission() results = %#v, want accepted case result", store.completedResults)
	}
	if runner.request.Language != spike.LanguageCPP17 {
		t.Fatalf("runner received language %q, want %q", runner.request.Language, spike.LanguageCPP17)
	}
	if runner.request.TimeLimit != time.Second {
		t.Fatalf("runner received time limit %s, want %s", runner.request.TimeLimit, time.Second)
	}
	if loader.receivedKey != "bundles/two-sum.json" || loader.receivedSHA != "bundle-sha" {
		t.Fatalf("loader received unexpected bundle metadata: key=%q sha=%q", loader.receivedKey, loader.receivedSHA)
	}
}

func TestProcessOneMapsCompileErrorToFinalSubmissionStatus(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", Language: "cpp17", SourceCode: "broken", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictCompileError}}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "1\n"}}}, runner)

	_, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if store.completedStatus != "compile_error" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "compile_error")
	}
}

func TestProcessOneMapsWrongAnswerToFinalSubmissionStatus(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", Language: "cpp17", SourceCode: "wrong", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictWrongAnswer, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictWrongAnswer, Duration: 10 * time.Millisecond}}}}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "2\n"}}}, runner)

	_, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if store.completedStatus != "wrong_answer" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "wrong_answer")
	}
}

func TestProcessOneMapsTimeLimitExceededToFinalSubmissionStatus(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", Language: "python", SourceCode: "while True: pass", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictTimeLimitExceeded, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictTimeLimitExceeded, Duration: 1500 * time.Millisecond}}}}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "2\n"}}}, runner)

	_, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if store.completedStatus != "time_limit_exceeded" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "time_limit_exceeded")
	}
}

func TestProcessOneReturnsLoaderErrors(t *testing.T) {
	processor := NewProcessor(&stubStore{job: SubmissionJob{SubmissionID: "submission-id", Language: "cpp17", SourceCode: "int main() {}", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}, &stubLoader{err: errors.New("missing bundle")}, &stubRunner{})
	_, err := processor.ProcessOne(context.Background())
	if err == nil {
		t.Fatal("ProcessOne() should return bundle loader errors")
	}
}
