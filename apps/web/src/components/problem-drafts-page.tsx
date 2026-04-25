"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { fetchProblemDrafts, formatDateTime, type ProblemDraftLifecycleStatus, type ProblemDraftSummary } from "../lib/api";
import { SignInButton, useAuth } from "./auth";

type ProblemDraftsPageProps = {
  authEnabled: boolean;
};

export function ProblemDraftsPage({ authEnabled }: ProblemDraftsPageProps) {
  if (!authEnabled) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Drafts unavailable</h1>
          <p className="empty-state__text">Frontend authentication is not configured in this environment yet.</p>
        </section>
      </main>
    );
  }

  return <AuthenticatedProblemDraftsPage />;
}

function AuthenticatedProblemDraftsPage() {
  const { getToken, isLoaded, isSignedIn } = useAuth();

  const draftsQuery = useQuery({
    enabled: isLoaded && isSignedIn,
    queryKey: ["problem-drafts"],
    queryFn: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error("Missing problem draft token");
      }

      return fetchProblemDrafts(token);
    },
  });

  return (
    <main className="app-main">
      <section className="page-hero">
        <span className="page-kicker">Drafts</span>
        <h1 className="page-title">Your authored drafts</h1>
        <p className="page-subtitle">Browse drafts you are still editing or waiting to get through moderation, then jump back into the authoring flow from one place.</p>
        <div className="history-actions">
          <Link className="table-link" href="/drafts/new">
            New draft
          </Link>
        </div>
      </section>

      {!isLoaded ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading drafts</h2>
        </section>
      ) : null}

      {isLoaded && !isSignedIn ? (
        <section className="empty-state">
          <h2 className="empty-state__title">Sign in to manage drafts</h2>
          <p className="empty-state__text">Draft creation and editing are tied to your Cabugi account.</p>
          <div className="history-actions">
            <SignInButton mode="modal">
              <button className="workspace-button" type="button">
                Sign in to continue
              </button>
            </SignInButton>
          </div>
        </section>
      ) : null}

      {draftsQuery.error ? (
        <section className="error-state" role="alert">
          <h2 className="error-state__title">Drafts unavailable</h2>
          <p className="error-state__text">Unable to load your drafts right now.</p>
        </section>
      ) : null}

      {draftsQuery.data && draftsQuery.data.length === 0 ? (
        <section className="empty-state">
          <h2 className="empty-state__title">No drafts yet</h2>
          <p className="empty-state__text">Start a draft problem to begin writing statements, uploading hidden tests, and preparing work for moderation.</p>
          <div className="history-actions">
            <Link className="workspace-button" href="/drafts/new">
              Create your first draft
            </Link>
          </div>
        </section>
      ) : null}

      {draftsQuery.data && draftsQuery.data.length > 0 ? <ProblemDraftList drafts={draftsQuery.data} /> : null}
    </main>
  );
}

function ProblemDraftList({ drafts }: { drafts: ProblemDraftSummary[] }) {
  return (
    <section className="table-shell">
      <div className="panel-toolbar">
        <span className="panel-toolbar__meta mono">{drafts.length} drafts</span>
      </div>

      <div className="table-wrap">
        <table className="data-table" aria-label="Problem drafts">
          <thead>
            <tr>
              <th scope="col">Title</th>
              <th scope="col">Slug</th>
              <th scope="col">Status</th>
              <th scope="col">Updated</th>
              <th scope="col">Submitted</th>
              <th scope="col">Action</th>
            </tr>
          </thead>
          <tbody>
            {drafts.map((draft) => (
              <tr key={`${draft.slug}-${draft.versionNumber}`}>
                <td>
                  <Link className="table-link" href={`/drafts/${draft.slug}`}>
                    {draft.title}
                  </Link>
                </td>
                <td className="table-code">{draft.slug}</td>
                <td>
                  <span className={`draft-status draft-status--${draft.lifecycleStatus}`}>{formatDraftStatus(draft.lifecycleStatus)}</span>
                </td>
                <td className="table-code">{formatDateTime(draft.updatedAt)}</td>
                <td className="table-code">{draft.submittedForReviewAt ? formatDateTime(draft.submittedForReviewAt) : "Not submitted"}</td>
                <td className="table-actions">
                  <Link className="table-link" href={`/drafts/${draft.slug}`}>
                    {draft.lifecycleStatus === "draft" ? "Resume editing" : "Open draft"}
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function formatDraftStatus(status: ProblemDraftLifecycleStatus) {
  switch (status) {
    case "draft":
      return "Draft";
    case "in_review":
      return "In Review";
    case "published":
      return "Published";
    case "archived":
      return "Archived";
  }
}
