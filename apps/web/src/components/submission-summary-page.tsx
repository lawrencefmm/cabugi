"use client";

import { useQuery } from "@tanstack/react-query";

import { fetchSubmission, formatQueuedAt, formatSubmissionLanguage, type SubmissionDetail, type SubmissionStatus } from "../lib/api";
import { SignInButton, useAuth } from "./auth";
import { VerdictBadge } from "./verdict-badge";

type SubmissionSummaryPageProps = {
  authEnabled: boolean;
  submissionId: string;
};

export function SubmissionSummaryPage({ authEnabled, submissionId }: SubmissionSummaryPageProps) {
  if (!authEnabled) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Submission view unavailable</h1>
          <p className="empty-state__text">Frontend authentication is not configured in this environment yet.</p>
        </section>
      </main>
    );
  }

  return <AuthenticatedSubmissionSummaryPage submissionId={submissionId} />;
}

function AuthenticatedSubmissionSummaryPage({ submissionId }: { submissionId: string }) {
  const { getToken, isLoaded, isSignedIn } = useAuth();

  const submissionQuery = useQuery({
    enabled: isLoaded && isSignedIn,
    queryKey: ["submission-summary", submissionId],
    queryFn: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error("Missing submission token");
      }

      return fetchSubmission(submissionId, token);
    },
  });

  return (
    <main className="app-main">
      <section className="page-hero">
        <a className="status-note" href="/submissions">
          Submissions / {submissionId}
        </a>
        <span className="page-kicker">Submission</span>
        <h1 className="page-title submission-identifier">{submissionId}</h1>
        <p className="page-subtitle">Inspect the stored verdict, aggregate counts, and per-test output for this submission.</p>
      </section>

      {!isLoaded ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading submission</h2>
        </section>
      ) : null}

      {isLoaded && !isSignedIn ? (
        <section className="empty-state">
          <h2 className="empty-state__title">Sign in to view this submission</h2>
          <div className="history-actions">
            <SignInButton mode="modal">
              <button className="workspace-button" type="button">
                Sign in to continue
              </button>
            </SignInButton>
          </div>
        </section>
      ) : null}

      {submissionQuery.error ? (
        <section className="error-state" role="alert">
          <h2 className="error-state__title">Submission unavailable</h2>
          <p className="error-state__text">Unable to load this submission right now.</p>
        </section>
      ) : null}

      {isLoaded && isSignedIn && submissionQuery.isPending ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading submission</h2>
        </section>
      ) : null}

      {submissionQuery.data ? (
        <section className="table-shell submission-detail">
          <div className="submission-detail__header">
            <div>
              <a className="table-link" href={`/problems/${submissionQuery.data.problemSlug}`}>
                {submissionQuery.data.problemSlug}
              </a>
              <p className="detail-meta mono">{formatSubmissionLanguage(submissionQuery.data.language)} • Queued {formatQueuedAt(submissionQuery.data.queuedAt)}</p>
            </div>

            <VerdictBadge verdict={submissionQuery.data.status} />
          </div>

          <div className="submission-stats" aria-label="Submission summary">
            <article className="submission-stat">
              <p className="submission-stat__label">Final verdict</p>
              <div className="submission-stat__value">
                <VerdictBadge verdict={submissionQuery.data.status} />
              </div>
            </article>

            <article className="submission-stat">
              <p className="submission-stat__label">Passed tests</p>
              <p className="submission-stat__value">{submissionQuery.data.passedTests}</p>
            </article>

            <article className="submission-stat">
              <p className="submission-stat__label">Total tests</p>
              <p className="submission-stat__value">{submissionQuery.data.totalTests}</p>
            </article>
          </div>

          {submissionQuery.data.compileOutputExcerpt ? <SubmissionDiagnostics compileOutputExcerpt={submissionQuery.data.compileOutputExcerpt} /> : null}

          <SubmissionResults submission={submissionQuery.data} />
        </section>
      ) : null}
    </main>
  );
}

function SubmissionResults({ submission }: { submission: SubmissionDetail }) {
  if (submission.results.length === 0) {
    return (
      <section className="submission-results-empty" aria-live="polite">
        <h2 className="submission-results-empty__title">Per-test results unavailable</h2>
        <p className="submission-results-empty__text">{emptyResultsMessage(submission.status)}</p>
      </section>
    );
  }

  return (
    <section className="submission-results" aria-label="Per-test results">
      <h2 className="submission-results__title">Per-test results</h2>
      <ol className="submission-results__list">
        {submission.results.map((result) => (
          <li className="submission-result" key={result.testIndex}>
            <div className="submission-result__header">
              <div>
                <p className="submission-result__title">Test {result.testIndex + 1}</p>
                <p className="submission-result__meta mono">Execution time: {result.executionTimeMs} ms</p>
              </div>

              <VerdictBadge verdict={result.verdict} />
            </div>

            {result.stdoutExcerpt ? (
              <ResultStream heading="Stdout excerpt" value={result.stdoutExcerpt} />
            ) : null}

            {result.stderrExcerpt ? (
              <ResultStream heading="Stderr excerpt" value={result.stderrExcerpt} />
            ) : null}
          </li>
        ))}
      </ol>
    </section>
  );
}

function SubmissionDiagnostics({ compileOutputExcerpt }: { compileOutputExcerpt: string }) {
  return (
    <section className="submission-results" aria-label="Submission diagnostics">
      <h2 className="submission-results__title">Compile output</h2>
      <ResultStream heading="Compiler excerpt" value={compileOutputExcerpt} />
    </section>
  );
}

function ResultStream({ heading, value }: { heading: string; value: string }) {
  return (
    <div className="submission-result__stream">
      <p className="submission-result__stream-label">{heading}</p>
      <pre className="submission-result__stream-value">{value}</pre>
    </div>
  );
}

function emptyResultsMessage(status: SubmissionStatus) {
  if (status === "compile_error") {
    return "No per-test results were recorded because compilation failed before execution started.";
  }

  if (status === "judge_failed") {
    return "The judge infrastructure could not finish this submission after repeated attempts.";
  }

  if (status === "queued" || status === "running") {
    return "Per-test results will appear after judging finishes.";
  }

  return "This submission finished without individual test-case rows.";
}
