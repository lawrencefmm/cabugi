package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

var ErrNoJobs = errors.New("no queued submissions")

const defaultLeaseRenewInterval = 10 * time.Second

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
	HandleJobFailure(context.Context, string, string, string) error
	CompleteSubmission(context.Context, string, string, string, []CaseResult) error
}

type Evaluator interface {
	Evaluate(context.Context, spike.Request) (spike.Result, error)
}

type Processor struct {
	store              Store
	loader             BundleLoader
	runner             Evaluator
	leaseRenewInterval time.Duration
}

func NewProcessor(store Store, loader BundleLoader, runner Evaluator, leaseRenewInterval time.Duration) Processor {
	if leaseRenewInterval <= 0 {
		leaseRenewInterval = defaultLeaseRenewInterval
	}

	return Processor{store: store, loader: loader, runner: runner, leaseRenewInterval: leaseRenewInterval}
}

func (processor Processor) ProcessOne(ctx context.Context) (bool, error) {
	job, err := processor.store.ClaimNextJob(ctx)
	if err != nil {
		if errors.Is(err, ErrNoJobs) {
			return false, nil
		}
		return false, err
	}

	processingCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	leaseErrs := make(chan error, 1)
	go func() {
		leaseErrs <- processor.keepLeaseAlive(processingCtx, cancel, job)
	}()

	err = processor.processClaimedJob(processingCtx, job)
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

		if handleErr := processor.store.HandleJobFailure(ctx, job.SubmissionID, job.LeaseToken, err.Error()); handleErr != nil {
			if errors.Is(handleErr, ErrJobLeaseLost) {
				return true, nil
			}

			return false, fmt.Errorf("record failed submission job: %w", handleErr)
		}

		return true, nil
	}

	return true, nil
}

func (processor Processor) processClaimedJob(ctx context.Context, job SubmissionJob) error {

	if err := processor.store.MarkSubmissionRunning(ctx, job.SubmissionID); err != nil {
		return err
	}

	cases, err := processor.loader.LoadCases(ctx, job.BundleKey, job.BundleSHA256)
	if err != nil {
		return err
	}

	language, err := spikeLanguage(job.Language)
	if err != nil {
		return err
	}

	result, err := processor.runner.Evaluate(ctx, spike.Request{
		Language:  language,
		Source:    job.SourceCode,
		TimeLimit: job.TimeLimit,
		Cases:     cases,
	})
	if err != nil {
		return err
	}

	if err := processor.store.CompleteSubmission(ctx, job.SubmissionID, job.LeaseToken, submissionStatus(result.Verdict), toCaseResults(result.CaseResults)); err != nil {
		return err
	}

	return nil
}

func (processor Processor) keepLeaseAlive(ctx context.Context, cancel context.CancelFunc, job SubmissionJob) error {
	ticker := time.NewTicker(processor.leaseRenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
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
			StdoutExcerpt:   result.Stdout,
			StderrExcerpt:   result.Stderr,
		})
	}

	return converted
}
