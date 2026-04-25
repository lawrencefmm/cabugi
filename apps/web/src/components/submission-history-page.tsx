"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { fetchSubmissions, formatQueuedAt, formatSubmissionLanguage, type Submission } from "../lib/api";
import { SignInButton, useAuth } from "./auth";
import { VerdictBadge } from "./verdict-badge";

type SubmissionHistoryPageProps = {
  authEnabled: boolean;
};

export function SubmissionHistoryPage({ authEnabled }: SubmissionHistoryPageProps) {
  if (!authEnabled) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Submission history unavailable</h1>
          <p className="empty-state__text">Frontend authentication is not configured in this environment yet.</p>
        </section>
      </main>
    );
  }

  return <AuthenticatedSubmissionHistoryPage />;
}

function AuthenticatedSubmissionHistoryPage() {
  const { getToken, isLoaded, isSignedIn } = useAuth();

  const submissionsQuery = useQuery({
    enabled: isLoaded && isSignedIn,
    queryKey: ["submission-history"],
    queryFn: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error("Missing submission token");
      }

      return fetchSubmissions(token);
    },
  });

  return (
    <main className="app-main">
      <section className="page-hero">
        <span className="page-kicker">Submissions</span>
        <h1 className="page-title">Recent attempts</h1>
        <p className="page-subtitle">Review queued, running, and finished submissions in a single dense history view.</p>
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

      {isLoaded && isSignedIn && submissionsQuery.isPending ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading submission history</h2>
        </section>
      ) : null}

      {isLoaded && !isSignedIn ? (
        <section className="empty-state">
          <h2 className="empty-state__title">Sign in to view your submissions</h2>
          <p className="empty-state__text">Submission history is personal to your account. Sign in to continue.</p>
          <div className="history-actions">
            <SignInButton mode="modal">
              <button className="workspace-button" type="button">
                Sign in to continue
              </button>
            </SignInButton>
          </div>
        </section>
      ) : null}

      {submissionsQuery.error ? (
        <section className="error-state" role="alert">
          <h2 className="error-state__title">History unavailable</h2>
          <p className="error-state__text">Unable to load submission history right now.</p>
          <div className="history-actions">
            <button className="workspace-button workspace-button--secondary" onClick={() => void submissionsQuery.refetch()} type="button">
              Retry history
            </button>
          </div>
        </section>
      ) : null}

      {submissionsQuery.data && submissionsQuery.data.length === 0 ? (
        <section className="empty-state">
          <h2 className="empty-state__title">No submissions yet</h2>
          <p className="empty-state__text">Once you submit a solution, it will appear here with its latest verdict.</p>
          <div className="history-actions">
            <Link className="table-link" href="/">
              Browse problems
            </Link>
          </div>
        </section>
      ) : null}

      {submissionsQuery.data && submissionsQuery.data.length > 0 ? <SubmissionHistoryList submissions={submissionsQuery.data} /> : null}
    </main>
  );
}

function SubmissionHistoryList({ submissions }: { submissions: Submission[] }) {
  return (
    <section className="table-shell">
      <div className="panel-toolbar">
        <span className="panel-toolbar__meta mono">{submissions.length} submissions</span>
      </div>

      <div className="table-wrap responsive-table">
        <table className="data-table" aria-label="Submission history">
          <thead>
            <tr>
              <th scope="col">Submission</th>
              <th scope="col">Problem</th>
              <th scope="col">Language</th>
              <th scope="col">Verdict</th>
              <th scope="col">Queued</th>
              <th scope="col">Action</th>
            </tr>
          </thead>
          <tbody>
            {submissions.map((submission) => (
              <tr key={submission.id}>
                <td className="table-code">{submission.id}</td>
                <td>
                  <Link className="table-link" href={`/problems/${submission.problemSlug}`}>
                    {submission.problemSlug}
                  </Link>
                </td>
                <td className="table-code">{formatSubmissionLanguage(submission.language)}</td>
                <td>
                  <VerdictBadge verdict={submission.status} />
                </td>
                <td className="table-code">{formatQueuedAt(submission.queuedAt)}</td>
                <td className="table-actions">
                  <Link className="table-link" href={`/submissions/${submission.id}`}>
                    View submission
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mobile-card-list" aria-label="Submission history cards">
        {submissions.map((submission) => (
          <article className="history-card" key={`${submission.id}-card`}>
            <div className="history-card__header">
              <div>
                <h2 className="workspace-panel__title">{submission.problemSlug}</h2>
                <p className="detail-meta mono">{submission.id}</p>
              </div>

              <VerdictBadge verdict={submission.status} />
            </div>

            <div className="history-card__body submission-stats" aria-label={`${submission.id} details`}>
              <article className="submission-stat">
                <p className="submission-stat__label">Language</p>
                <p className="submission-stat__value">{formatSubmissionLanguage(submission.language)}</p>
              </article>

              <article className="submission-stat">
                <p className="submission-stat__label">Queued</p>
                <p className="submission-stat__value">{formatQueuedAt(submission.queuedAt)}</p>
              </article>
            </div>

            <div className="history-card__footer">
              <Link aria-label={`Open problem ${submission.problemSlug}`} className="table-link" href={`/problems/${submission.problemSlug}`}>
                Open problem
              </Link>
              <Link aria-label={`Open submission ${submission.id}`} className="table-link" href={`/submissions/${submission.id}`}>
                Open submission
              </Link>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
