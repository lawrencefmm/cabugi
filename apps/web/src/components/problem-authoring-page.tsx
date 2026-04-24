"use client";

import { SignInButton, useAuth } from "@clerk/nextjs";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { type FormEvent, type ReactNode, useEffect, useState } from "react";

import {
  createProblemDraft,
  fetchProblemDraft,
  formatDraftError,
  formatMemoryLimit,
  formatTimeLimit,
  submitProblemDraftForReview,
  type ProblemDraft,
  type ProblemDraftCreateInput,
  uploadProblemDraftHiddenTestBundle,
  updateProblemDraft,
} from "../lib/api";

type ProblemAuthoringPageProps = {
  authEnabled: boolean;
  slug?: string;
};

type DraftFormState = ProblemDraftCreateInput;

const emptyDraftForm: DraftFormState = {
  slug: "",
  title: "",
  statementMarkdown: "",
  inputMarkdown: "",
  outputMarkdown: "",
  constraintsMarkdown: "",
  notesMarkdown: "",
  timeLimitMs: 1000,
  memoryLimitMb: 256,
  hiddenTestBundleKey: "",
  hiddenTestBundleSha256: "",
};

export function ProblemAuthoringPage({ authEnabled, slug }: ProblemAuthoringPageProps) {
  if (!authEnabled) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Problem authoring unavailable</h1>
          <p className="empty-state__text">Frontend authentication is not configured in this environment yet.</p>
        </section>
      </main>
    );
  }

  return <AuthenticatedProblemAuthoringPage slug={slug} />;
}

function AuthenticatedProblemAuthoringPage({ slug }: { slug?: string }) {
  const { getToken, isLoaded, isSignedIn } = useAuth();
  const router = useRouter();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<DraftFormState>(emptyDraftForm);
  const [bundleFile, setBundleFile] = useState<File | null>(null);
  const [draftMessage, setDraftMessage] = useState<string | null>(null);
  const isEditing = Boolean(slug);

  const draftQuery = useQuery({
    enabled: isLoaded && isSignedIn && isEditing,
    queryKey: ["problem-draft", slug],
    queryFn: async () => {
      const token = await getToken();
      if (!token || !slug) {
        throw new Error("Missing draft token");
      }

      return fetchProblemDraft(slug, token);
    },
  });

  useEffect(() => {
    if (!draftQuery.data) {
      return;
    }

    setForm(draftToFormState(draftQuery.data));
  }, [draftQuery.data]);

  const createDraftMutation = useMutation({
    mutationFn: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error("Missing draft token");
      }

      return createProblemDraft(normalizeDraftForm(form), token);
    },
    onSuccess: (draft) => {
      queryClient.setQueryData(["problem-draft", draft.slug], draft);
      router.push(`/drafts/${draft.slug}`);
    },
  });

  const updateDraftMutation = useMutation({
    mutationFn: async () => {
      const token = await getToken();
      if (!token || !slug) {
        throw new Error("Missing draft token");
      }

      const { slug: _slug, ...input } = normalizeDraftForm(form);
      return updateProblemDraft(slug, input, token);
    },
    onSuccess: (draft) => {
      queryClient.setQueryData(["problem-draft", draft.slug], draft);
      setDraftMessage("Draft changes saved.");
    },
  });

  const uploadBundleMutation = useMutation({
    mutationFn: async () => {
      const token = await getToken();
      if (!token || !bundleFile) {
        throw new Error("Missing hidden test bundle upload input");
      }

      return uploadProblemDraftHiddenTestBundle(bundleFile, token);
    },
    onSuccess: (uploadedBundle) => {
      setForm((current) => ({
        ...current,
        hiddenTestBundleKey: uploadedBundle.hiddenTestBundleKey,
        hiddenTestBundleSha256: uploadedBundle.hiddenTestBundleSha256,
      }));
      setBundleFile(null);
      setDraftMessage("Hidden test bundle uploaded.");
    },
  });

  const submitForReviewMutation = useMutation({
    mutationFn: async () => {
      const token = await getToken();
      if (!token || !slug) {
        throw new Error("Missing draft token");
      }

      return submitProblemDraftForReview(slug, token);
    },
    onSuccess: (draft) => {
      queryClient.setQueryData(["problem-draft", draft.slug], draft);
      setDraftMessage("Draft submitted for review.");
    },
  });

  const currentDraft = draftQuery.data;
  const canEditDraft = !isEditing || currentDraft?.lifecycleStatus === "draft";
  const actionError =
    uploadBundleMutation.error ?? createDraftMutation.error ?? updateDraftMutation.error ?? submitForReviewMutation.error ?? draftQuery.error ?? null;

  if (!isLoaded) {
    return (
      <main className="app-main">
        <section className="empty-state" role="status" aria-live="polite">
          <h1 className="empty-state__title">Loading authoring tools</h1>
          <p className="empty-state__text">Checking authentication and preparing the draft workspace.</p>
        </section>
      </main>
    );
  }

  if (!isSignedIn) {
    return (
      <main className="app-main">
        <section className="empty-state">
          <h1 className="empty-state__title">Sign in to author problems</h1>
          <p className="empty-state__text">Draft creation and editing are tied to your Cabugi account.</p>
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

  if (isEditing && draftQuery.isPending) {
    return (
      <main className="app-main">
        <section className="empty-state" role="status" aria-live="polite">
          <h1 className="empty-state__title">Loading draft</h1>
          <p className="empty-state__text">Fetching the latest saved problem draft.</p>
        </section>
      </main>
    );
  }

  if (isEditing && draftQuery.error) {
    return (
      <main className="app-main">
        <section className="error-state" role="alert">
          <h1 className="error-state__title">Draft unavailable</h1>
          <p className="error-state__text">{formatDraftError(draftQuery.error, "Problem draft not found.")}</p>
        </section>
      </main>
    );
  }

  return (
    <main className="app-main">
      <section className="page-hero">
        <a className="status-note" href={isEditing ? "/drafts/new" : "/"}>
          {isEditing ? "Create another draft" : "Back to problems"}
        </a>
        <span className="page-kicker">Problem Authoring</span>
        <h1 className="page-title">{isEditing ? form.title || slug : "Start a new problem draft."}</h1>
        <p className="page-subtitle">
          Draft your statement, set the limits, upload the hidden test bundle, and hand the problem off to moderation when it is ready.
        </p>
      </section>

      <section className="draft-layout">
        <section className="workspace-panel">
          <div className="workspace-panel__header">
            <div>
              <h2 className="workspace-panel__title">Draft Editor</h2>
              <p className="workspace-panel__subtitle">
                {canEditDraft
                  ? "Save incremental draft updates as you refine the statement and upload the hidden test bundle used for judging."
                  : "This draft is no longer editable because it has already entered moderation."}
              </p>
            </div>

            {currentDraft ? <DraftStatus status={currentDraft.lifecycleStatus} /> : null}
          </div>

          <form className="draft-form" onSubmit={(event) => handleDraftSubmit(event)}>
            <div className="draft-grid">
              <DraftField label="Slug">
                <input
                  disabled={isEditing}
                  name="slug"
                  required
                  type="text"
                  value={form.slug}
                  onChange={(event) => updateField("slug", event.target.value)}
                />
              </DraftField>

              <DraftField label="Title">
                <input name="title" required type="text" value={form.title} onChange={(event) => updateField("title", event.target.value)} />
              </DraftField>
            </div>

            <div className="draft-grid">
              <DraftField label="Time limit (ms)">
                <input
                  min={1}
                  name="timeLimitMs"
                  required
                  type="number"
                  value={form.timeLimitMs}
                  onChange={(event) => updateNumberField("timeLimitMs", event.target.value)}
                />
              </DraftField>

              <DraftField label="Memory limit (MB)">
                <input
                  min={1}
                  name="memoryLimitMb"
                  required
                  type="number"
                  value={form.memoryLimitMb}
                  onChange={(event) => updateNumberField("memoryLimitMb", event.target.value)}
                />
              </DraftField>
            </div>

            <DraftField label="Statement Markdown">
              <textarea value={form.statementMarkdown} onChange={(event) => updateField("statementMarkdown", event.target.value)} />
            </DraftField>

            <div className="draft-grid">
              <DraftField label="Input Markdown">
                <textarea value={form.inputMarkdown} onChange={(event) => updateField("inputMarkdown", event.target.value)} />
              </DraftField>

              <DraftField label="Output Markdown">
                <textarea value={form.outputMarkdown} onChange={(event) => updateField("outputMarkdown", event.target.value)} />
              </DraftField>
            </div>

            <div className="draft-grid">
              <DraftField label="Constraints Markdown">
                <textarea value={form.constraintsMarkdown} onChange={(event) => updateField("constraintsMarkdown", event.target.value)} />
              </DraftField>

              <DraftField label="Notes Markdown">
                <textarea value={form.notesMarkdown} onChange={(event) => updateField("notesMarkdown", event.target.value)} />
              </DraftField>
            </div>

            <div className="draft-grid">
              <DraftField label="Hidden test bundle file">
                <input
                  accept=".json,application/json"
                  disabled={!canEditDraft || uploadBundleMutation.isPending}
                  type="file"
                  onChange={(event) => {
                    setDraftMessage(null);
                    setBundleFile(event.target.files?.[0] ?? null);
                  }}
                />
              </DraftField>

              <DraftField label="Bundle upload action">
                <div className="draft-actions">
                  <button
                    className="workspace-button workspace-button--secondary"
                    disabled={!canEditDraft || !bundleFile || uploadBundleMutation.isPending}
                    onClick={() => uploadBundleMutation.mutate()}
                    type="button"
                  >
                    {uploadBundleMutation.isPending ? "Uploading..." : "Upload bundle"}
                  </button>
                  <span className="workspace-auth-gate__text mono">{bundleFile?.name ?? (form.hiddenTestBundleKey ? "Bundle uploaded" : "No file selected")}</span>
                </div>
              </DraftField>
            </div>

            <div className="draft-grid">
              <DraftField label="Hidden test bundle key">
                <input
                  readOnly
                  type="text"
                  value={form.hiddenTestBundleKey ?? ""}
                />
              </DraftField>

              <DraftField label="Hidden test bundle SHA-256">
                <input
                  readOnly
                  type="text"
                  value={form.hiddenTestBundleSha256 ?? ""}
                />
              </DraftField>
            </div>

            {actionError ? <p className="workspace-error" role="alert">{formatDraftError(actionError, "Problem draft not found.")}</p> : null}
            {draftMessage ? <p className="draft-success">{draftMessage}</p> : null}

            <div className="draft-actions">
              <button
                className="workspace-button"
                disabled={!canEditDraft || uploadBundleMutation.isPending || createDraftMutation.isPending || updateDraftMutation.isPending}
                type="submit"
              >
                {isEditing ? (updateDraftMutation.isPending ? "Saving..." : "Save changes") : createDraftMutation.isPending ? "Creating..." : "Create draft"}
              </button>

              {isEditing ? (
                <button
                  className="workspace-button workspace-button--secondary"
                  disabled={!canEditDraft || uploadBundleMutation.isPending || submitForReviewMutation.isPending}
                  onClick={() => void handleSubmitForReview()}
                  type="button"
                >
                  {submitForReviewMutation.isPending ? "Submitting..." : "Submit for review"}
                </button>
              ) : null}
            </div>
          </form>
        </section>

        <aside className="problem-sidebar draft-sidebar">
          <div className="problem-sidebar__section">
            <h2 className="problem-sidebar__heading">Draft Metadata</h2>
            <div className="problem-sidebar__list">
              <div>
                <span className="problem-sidebar__meta-label">Slug</span>
                <span className="problem-sidebar__meta-value">{form.slug || "Will be set on create"}</span>
              </div>
              <div>
                <span className="problem-sidebar__meta-label">Time limit</span>
                <span className="problem-sidebar__meta-value">{formatTimeLimit(form.timeLimitMs)}</span>
              </div>
              <div>
                <span className="problem-sidebar__meta-label">Memory limit</span>
                <span className="problem-sidebar__meta-value">{formatMemoryLimit(form.memoryLimitMb)}</span>
              </div>
              {currentDraft ? (
                <div>
                  <span className="problem-sidebar__meta-label">Version</span>
                  <span className="problem-sidebar__meta-value">{currentDraft.versionNumber}</span>
                </div>
              ) : null}
            </div>
          </div>

          <div className="problem-sidebar__section">
            <h2 className="problem-sidebar__heading">Review Checklist</h2>
            <div className="draft-checklist">
              <p className="workspace-auth-gate__text">Fill the statement fields, confirm the limits, and upload a validated hidden bundle before submitting for review.</p>
            </div>
          </div>
        </aside>
      </section>
    </main>
  );

  function updateField<Key extends keyof DraftFormState>(key: Key, value: DraftFormState[Key]) {
    setDraftMessage(null);
    setForm((current) => ({ ...current, [key]: value }));
  }

  function updateNumberField(key: "timeLimitMs" | "memoryLimitMb", value: string) {
    const nextValue = Number(value);
    setDraftMessage(null);
    setForm((current) => ({ ...current, [key]: Number.isFinite(nextValue) ? nextValue : 0 }));
  }

  function handleDraftSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setDraftMessage(null);

    if (!isEditing) {
      createDraftMutation.mutate();
      return;
    }

    updateDraftMutation.mutate();
  }

  function handleSubmitForReview() {
    setDraftMessage(null);
    submitForReviewMutation.mutate();
  }
}

function DraftField({ children, label }: { children: ReactNode; label: string }) {
  return (
    <label className="draft-field">
      <span className="draft-field__label">{label}</span>
      {children}
    </label>
  );
}

function DraftStatus({ status }: { status: ProblemDraft["lifecycleStatus"] }) {
  return <span className={`draft-status draft-status--${status}`}>{formatDraftStatus(status)}</span>;
}

function draftToFormState(draft: ProblemDraft): DraftFormState {
  return {
    slug: draft.slug,
    title: draft.title,
    statementMarkdown: draft.statementMarkdown,
    inputMarkdown: draft.inputMarkdown,
    outputMarkdown: draft.outputMarkdown,
    constraintsMarkdown: draft.constraintsMarkdown,
    notesMarkdown: draft.notesMarkdown,
    timeLimitMs: draft.timeLimitMs,
    memoryLimitMb: draft.memoryLimitMb,
    hiddenTestBundleKey: draft.hiddenTestBundleKey,
    hiddenTestBundleSha256: draft.hiddenTestBundleSha256,
  };
}

function normalizeDraftForm(form: DraftFormState): DraftFormState {
  return {
    ...form,
    slug: form.slug.trim(),
    title: form.title.trim(),
    hiddenTestBundleKey: form.hiddenTestBundleKey?.trim() ?? "",
    hiddenTestBundleSha256: form.hiddenTestBundleSha256?.trim() ?? "",
  };
}

function formatDraftStatus(status: ProblemDraft["lifecycleStatus"]) {
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
