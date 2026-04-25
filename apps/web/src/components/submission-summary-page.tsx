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
import { starterCodeTemplates } from "../lib/starter-code";
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
        <section className="submission-detail submission-detail--command">
          <div className="submission-command-bar">
            <Link className="status-note status-note--ghost" href="/submissions">
              Back to submissions
            </Link>
            <div className="history-actions history-actions--flush">
              <Link className="table-link" href={`/problems/${submission.problemSlug}`}>
                Back to problem
              </Link>
              <Link className="table-link" href="/submissions">
                Submission history
              </Link>
            </div>
          </div>

          <div className="submission-detail__headline">
            <div>
              <h2 className="page-title submission-identifier">Submission #{shortSubmissionId(submission.id)}</h2>
              <p className="detail-meta">{submission.problemSlug} · {formatSubmissionLanguage(submission.language)} · Submitted {formatQueuedAt(submission.queuedAt)}</p>
            </div>
            <VerdictBadge verdict={submission.status} />
          </div>

          {isLiveSubmission ? (
            <section className="submission-live-note" aria-live="polite">
              <h2 className="submission-live-note__title">Live status stream</h2>
              <p className="submission-live-note__text">The judge is still processing this run. Verdict and per-test sections will refresh automatically until a final result lands.</p>
            </section>
          ) : null}

          <div className="submission-top-cards" aria-label="Submission summary">
            <SubmissionStatCard label="Queue Status" value={isLiveSubmission ? "Processing" : "Processed"} detail={formatQueuedAt(submission.queuedAt)} />
            <SubmissionStatCard label="Final verdict" value={formatSubmissionStatus(submission.status)} detail={isLiveSubmission ? "Waiting for judge" : "All available results stored"} />
            <SubmissionStatCard label="Score" value={`${scoreFor(submission)} / 100`} detail="Full score" />
            <SubmissionStatCard label="Submission ID" value={`#${shortSubmissionId(submission.id)}`} detail={submission.id} />
          </div>

          <div className="submission-command-grid">
            <section className={`submission-verdict-card submission-verdict-card--${submission.status}`}>
              <div className="submission-verdict-card__status">
                <span aria-hidden="true" />
                <div>
                  <h2>{formatSubmissionStatus(submission.status)}</h2>
                  <p>{submission.passedTests} of {submission.totalTests} testcases passed.</p>
                </div>
              </div>
              <div className="submission-stats" aria-label="Submission totals">
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
            </section>

            <SubmissionResults submission={submission} />

            <section className="code-snapshot-panel">
              <div className="panel-toolbar">
                <h2 className="submission-results__title">Code Snapshot</h2>
                <span className="panel-toolbar__meta mono">{formatSubmissionLanguage(submission.language)}</span>
              </div>
              <pre className="code-snapshot-panel__code">{starterCodeTemplates[submission.language]}</pre>
            </section>

            {submission.compileOutputExcerpt ? <SubmissionDiagnostics compileOutputExcerpt={submission.compileOutputExcerpt} /> : null}

            <section className="performance-panel">
              <h2 className="submission-results__title">Performance Distribution</h2>
              <div className="performance-panel__body">
                <div>
                  <strong>{fastestResultMs(submission)} ms</strong>
                  <span>Your Time</span>
                </div>
                <div className="performance-bars" aria-hidden="true">
                  {Array.from({ length: 18 }, (_, index) => <span key={index} className={index === 7 ? "performance-bars__you" : ""} />)}
                </div>
                <div>
                  <strong>{maxMemoryFor(submission)}</strong>
                  <span>Memory</span>
                </div>
              </div>
            </section>
          </div>
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
    <section className="submission-results submission-results--table" aria-label="Per-test results">
      <div className="panel-toolbar">
        <h2 className="submission-results__title">Testcases</h2>
        <span className="panel-toolbar__meta mono">({submission.passedTests} / {submission.totalTests} passed)</span>
      </div>
      <div className="table-wrap">
        <table className="data-table data-table--compact">
          <thead>
            <tr>
              <th scope="col">#</th>
              <th scope="col">Input Group</th>
              <th scope="col">Verdict</th>
              <th scope="col">Time</th>
              <th scope="col">Memory</th>
            </tr>
          </thead>
          <tbody>
            {submission.results.map((result) => (
              <tr key={result.testIndex}>
                <td className="table-code">{result.testIndex + 1}</td>
                <td>Test {result.testIndex + 1}</td>
                <td><VerdictBadge verdict={result.verdict} /></td>
                <td className="table-code">{result.executionTimeMs} ms</td>
                <td className="table-code">{formatBytes(result.memoryBytes)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="submission-results__streams">
        {submission.results.map((result) => (
          <div className="submission-result" key={`${result.testIndex}-streams`}>
            {result.stdoutExcerpt ? <ResultStream heading="Stdout excerpt" value={result.stdoutExcerpt} /> : null}
            {result.stderrExcerpt ? <ResultStream heading="Stderr excerpt" value={result.stderrExcerpt} /> : null}
          </div>
        ))}
      </div>
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

function SubmissionStatCard({ detail, label, value }: { detail: string; label: string; value: string }) {
  return (
    <article className="submission-top-card">
      <p className="submission-stat__label">{label}</p>
      <p className="submission-stat__value">{value}</p>
      <p className="detail-meta">{detail}</p>
    </article>
  );
}

function shortSubmissionId(id: string) {
  return id.length > 10 ? id.slice(0, 10) : id;
}

function scoreFor(submission: SubmissionDetail) {
  if (submission.totalTests <= 0) {
    return 0;
  }

  return Math.round((submission.passedTests / submission.totalTests) * 100);
}

function fastestResultMs(submission: SubmissionDetail) {
  if (submission.results.length === 0) {
    return 0;
  }

  return Math.min(...submission.results.map((result) => result.executionTimeMs));
}

function maxMemoryFor(submission: SubmissionDetail) {
  const maxMemory = Math.max(0, ...submission.results.map((result) => result.memoryBytes));
  return formatBytes(maxMemory);
}

function formatBytes(bytes: number) {
  if (bytes <= 0) {
    return "-";
  }

  const mb = bytes / 1024 / 1024;
  if (mb >= 1) {
    return `${mb.toFixed(1)} MB`;
  }

  const kb = bytes / 1024;
  return `${kb.toFixed(1)} KB`;
}

function formatSubmissionStatus(status: SubmissionStatus) {
  return status
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
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
