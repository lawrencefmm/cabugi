package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

var ErrNoJobs = errors.New("no queued submissions")

type SubmissionJob struct {
	SubmissionID string
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
	MarkSubmissionRunning(context.Context, string) error
	CompleteSubmission(context.Context, string, string, []CaseResult) error
}

type Evaluator interface {
	Evaluate(context.Context, spike.Request) (spike.Result, error)
}

type Processor struct {
	store  Store
	loader BundleLoader
	runner Evaluator
}

func NewProcessor(store Store, loader BundleLoader, runner Evaluator) Processor {
	return Processor{store: store, loader: loader, runner: runner}
}

func (processor Processor) ProcessOne(ctx context.Context) (bool, error) {
	job, err := processor.store.ClaimNextJob(ctx)
	if err != nil {
		if errors.Is(err, ErrNoJobs) {
			return false, nil
		}
		return false, err
	}

	if err := processor.store.MarkSubmissionRunning(ctx, job.SubmissionID); err != nil {
		return false, err
	}

	cases, err := processor.loader.LoadCases(ctx, job.BundleKey, job.BundleSHA256)
	if err != nil {
		return false, err
	}

	language, err := spikeLanguage(job.Language)
	if err != nil {
		return false, err
	}

	result, err := processor.runner.Evaluate(ctx, spike.Request{
		Language:  language,
		Source:    job.SourceCode,
		TimeLimit: job.TimeLimit,
		Cases:     cases,
	})
	if err != nil {
		return false, err
	}

	if err := processor.store.CompleteSubmission(ctx, job.SubmissionID, submissionStatus(result.Verdict), toCaseResults(result.CaseResults)); err != nil {
		return false, err
	}

	return true, nil
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
