"use client";

import { SignInButton, useAuth } from "@clerk/nextjs";
import { useQuery } from "@tanstack/react-query";

import { fetchSubmission, formatQueuedAt, formatSubmissionStatus } from "../lib/api";

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
          Back to submission history
        </a>
        <span className="page-kicker">Submission</span>
        <h1 className="page-title">{submissionId}</h1>
      </section>

      {!isLoaded ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading submission</h2>
          <p className="empty-state__text">Checking authentication and fetching submission details.</p>
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

      {submissionQuery.data ? (
        <section className="history-card">
          <div className="history-card__header">
            <div>
              <a className="history-card__problem-link" href={`/problems/${submissionQuery.data.problemSlug}`}>
                {submissionQuery.data.problemSlug}
              </a>
              <p className="history-card__meta">{submissionQuery.data.language === "cpp17" ? "C++17" : "Python"}</p>
            </div>
          </div>

          <div className="history-card__footer">
            <span className={`workspace-status workspace-status--${submissionQuery.data.status}`}>
              <span className="workspace-status__label">Status</span>
              <span className="workspace-status__value">{formatSubmissionStatus(submissionQuery.data.status)}</span>
            </span>
            <span className="history-card__timestamp">{formatQueuedAt(submissionQuery.data.queuedAt)}</span>
          </div>
        </section>
      ) : null}
    </main>
  );
}
