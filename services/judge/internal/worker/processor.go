package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

var ErrNoJobs = errors.New("no queued submissions")

const defaultLeaseRenewInterval = 10 * time.Second

const maxArtifactExcerptRunes = 4000

type SubmissionJob struct {
	SubmissionID string
	LeaseToken   string
	Language     string
	SourceCode   string
	BundleKey    string
	BundleSHA256 string
	TimeLimit    time.Duration
}

type CaseResult struct {
	Verdict         string
	ExecutionTimeMS int
	StdoutExcerpt   string
	StderrExcerpt   string
}

type Store interface {
	ClaimNextJob(context.Context) (SubmissionJob, error)
	RenewJobLease(context.Context, string, string) error
	MarkSubmissionRunning(context.Context, string) error
	HandleJobFailure(context.Context, string, string, string) (FailureAction, error)
	CompleteSubmission(context.Context, string, string, string, string, []CaseResult) error
}

type Observer interface {
	Heartbeat()
	JobClaimed(string, string)
	JobCompleted(string, string)
	JobRetried(string, string)
	JobTerminalFailure(string, string)
	NoJobs()
}

type Evaluator interface {
	Evaluate(context.Context, spike.Request) (spike.Result, error)
}

type Processor struct {
	store              Store
	loader             BundleLoader
	runner             Evaluator
	observer           Observer
	leaseRenewInterval time.Duration
}

func NewProcessor(store Store, loader BundleLoader, runner Evaluator, observer Observer, leaseRenewInterval time.Duration) Processor {
	if leaseRenewInterval <= 0 {
		leaseRenewInterval = defaultLeaseRenewInterval
	}

	if observer == nil {
		observer = noopObserver{}
	}

	return Processor{store: store, loader: loader, runner: runner, observer: observer, leaseRenewInterval: leaseRenewInterval}
}

func (processor Processor) ProcessOne(ctx context.Context) (bool, error) {
	processor.observer.Heartbeat()
	job, err := processor.store.ClaimNextJob(ctx)
	if err != nil {
		if errors.Is(err, ErrNoJobs) {
			processor.observer.NoJobs()
			return false, nil
		}
		return false, err
	}
	processor.observer.JobClaimed(job.SubmissionID, job.Language)

	processingCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	leaseErrs := make(chan error, 1)
	go func() {
		leaseErrs <- processor.keepLeaseAlive(processingCtx, cancel, job)
	}()

	completionStatus, err := processor.processClaimedJob(processingCtx, job)
	cancel()
	leaseErr := <-leaseErrs
	if leaseErr != nil && (err == nil || errors.Is(err, context.Canceled)) {
		err = leaseErr
	}

	if err != nil {
		if errors.Is(err, ErrJobLeaseLost) || errors.Is(err, context.Canceled) {
			if errors.Is(err, context.Canceled) && leaseErr == nil && ctx.Err() != nil {
				return false, nil
			}

			return true, nil
		}

		action, handleErr := processor.store.HandleJobFailure(ctx, job.SubmissionID, job.LeaseToken, err.Error())
		if handleErr != nil {
			if errors.Is(handleErr, ErrJobLeaseLost) {
				return true, nil
			}

			return false, fmt.Errorf("record failed submission job: %w", handleErr)
		}
		if action == FailureActionTerminal {
			processor.observer.JobTerminalFailure(job.SubmissionID, err.Error())
		} else {
			processor.observer.JobRetried(job.SubmissionID, err.Error())
		}

		return true, nil
	}
	processor.observer.JobCompleted(job.SubmissionID, completionStatus)

	return true, nil
}

func (processor Processor) processClaimedJob(ctx context.Context, job SubmissionJob) (string, error) {

	if err := processor.store.MarkSubmissionRunning(ctx, job.SubmissionID); err != nil {
		return "", err
	}

	cases, err := processor.loader.LoadCases(ctx, job.BundleKey, job.BundleSHA256)
	if err != nil {
		return "", err
	}

	language, err := spikeLanguage(job.Language)
	if err != nil {
		return "", err
	}

	result, err := processor.runner.Evaluate(ctx, spike.Request{
		Language:  language,
		Source:    job.SourceCode,
		TimeLimit: job.TimeLimit,
		Cases:     cases,
	})
	if err != nil {
		return "", err
	}

	status := submissionStatus(result.Verdict)
	if err := processor.store.CompleteSubmission(ctx, job.SubmissionID, job.LeaseToken, status, artifactExcerpt(result.CompileOutput), toCaseResults(result.CaseResults)); err != nil {
		return "", err
	}

	return status, nil
}

func (processor Processor) keepLeaseAlive(ctx context.Context, cancel context.CancelFunc, job SubmissionJob) error {
	ticker := time.NewTicker(processor.leaseRenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			processor.observer.Heartbeat()
			if err := processor.store.RenewJobLease(ctx, job.SubmissionID, job.LeaseToken); err != nil {
				cancel()
				if errors.Is(err, ErrJobLeaseLost) {
					return ErrJobLeaseLost
				}

				return fmt.Errorf("renew submission job lease: %w", err)
			}
		}
	}
}

type noopObserver struct{}

func (noopObserver) Heartbeat()                        {}
func (noopObserver) JobClaimed(string, string)         {}
func (noopObserver) JobCompleted(string, string)       {}
func (noopObserver) JobRetried(string, string)         {}
func (noopObserver) JobTerminalFailure(string, string) {}
func (noopObserver) NoJobs()                           {}

func spikeLanguage(language string) (spike.Language, error) {
	switch language {
	case "cpp17":
		return spike.LanguageCPP17, nil
	case "python":
		return spike.LanguagePython, nil
	default:
		return "", fmt.Errorf("unsupported language %q", language)
	}
}

func submissionStatus(verdict spike.Verdict) string {
	switch verdict {
	case spike.VerdictAccepted:
		return "accepted"
	case spike.VerdictWrongAnswer:
		return "wrong_answer"
	case spike.VerdictCompileError:
		return "compile_error"
	case spike.VerdictRuntimeError:
		return "runtime_error"
	case spike.VerdictTimeLimitExceeded:
		return "time_limit_exceeded"
	default:
		return "runtime_error"
	}
}

func toCaseResults(results []spike.CaseResult) []CaseResult {
	converted := make([]CaseResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, CaseResult{
			Verdict:         submissionStatus(result.Verdict),
			ExecutionTimeMS: int(result.Duration.Milliseconds()),
			StdoutExcerpt:   artifactExcerpt(result.Stdout),
			StderrExcerpt:   artifactExcerpt(result.Stderr),
		})
	}

	return converted
}

func artifactExcerpt(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxArtifactExcerptRunes {
		return value
	}

	return string(runes[:maxArtifactExcerptRunes]) + "\n...[truncated]"
}
