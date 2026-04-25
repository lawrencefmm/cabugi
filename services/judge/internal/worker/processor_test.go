package worker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

type stubStore struct {
	job              SubmissionJob
	claimErr         error
	leaseRenewCalls  int
	leaseRenewErr    error
	leaseRenewedID   string
	leaseRenewedWith string
	runningID        string
	failedID         string
	failedError      string
	failureAction    FailureAction
	handleFailureErr error
	completedID      string
	completedStatus  string
	completedCompile string
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

type blockingRunner struct {
	started chan struct{}
	release chan struct{}
	result  spike.Result
	err     error
	once    sync.Once
}

type stubObserver struct {
	claimedID       string
	claimedLanguage string
	completedID     string
	completedStatus string
	retriedID       string
	retryError      string
	terminalID      string
	terminalError   string
	noJobs          int
	heartbeatCount  int
}

func (store *stubStore) ClaimNextJob(context.Context) (SubmissionJob, error) {
	return store.job, store.claimErr
}

func (store *stubStore) RenewJobLease(_ context.Context, submissionID string, leaseToken string) error {
	store.leaseRenewCalls++
	store.leaseRenewedID = submissionID
	store.leaseRenewedWith = leaseToken
	return store.leaseRenewErr
}

func (store *stubStore) MarkSubmissionRunning(_ context.Context, submissionID string) error {
	store.runningID = submissionID
	return nil
}

func (store *stubStore) HandleJobFailure(_ context.Context, submissionID string, _ string, lastError string) (FailureAction, error) {
	store.failedID = submissionID
	store.failedError = lastError
	if store.failureAction == "" {
		store.failureAction = FailureActionRetried
	}

	return store.failureAction, store.handleFailureErr
}

func (store *stubStore) CompleteSubmission(_ context.Context, submissionID string, _ string, status string, compileOutputExcerpt string, results []CaseResult) error {
	store.completedID = submissionID
	store.completedStatus = status
	store.completedCompile = compileOutputExcerpt
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

func (runner *blockingRunner) Evaluate(ctx context.Context, _ spike.Request) (spike.Result, error) {
	runner.once.Do(func() {
		close(runner.started)
	})

	select {
	case <-runner.release:
		return runner.result, runner.err
	case <-ctx.Done():
		return spike.Result{}, ctx.Err()
	}
}

func (observer *stubObserver) Heartbeat() {
	observer.heartbeatCount++
}

func (observer *stubObserver) JobClaimed(submissionID string, language string) {
	observer.claimedID = submissionID
	observer.claimedLanguage = language
}

func (observer *stubObserver) JobCompleted(submissionID string, status string) {
	observer.completedID = submissionID
	observer.completedStatus = status
}

func (observer *stubObserver) JobRetried(submissionID string, lastError string) {
	observer.retriedID = submissionID
	observer.retryError = lastError
}

func (observer *stubObserver) JobTerminalFailure(submissionID string, lastError string) {
	observer.terminalID = submissionID
	observer.terminalError = lastError
}

func (observer *stubObserver) NoJobs() {
	observer.noJobs++
}

func TestProcessOneReturnsFalseWhenNoJobsExist(t *testing.T) {
	observer := &stubObserver{}
	processor := NewProcessor(&stubStore{claimErr: ErrNoJobs}, &stubLoader{}, &stubRunner{}, observer, time.Second)
	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if processed {
		t.Fatal("ProcessOne() should report no work when the queue is empty")
	}
	if observer.noJobs != 1 {
		t.Fatalf("observer.NoJobs() calls = %d, want 1", observer.noJobs)
	}
}

func TestProcessOneClaimsRunsAndCompletesSubmission(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "cpp17", SourceCode: "int main() {}", BundleKey: "bundles/two-sum.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictAccepted, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictAccepted, Duration: 25 * time.Millisecond}}}}
	loader := &stubLoader{cases: []spike.TestCase{{Input: "21\n", ExpectedOutput: "42\n"}}}
	observer := &stubObserver{}
	processor := NewProcessor(store, loader, runner, observer, time.Second)

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
	if observer.claimedID != "submission-id" || observer.claimedLanguage != "cpp17" || observer.completedID != "submission-id" || observer.completedStatus != "accepted" {
		t.Fatalf("observer recorded unexpected claim/completion: %#v", observer)
	}
}

func TestProcessOneMapsCompileErrorToFinalSubmissionStatus(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "cpp17", SourceCode: "broken", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictCompileError, CompileOutput: "main.cpp:1: error: expected ';'"}}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "1\n"}}}, runner, nil, time.Second)

	_, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if store.completedStatus != "compile_error" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "compile_error")
	}
	if store.completedCompile == "" {
		t.Fatal("CompleteSubmission() should persist compile output excerpts for compile errors")
	}
}

func TestProcessOneMapsWrongAnswerToFinalSubmissionStatus(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "cpp17", SourceCode: "wrong", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictWrongAnswer, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictWrongAnswer, Duration: 10 * time.Millisecond}}}}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "2\n"}}}, runner, nil, time.Second)

	_, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if store.completedStatus != "wrong_answer" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "wrong_answer")
	}
}

func TestProcessOneMapsTimeLimitExceededToFinalSubmissionStatus(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "python", SourceCode: "while True: pass", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &stubRunner{result: spike.Result{Verdict: spike.VerdictTimeLimitExceeded, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictTimeLimitExceeded, Duration: 1500 * time.Millisecond}}}}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "2\n"}}}, runner, nil, time.Second)

	_, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if store.completedStatus != "time_limit_exceeded" {
		t.Fatalf("CompleteSubmission() status = %q, want %q", store.completedStatus, "time_limit_exceeded")
	}
}

func TestProcessOneReturnsLoaderErrors(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "cpp17", SourceCode: "int main() {}", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	observer := &stubObserver{}
	processor := NewProcessor(store, &stubLoader{err: errors.New("missing bundle")}, &stubRunner{}, observer, time.Second)
	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessOne() should report that a claimed job was handled")
	}
	if store.failedID != "submission-id" {
		t.Fatalf("HandleJobFailure() received %q, want %q", store.failedID, "submission-id")
	}
	if store.failedError == "" {
		t.Fatal("HandleJobFailure() should receive a failure message")
	}
	if observer.retriedID != "submission-id" || observer.retryError == "" {
		t.Fatalf("observer retry event = %#v, want submission retry", observer)
	}
}

func TestProcessOneObservesTerminalFailure(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "cpp17", SourceCode: "int main() {}", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}, failureAction: FailureActionTerminal}
	observer := &stubObserver{}
	processor := NewProcessor(store, &stubLoader{err: errors.New("checksum mismatch")}, &stubRunner{}, observer, time.Second)

	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessOne() should report that a claimed job was handled")
	}
	if observer.terminalID != "submission-id" || observer.terminalError == "" {
		t.Fatalf("observer terminal event = %#v, want terminal failure", observer)
	}
}

func TestProcessOneRenewsLeaseWhileRunningLongSubmission(t *testing.T) {
	store := &stubStore{job: SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "python", SourceCode: "print(42)", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second}}
	runner := &blockingRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
		result:  spike.Result{Verdict: spike.VerdictAccepted, CaseResults: []spike.CaseResult{{Verdict: spike.VerdictAccepted, Duration: 5 * time.Millisecond}}},
	}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "1\n"}}}, runner, nil, 5*time.Millisecond)

	done := make(chan struct{})
	go func() {
		_, err := processor.ProcessOne(context.Background())
		if err != nil {
			t.Errorf("ProcessOne() error = %v", err)
		}
		close(done)
	}()

	<-runner.started
	time.Sleep(20 * time.Millisecond)
	close(runner.release)
	<-done

	if store.leaseRenewCalls == 0 {
		t.Fatal("ProcessOne() should renew the lease while a job is still running")
	}
	if store.leaseRenewedID != "submission-id" || store.leaseRenewedWith != "lease-token" {
		t.Fatalf("RenewJobLease() received id=%q token=%q", store.leaseRenewedID, store.leaseRenewedWith)
	}
}

func TestProcessOneStopsWhenLeaseIsLost(t *testing.T) {
	store := &stubStore{
		job:           SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "python", SourceCode: "print(42)", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second},
		leaseRenewErr: ErrJobLeaseLost,
	}
	runner := &blockingRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "1\n"}}}, runner, nil, 5*time.Millisecond)

	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessOne() should report that a claimed job was handled")
	}
	if store.completedID != "" {
		t.Fatalf("CompleteSubmission() should not be called after lease loss, got %q", store.completedID)
	}
	if store.failedID != "" {
		t.Fatalf("HandleJobFailure() should not be called after lease loss, got %q", store.failedID)
	}
	if store.leaseRenewCalls == 0 {
		t.Fatal("RenewJobLease() should be attempted before lease loss is handled")
	}
}

func TestProcessOneRecordsHeartbeatFailures(t *testing.T) {
	store := &stubStore{
		job:           SubmissionJob{SubmissionID: "submission-id", LeaseToken: "lease-token", Language: "python", SourceCode: "print(42)", BundleKey: "bundle.json", BundleSHA256: "bundle-sha", TimeLimit: time.Second},
		leaseRenewErr: errors.New("db unavailable"),
	}
	runner := &blockingRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	processor := NewProcessor(store, &stubLoader{cases: []spike.TestCase{{Input: "1\n", ExpectedOutput: "1\n"}}}, runner, nil, 5*time.Millisecond)

	processed, err := processor.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessOne() should report that a claimed job was handled")
	}
	if store.failedID != "submission-id" {
		t.Fatalf("HandleJobFailure() received %q, want %q", store.failedID, "submission-id")
	}
	if !strings.Contains(store.failedError, "renew submission job lease") {
		t.Fatalf("HandleJobFailure() error = %q, want lease renewal context", store.failedError)
	}
}
