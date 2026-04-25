"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import {
  ApiError,
  applyModerationDecision,
  fetchModerationProblemDrafts,
  fetchProblemDraft,
  formatDraftError,
  formatMemoryLimit,
  formatModerationError,
  formatTimeLimit,
  type ModerationDecision,
  type ModerationQueueItem,
  type ProblemDraft,
} from "../lib/api";
import { localTestAuthProfileLabels, type LocalTestAuthProfile } from "../lib/local-test-auth";
import { SignInButton, useAuth } from "./auth";
import { ProblemMarkdown } from "./problem-markdown";

type ModerationProblemDraftsPageProps = {
  authEnabled: boolean;
};

export function ModerationProblemDraftsPage({ authEnabled }: ModerationProblemDraftsPageProps) {
  if (!authEnabled) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Moderation unavailable</h1>
          <p className="empty-state__text">Frontend authentication is not configured in this environment yet.</p>
        </section>
      </main>
    );
  }

  return <AuthenticatedModerationProblemDraftsPage />;
}

function AuthenticatedModerationProblemDraftsPage() {
  const { getToken, isLoaded, isSignedIn, localTestProfile, mode } = useAuth();
  const queryClient = useQueryClient();
  const [selectedSlug, setSelectedSlug] = useState<string | null>(null);
  const [moderationNotes, setModerationNotes] = useState("");
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const moderationQueueQuery = useQuery({
    enabled: isLoaded && isSignedIn,
    queryKey: ["moderation-problem-drafts"],
    queryFn: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error("Missing moderation token");
      }

      return fetchModerationProblemDrafts(token);
    },
  });

  useEffect(() => {
    if (!moderationQueueQuery.data) {
      return;
    }

    const stillExists = selectedSlug ? moderationQueueQuery.data.some((draft) => draft.slug === selectedSlug) : false;
    if (stillExists) {
      return;
    }

    setSelectedSlug(moderationQueueQuery.data[0]?.slug ?? null);
  }, [moderationQueueQuery.data, selectedSlug]);

  const selectedDraftQuery = useQuery({
    enabled: isLoaded && isSignedIn && selectedSlug !== null && !(moderationQueueQuery.error instanceof ApiError && moderationQueueQuery.error.status === 403),
    queryKey: ["moderation-problem-draft", selectedSlug],
    queryFn: async () => {
      const token = await getToken();
      if (!token || !selectedSlug) {
        throw new Error("Missing moderation draft token");
      }

      return fetchProblemDraft(selectedSlug, token);
    },
  });

  const moderationDecisionMutation = useMutation({
    mutationFn: async (decision: ModerationDecision) => {
      const token = await getToken();
      if (!token || !selectedSlug) {
        throw new Error("Missing moderation draft token");
      }

      return applyModerationDecision(
        selectedSlug,
        {
          decision,
          moderationNotes: moderationNotes.trim(),
        },
        token,
      );
    },
    onSuccess: async (draft) => {
      setSuccessMessage(`Decision applied: ${formatDecisionSuccess(draft.lifecycleStatus)}`);
      setModerationNotes("");
      setSelectedSlug(null);
      queryClient.setQueryData(["moderation-problem-draft", draft.slug], draft);
      await queryClient.invalidateQueries({ queryKey: ["moderation-problem-drafts"] });
    },
  });

  const moderationError = moderationDecisionMutation.error ?? selectedDraftQuery.error ?? null;

  if (!isLoaded) {
    return (
      <main className="app-main">
        <section className="empty-state" role="status" aria-live="polite">
          <h1 className="empty-state__title">Loading moderation tools</h1>
        </section>
      </main>
    );
  }

  if (!isSignedIn) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Sign in to moderate drafts</h1>
          <p className="empty-state__text">Moderation actions are restricted to authenticated staff accounts.</p>
          <div className="history-actions">
            <SignInButton mode="modal">
              <button className="workspace-button" type="button">
                Sign in to continue
              </button>
            </SignInButton>
          </div>
        </section>
      </main>
    );
  }

  if (moderationQueueQuery.isPending) {
    return (
      <main className="app-main">
        <section className="empty-state" role="status" aria-live="polite">
          <h1 className="empty-state__title">Loading moderation queue</h1>
        </section>
      </main>
    );
  }

  if (moderationQueueQuery.error) {
    const forbidden = moderationQueueQuery.error instanceof ApiError && moderationQueueQuery.error.status === 403;
    return (
      <main className="app-main">
        <section className="error-state" role="alert">
          <h1 className="error-state__title">{forbidden ? "Moderator access required" : "Moderation unavailable"}</h1>
          <p className="error-state__text">
            {formatModerationError(moderationQueueQuery.error, "Your account does not have moderator access.")}
          </p>

          {forbidden && mode === "local_test" ? <LocalModerationSetupPanel localTestProfile={localTestProfile} /> : null}
        </section>
      </main>
    );
  }

  const queue = moderationQueueQuery.data ?? [];
  const selectedDraft = selectedDraftQuery.data;

  return (
    <main className="app-main">
      <section className="page-hero">
        <span className="page-kicker">Moderation</span>
        <h1 className="page-title">Review submitted problem drafts.</h1>
        <p className="page-subtitle">Work through the moderation queue, inspect the full draft detail, and either publish, reject, or send the draft back for changes.</p>
      </section>

      {queue.length === 0 ? (
        <section className="empty-state">
          <h2 className="empty-state__title">Nothing to review</h2>
          <p className="empty-state__text">Drafts submitted for review will appear here once authors hand them off to moderators.</p>
        </section>
      ) : (
        <section className="moderation-layout">
          <aside className="workspace-panel moderation-queue">
            <div className="workspace-panel__header">
              <div>
                <h2 className="workspace-panel__title">Review Queue</h2>
                <p className="workspace-panel__subtitle">Oldest drafts appear first so moderation stays predictable.</p>
              </div>
            </div>

            <ul className="moderation-queue__list" aria-label="Moderation review queue">
              {queue.map((draft) => (
                <li key={`${draft.slug}-${draft.versionNumber}`}>
                  <button
                    className={`moderation-queue__button${draft.slug === selectedSlug ? " moderation-queue__button--active" : ""}`}
                    onClick={() => {
                      setSelectedSlug(draft.slug);
                      setSuccessMessage(null);
                    }}
                    type="button"
                  >
                    <span className="moderation-queue__title">{draft.title}</span>
                    <span className="moderation-queue__meta">{draft.slug}</span>
                    <span className="moderation-queue__meta">Submitted {formatSubmittedAt(draft)}</span>
                  </button>
                </li>
              ))}
            </ul>
          </aside>

          <section className="workspace-panel moderation-detail">
            {selectedDraftQuery.isPending ? (
              <section className="empty-state" role="status" aria-live="polite">
                <h2 className="empty-state__title">Loading draft detail</h2>
              </section>
            ) : null}

            {selectedDraftQuery.error ? (
              <section className="error-state" role="alert">
                <h2 className="error-state__title">Draft detail unavailable</h2>
                <p className="error-state__text">{formatDraftError(selectedDraftQuery.error, "Problem draft not found.")}</p>
              </section>
            ) : null}

            {selectedDraft ? (
              <>
                <div className="workspace-panel__header">
                  <div>
                    <h2 className="workspace-panel__title">{selectedDraft.title}</h2>
                    <p className="workspace-panel__subtitle">Review the statement, limits, and hidden bundle metadata before deciding what happens next.</p>
                  </div>

                  <span className={`draft-status draft-status--${selectedDraft.lifecycleStatus}`}>{formatDraftLifecycle(selectedDraft)}</span>
                </div>

                <div className="draft-meta-grid">
                  <article className="submission-stat">
                    <p className="submission-stat__label">Slug</p>
                    <p className="submission-stat__value">{selectedDraft.slug}</p>
                  </article>
                  <article className="submission-stat">
                    <p className="submission-stat__label">Time limit</p>
                    <p className="submission-stat__value">{formatTimeLimit(selectedDraft.timeLimitMs)}</p>
                  </article>
                  <article className="submission-stat">
                    <p className="submission-stat__label">Memory limit</p>
                    <p className="submission-stat__value">{formatMemoryLimit(selectedDraft.memoryLimitMb)}</p>
                  </article>
                </div>

                <div className="moderation-sections">
                  <ModerationSection heading="Statement" content={selectedDraft.statementMarkdown} />
                  <ModerationSection heading="Input" content={selectedDraft.inputMarkdown} />
                  <ModerationSection heading="Output" content={selectedDraft.outputMarkdown} />
                  <ModerationSection heading="Constraints" content={selectedDraft.constraintsMarkdown} />
                  <ModerationSection heading="Notes" content={selectedDraft.notesMarkdown} />
                </div>

                <section className="moderation-notes">
                  <DraftStatusLine label="Hidden bundle key" value={selectedDraft.hiddenTestBundleKey || "Not provided"} />
                  <DraftStatusLine label="Hidden bundle SHA-256" value={selectedDraft.hiddenTestBundleSha256 || "Not provided"} />

                  <label className="draft-field">
                    <span className="draft-field__label">Moderation notes</span>
                    <textarea value={moderationNotes} onChange={(event) => setModerationNotes(event.target.value)} />
                  </label>

                  {moderationError ? <p className="workspace-error" role="alert">{formatDraftError(moderationError, "Problem draft not found.")}</p> : null}
                  {successMessage ? <p className="draft-success">{successMessage}</p> : null}

                  <div className="moderation-actions">
                    <button
                      className="workspace-button"
                      disabled={moderationDecisionMutation.isPending || moderationNotes.trim() === ""}
                      onClick={() => moderationDecisionMutation.mutate("approve")}
                      type="button"
                    >
                      {moderationDecisionMutation.isPending ? "Applying..." : "Approve"}
                    </button>
                    <button
                      className="workspace-button workspace-button--secondary"
                      disabled={moderationDecisionMutation.isPending || moderationNotes.trim() === ""}
                      onClick={() => moderationDecisionMutation.mutate("request_changes")}
                      type="button"
                    >
                      Request changes
                    </button>
                    <button
                      className="workspace-button workspace-button--danger"
                      disabled={moderationDecisionMutation.isPending || moderationNotes.trim() === ""}
                      onClick={() => moderationDecisionMutation.mutate("reject")}
                      type="button"
                    >
                      Reject draft
                    </button>
                  </div>
                </section>
              </>
            ) : null}
          </section>
        </section>
      )}
    </main>
  );
}

function ModerationSection({ content, heading }: { content: string; heading: string }) {
  return (
    <section className="moderation-section">
      <h3 className="problem-section__heading">{heading}</h3>
      <ProblemMarkdown content={content} />
    </section>
  );
}

function LocalModerationSetupPanel({ localTestProfile }: { localTestProfile: LocalTestAuthProfile | null }) {
  const activeSubject = formatLocalTestSubject(localTestProfile);
  const sessionLabel = localTestProfile ? localTestAuthProfileLabels[localTestProfile] : "Unknown";

  return (
    <section className="local-moderation-help" aria-label="Local moderation setup guidance">
      <div className="local-moderation-help__callout">
        <p className="local-moderation-help__eyebrow">Local dev path</p>
        <p className="local-moderation-help__summary">
          {localTestProfile === "moderator"
            ? "The browser is already using the checked-in moderator token. The missing piece is the moderator role in PostgreSQL."
            : "The browser is not using the local moderator identity yet, or it is still on the author session. Switch to the moderator session in the header, then grant the moderator role in PostgreSQL."}
        </p>
      </div>

      <div className="local-moderation-help__meta-grid">
        <div className="local-moderation-help__meta-card">
          <p className="local-moderation-help__meta-label">auth_mode</p>
          <p className="local-moderation-help__meta-value">local_test</p>
        </div>
        <div className="local-moderation-help__meta-card">
          <p className="local-moderation-help__meta-label">session_profile</p>
          <p className="local-moderation-help__meta-value">{sessionLabel}</p>
        </div>
        <div className="local-moderation-help__meta-card">
          <p className="local-moderation-help__meta-label">active_subject</p>
          <p className="local-moderation-help__meta-value">{activeSubject}</p>
        </div>
        <div className="local-moderation-help__meta-card">
          <p className="local-moderation-help__meta-label">required_role</p>
          <p className="local-moderation-help__meta-value">moderator</p>
        </div>
      </div>

      <div className="local-moderation-help__steps">
        <section className="local-moderation-help__step">
          <h2 className="local-moderation-help__step-title">1. Use the checked-in moderator browser session</h2>
          <p className="local-moderation-help__step-text">
            The header controls expose two local test identities: <code>user_e2e_author</code> and <code>user_e2e_moderator</code>. Moderation only succeeds after switching the browser session to <code>Moderator</code>.
          </p>
        </section>

        <section className="local-moderation-help__step">
          <h2 className="local-moderation-help__step-title">2. Bootstrap the first admin once</h2>
          <p className="local-moderation-help__step-text">
            Run this from <code>services/api</code> against your local database if no admin exists yet.
          </p>
          <pre className="local-moderation-help__command">
            <code>{localAdminBootstrapCommand}</code>
          </pre>
        </section>

        <section className="local-moderation-help__step">
          <h2 className="local-moderation-help__step-title">3. Grant the moderator role to the browser moderator subject</h2>
          <p className="local-moderation-help__step-text">
            This aligns the checked-in moderator browser token with the staff role required by the moderation endpoints.
          </p>
          <pre className="local-moderation-help__command">
            <code>{localModeratorGrantCommand}</code>
          </pre>
        </section>
      </div>
    </section>
  );
}

function DraftStatusLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="draft-status-line">
      <span className="draft-status-line__label">{label}</span>
      <span className="draft-status-line__value">{value}</span>
    </div>
  );
}

function formatDraftLifecycle(draft: ProblemDraft) {
  switch (draft.lifecycleStatus) {
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

function formatSubmittedAt(item: ModerationQueueItem) {
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(item.submittedForReviewAt));
}

function formatDecisionSuccess(status: ProblemDraft["lifecycleStatus"]) {
  switch (status) {
    case "published":
      return "Draft approved and published.";
    case "archived":
      return "Draft rejected and archived.";
    case "draft":
      return "Draft returned to the author for changes.";
    case "in_review":
      return "Draft remains in review.";
  }
}

const localAdminBootstrapCommand = `DATABASE_URL="postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable" \\
  go run ./cmd/grant-staff-role --bootstrap-first-admin --target-subject user_e2e_author --role admin`;

const localModeratorGrantCommand = `DATABASE_URL="postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable" \\
  go run ./cmd/grant-staff-role --requester-subject user_e2e_author --target-subject user_e2e_moderator --role moderator`;

function formatLocalTestSubject(profile: LocalTestAuthProfile | null) {
  switch (profile) {
    case "author":
      return "user_e2e_author";
    case "moderator":
      return "user_e2e_moderator";
    default:
      return "unknown";
  }
}
