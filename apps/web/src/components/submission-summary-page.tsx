"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import {
  ApiError,
  fetchSubmission,
  formatQueuedAt,
  formatSubmissionLanguage,
  isTerminalSubmissionStatus,
  type SubmissionDetail,
  type SubmissionStatus,
} from "../lib/api";
import { SignInButton, useAuth } from "./auth";
import { VerdictBadge } from "./verdict-badge";

type SubmissionSummaryPageProps = {
  authEnabled: boolean;
  submissionId: string;
  pollIntervalMs?: number;
};

export function SubmissionSummaryPage({ authEnabled, pollIntervalMs = 1500, submissionId }: SubmissionSummaryPageProps) {
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

  return <AuthenticatedSubmissionSummaryPage pollIntervalMs={pollIntervalMs} submissionId={submissionId} />;
}

function AuthenticatedSubmissionSummaryPage({ pollIntervalMs, submissionId }: { pollIntervalMs: number; submissionId: string }) {
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
    refetchInterval: (query) => {
      const submission = query.state.data;
      if (!submission || isTerminalSubmissionStatus(submission.status)) {
        return false;
      }

      return pollIntervalMs;
    },
  });

  const submission = submissionQuery.data;
  const isLiveSubmission = submission ? !isTerminalSubmissionStatus(submission.status) : false;

  return (
    <main className="app-main">
      <section className="page-hero">
        <Link className="status-note" href="/submissions">
          Submissions / {submissionId}
        </Link>
        <span className="page-kicker">{isLiveSubmission ? "Live Submission" : "Submission"}</span>
        <h1 className="page-title submission-identifier">{submissionId}</h1>
        <p className="page-subtitle">
          {isLiveSubmission
            ? "Judging is still in flight. This page refreshes automatically until the verdict reaches a terminal state."
            : "Inspect the stored verdict, aggregate counts, and per-test output for this submission."}
        </p>
        {submission ? (
          <div className="meta-strip" aria-label="Submission metadata overview">
            <span className="meta-chip mono">problem {submission.problemSlug}</span>
            <span className="meta-chip mono">{formatSubmissionLanguage(submission.language)}</span>
            <span className="meta-chip mono">queued {formatQueuedAt(submission.queuedAt)}</span>
            {isLiveSubmission ? <span className="meta-chip mono">polling {formatPollingInterval(pollIntervalMs)}</span> : null}
          </div>
        ) : null}
      </section>

      {!isLoaded ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Checking submission access</h2>
          <p className="empty-state__text">Published problems stay available while your account finishes loading.</p>
          <div className="history-actions">
            <Link className="table-link" href="/">
              Browse problems
            </Link>
          </div>
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
          <p className="error-state__text">{formatSubmissionLoadError(submissionQuery.error)}</p>
          <div className="history-actions">
            <button className="workspace-button workspace-button--secondary" onClick={() => void submissionQuery.refetch()} type="button">
              Retry submission
            </button>
            <Link className="table-link" href="/submissions">
              Back to submissions
            </Link>
          </div>
        </section>
      ) : null}

      {isLoaded && isSignedIn && submissionQuery.isPending ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading stored submission</h2>
        </section>
      ) : null}

      {submission ? (
        <section className="table-shell submission-detail">
          <div className="submission-detail__header">
            <div>
              <Link className="table-link" href={`/problems/${submission.problemSlug}`}>
                {submission.problemSlug}
              </Link>
              <p className="detail-meta mono">{formatSubmissionLanguage(submission.language)} • Queued {formatQueuedAt(submission.queuedAt)}</p>
            </div>

            <VerdictBadge verdict={submission.status} />
          </div>

          <div className="history-actions">
            <Link className="table-link" href={`/problems/${submission.problemSlug}`}>
              Back to problem
            </Link>
            <Link className="table-link" href="/submissions">
              Submission history
            </Link>
          </div>

          {isLiveSubmission ? (
            <section className="submission-live-note" aria-live="polite">
              <h2 className="submission-live-note__title">Live status stream</h2>
              <p className="submission-live-note__text">The judge is still processing this run. Verdict and per-test sections will refresh automatically until a final result lands.</p>
            </section>
          ) : null}

          <div className="submission-stats" aria-label="Submission summary">
            <article className="submission-stat">
              <p className="submission-stat__label">Final verdict</p>
              <div className="submission-stat__value">
                <VerdictBadge verdict={submission.status} />
              </div>
            </article>

            <article className="submission-stat">
              <p className="submission-stat__label">Passed tests</p>
              <p className="submission-stat__value">{submission.passedTests}</p>
            </article>

            <article className="submission-stat">
              <p className="submission-stat__label">Total tests</p>
              <p className="submission-stat__value">{submission.totalTests}</p>
            </article>
          </div>

          {submission.compileOutputExcerpt ? <SubmissionDiagnostics compileOutputExcerpt={submission.compileOutputExcerpt} /> : null}

          <SubmissionResults submission={submission} />
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

            {result.stdoutExcerpt ? <ResultStream heading="Stdout excerpt" value={result.stdoutExcerpt} /> : null}

            {result.stderrExcerpt ? <ResultStream heading="Stderr excerpt" value={result.stderrExcerpt} /> : null}
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
    return "Per-test results will appear automatically after judging finishes.";
  }

  return "This submission finished without individual test-case rows.";
}

function formatSubmissionLoadError(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      return "Your session token is missing or expired. Sign in again and reload this submission.";
    }
    if (error.status === 404) {
      return "This submission is not available anymore, or it does not belong to the current account.";
    }
  }

  return "Unable to load this submission right now.";
}

function formatPollingInterval(pollIntervalMs: number) {
  if (pollIntervalMs < 1000) {
    return `${pollIntervalMs}ms`;
  }

  const seconds = pollIntervalMs / 1000;
  return Number.isInteger(seconds) ? `${seconds}s` : `${seconds.toFixed(1)}s`;
}
