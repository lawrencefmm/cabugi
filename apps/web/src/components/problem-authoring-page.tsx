"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, type ReactNode, useDeferredValue, useEffect, useState } from "react";

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
import { SignInButton, useAuth } from "./auth";
import { ProblemMarkdown } from "./problem-markdown";

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
  const deferredPreview = useDeferredValue(form);

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
  const readinessChecks = buildReadinessChecks(form);
  const readyCheckCount = readinessChecks.filter((check) => check.ready).length;
  const readyForReview = readinessChecks.every((check) => check.ready);
  const canSubmitForReview = canEditDraft && readyForReview && !uploadBundleMutation.isPending && !submitForReviewMutation.isPending;
  const actionError =
    uploadBundleMutation.error ?? createDraftMutation.error ?? updateDraftMutation.error ?? submitForReviewMutation.error ?? draftQuery.error ?? null;

  if (!isLoaded) {
    return (
      <main className="app-main">
        <section className="empty-state" role="status" aria-live="polite">
          <h1 className="empty-state__title">Checking authoring access</h1>
          <p className="empty-state__text">Published problems stay available while your account finishes loading.</p>
          <div className="history-actions">
            <Link className="table-link" href="/">
              Browse problems
            </Link>
          </div>
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
          <div className="history-actions">
            <Link className="table-link" href="/drafts">
              Back to drafts
            </Link>
          </div>
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
          <div className="history-actions">
            <Link className="table-link" href="/drafts">
              Back to drafts
            </Link>
            <Link className="table-link" href="/drafts/new">
              Start a new draft
            </Link>
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className="app-main">
      <section className="page-hero">
        <Link className="status-note" href={isEditing ? "/drafts/new" : "/drafts"}>
          {isEditing ? "Create another draft" : "Back to drafts"}
        </Link>
        <span className="page-kicker">Problem Authoring</span>
        <h1 className="page-title">{isEditing ? form.title || slug : "Start a new problem draft."}</h1>
        <p className="page-subtitle">
          Draft your statement, set the limits, upload the hidden test bundle, and hand the problem off to moderation when it is ready.
        </p>
      </section>

      <section className="draft-layout draft-layout--redesign">
        <DraftSectionRail checks={readinessChecks} />

        <section className="authoring-main-column">
          <section className="workspace-panel authoring-editor-panel">
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
                  disabled={!canSubmitForReview}
                  onClick={() => void handleSubmitForReview()}
                  type="button"
                >
                  {submitForReviewMutation.isPending ? "Submitting..." : "Submit for review"}
                </button>
              ) : null}
            </div>
            </form>
          </section>

          <section aria-label="Draft preview" className="workspace-panel draft-preview-panel">
            <div className="workspace-panel__header">
              <div>
                <h2 className="workspace-panel__title">Live Preview</h2>
                <p className="workspace-panel__subtitle">This preview updates as you edit so you can review the exact statement structure before sending the draft to moderators.</p>
              </div>
            </div>

            <div className="moderation-sections moderation-sections--preview">
              <PreviewSection content={deferredPreview.statementMarkdown} emptyMessage="Start writing the statement to preview it here." heading="Statement" />
              <PreviewSection content={deferredPreview.inputMarkdown} emptyMessage="Describe the input format to preview it here." heading="Input" />
              <PreviewSection content={deferredPreview.outputMarkdown} emptyMessage="Describe the expected output to preview it here." heading="Output" />
              <PreviewSection content={deferredPreview.constraintsMarkdown} emptyMessage="Add constraints or complexity notes to preview them here." heading="Constraints" />
              <PreviewSection content={deferredPreview.notesMarkdown} emptyMessage="Optional implementation notes or clarifications will appear here." heading="Notes" />
            </div>
          </section>
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
            <h2 className="problem-sidebar__heading">Review Readiness</h2>
            <div className="draft-checklist">
              <p className="workspace-auth-gate__text">
                {isEditing
                  ? readyForReview
                    ? "Ready to submit for review. Keep saving as needed, then send this draft to moderation when you are satisfied with the preview below."
                    : `${readyCheckCount} of ${readinessChecks.length} checks complete. Save anytime, then finish the missing items before submitting for review.`
                  : `${readyCheckCount} of ${readinessChecks.length} checks complete. Create the draft first, then come back to submit it for review once the checklist is green.`}
              </p>

              <div className="draft-status-line">
                <span className="draft-status-line__label">Hidden bundle</span>
                <span className="draft-status-line__value">{form.hiddenTestBundleKey ? "Uploaded and linked" : bundleFile ? `Selected: ${bundleFile.name}` : "Missing"}</span>
              </div>

              <ul className="draft-readiness-list" aria-label="Draft readiness checks">
                {readinessChecks.map((check) => (
                  <li className="draft-readiness-item" key={check.label}>
                    <div>
                      <p className="draft-readiness-item__label">{check.label}</p>
                      <p className="draft-readiness-item__hint">{check.hint}</p>
                    </div>
                    <span className={`draft-readiness-badge${check.ready ? " draft-readiness-badge--ready" : ""}`}>{check.ready ? "Ready" : "Missing"}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>

          <div className="problem-sidebar__section">
            <h2 className="problem-sidebar__heading">Hidden Bundle State</h2>
            <div className="draft-checklist">
              <div className="draft-status-line">
                <span className="draft-status-line__label">Bundle key</span>
                <span className="draft-status-line__value">{form.hiddenTestBundleKey || "Not uploaded yet"}</span>
              </div>
              <div className="draft-status-line">
                <span className="draft-status-line__label">Bundle SHA-256</span>
                <span className="draft-status-line__value">{form.hiddenTestBundleSha256 || "Waiting for validated upload"}</span>
              </div>
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

function DraftSectionRail({ checks }: { checks: ReturnType<typeof buildReadinessChecks> }) {
  const sections = ["Title", "Statement", "Input Format", "Output Format", "Examples", "Constraints", "Tags", "Limits & Languages", "Hidden Tests", "Checklist"];

  return (
    <aside className="draft-sections" aria-label="Draft sections">
      <p className="sidebar-label">Sections</p>
      <ol className="draft-sections__list">
        {sections.map((section, index) => {
          const ready = checks[index % checks.length]?.ready ?? false;

          return (
            <li className={ready ? "draft-sections__item draft-sections__item--ready" : "draft-sections__item"} key={section}>
              <span>{index + 1}. {section}</span>
              <span aria-label={ready ? "Ready" : "Incomplete"} />
            </li>
          );
        })}
      </ol>
      <button className="workspace-button workspace-button--secondary" type="button">
        Validate All
      </button>
    </aside>
  );
}

function DraftStatus({ status }: { status: ProblemDraft["lifecycleStatus"] }) {
  return <span className={`draft-status draft-status--${status}`}>{formatDraftStatus(status)}</span>;
}

function PreviewSection({ content, emptyMessage, heading }: { content: string; emptyMessage: string; heading: string }) {
  return (
    <section className="moderation-section">
      <h3 className="problem-section__heading">{heading}</h3>
      {content.trim() ? <ProblemMarkdown content={content} /> : <p className="workspace-auth-gate__text">{emptyMessage}</p>}
    </section>
  );
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

function buildReadinessChecks(form: DraftFormState) {
  return [
    {
      label: "Slug",
      hint: "Set a stable URL-friendly slug for the draft.",
      ready: form.slug.trim() !== "",
    },
    {
      label: "Title",
      hint: "Use the public-facing problem title shown in the problemset.",
      ready: form.title.trim() !== "",
    },
    {
      label: "Statement",
      hint: "Explain the task clearly enough to preview the main statement.",
      ready: form.statementMarkdown.trim() !== "",
    },
    {
      label: "Input and output",
      hint: "Describe both sides of the contract before review.",
      ready: form.inputMarkdown.trim() !== "" && form.outputMarkdown.trim() !== "",
    },
    {
      label: "Constraints",
      hint: "Surface the key numeric or complexity limits moderators should validate.",
      ready: form.constraintsMarkdown.trim() !== "",
    },
    {
      label: "Execution limits",
      hint: "Time and memory limits must both stay positive.",
      ready: form.timeLimitMs > 0 && form.memoryLimitMb > 0,
    },
    {
      label: "Hidden test bundle uploaded",
      hint: "A validated hidden bundle is required before moderation can start.",
      ready: (form.hiddenTestBundleKey ?? "").trim() !== "" && (form.hiddenTestBundleSha256 ?? "").trim() !== "",
    },
  ];
}
