"use client";

import { SignInButton, useAuth } from "@clerk/nextjs";
import { useQuery } from "@tanstack/react-query";

import { fetchSubmissions, formatQueuedAt, formatSubmissionStatus, type Submission } from "../lib/api";

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
        <span className="page-kicker">Submission History</span>
        <h1 className="page-title">Track every recent attempt.</h1>
        <p className="page-subtitle">
          Review queued, running, and finished submissions in one place so you can move between problems without losing context.
        </p>
      </section>

      {!isLoaded ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading history</h2>
          <p className="empty-state__text">Checking authentication and fetching your recent submissions.</p>
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
          <p className="error-state__text">Unable to load submission history from the Cabugi API right now.</p>
        </section>
      ) : null}

      {submissionsQuery.data && submissionsQuery.data.length === 0 ? (
        <section className="empty-state">
          <h2 className="empty-state__title">No submissions yet</h2>
          <p className="empty-state__text">Once you submit a solution, it will appear here with its latest verdict.</p>
        </section>
      ) : null}

      {submissionsQuery.data && submissionsQuery.data.length > 0 ? <SubmissionHistoryList submissions={submissionsQuery.data} /> : null}
    </main>
  );
}

function SubmissionHistoryList({ submissions }: { submissions: Submission[] }) {
  return (
    <section className="history-list" aria-label="Submission history">
      {submissions.map((submission) => (
        <article className="history-card" key={submission.id}>
          <div className="history-card__header">
            <div>
              <a className="history-card__problem-link" href={`/problems/${submission.problemSlug}`}>
                {submission.problemSlug}
              </a>
              <p className="history-card__meta">{submission.language === "cpp17" ? "C++17" : "Python"}</p>
            </div>

            <a className="history-card__detail-link" href={`/submissions/${submission.id}`}>
              View submission
            </a>
          </div>

          <div className="history-card__footer">
            <span className={`workspace-status workspace-status--${submission.status}`}>
              <span className="workspace-status__label">Status</span>
              <span className="workspace-status__value">{formatSubmissionStatus(submission.status)}</span>
            </span>
            <span className="history-card__timestamp">{formatQueuedAt(submission.queuedAt)}</span>
          </div>
        </article>
      ))}
    </section>
  );
}
