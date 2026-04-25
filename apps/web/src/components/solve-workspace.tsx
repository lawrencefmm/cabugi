"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import Editor from "@monaco-editor/react";
import { useMutation } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { ApiError, createSubmission, formatSubmissionLanguage } from "../lib/api";
import { starterCodeTemplates, type SubmissionLanguage } from "../lib/starter-code";
import { SignInButton, useAuth } from "./auth";

declare global {
  interface Window {
    __CABUGI_E2E__?: {
      solveWorkspace?: {
        setLanguage: (language: SubmissionLanguage) => void;
        setSourceCode: (sourceCode: string) => void;
      };
    };
  }
}

type SolveWorkspaceProps = {
  authEnabled: boolean;
  problemSlug: string;
};

type StoredSolveWorkspaceState = {
  language: SubmissionLanguage;
  latestSubmissionId: string | null;
  sources: Record<SubmissionLanguage, string>;
};

const solveWorkspaceStorageKeyPrefix = "cabugi_solve_workspace:";

export function SolveWorkspace({ authEnabled, problemSlug }: SolveWorkspaceProps) {
  if (!authEnabled) {
    return (
      <section className="workspace-panel">
        <div className="workspace-panel__header">
          <div>
            <h2 className="workspace-panel__title">Solve Workspace</h2>
            <p className="workspace-panel__subtitle">Frontend authentication is not configured in this environment yet.</p>
          </div>
        </div>
      </section>
    );
  }

  return <AuthenticatedSolveWorkspace problemSlug={problemSlug} />;
}

function AuthenticatedSolveWorkspace({ problemSlug }: { problemSlug: string }) {
  const auth = useAuth();
  const router = useRouter();
  const { getToken, isLoaded, isSignedIn } = auth;
  const [language, setLanguage] = useState<SubmissionLanguage>("cpp17");
  const [latestSubmissionId, setLatestSubmissionId] = useState<string | null>(null);
  const [loadedProblemSlug, setLoadedProblemSlug] = useState<string | null>(null);
  const [sourceByLanguage, setSourceByLanguage] = useState<Record<SubmissionLanguage, string>>(createDefaultSources);

  useEffect(() => {
    const storedState = readSolveWorkspaceState(problemSlug);
    setLanguage(storedState.language);
    setLatestSubmissionId(storedState.latestSubmissionId);
    setSourceByLanguage(storedState.sources);
    setLoadedProblemSlug(problemSlug);
  }, [problemSlug]);

  useEffect(() => {
    if (loadedProblemSlug !== problemSlug) {
      return;
    }

    writeSolveWorkspaceState(problemSlug, {
      language,
      latestSubmissionId,
      sources: sourceByLanguage,
    });
  }, [language, latestSubmissionId, loadedProblemSlug, problemSlug, sourceByLanguage]);

  useEffect(() => {
    if (auth.mode !== "local_test") {
      return;
    }

    if (!window.__CABUGI_E2E__) {
      window.__CABUGI_E2E__ = {};
    }
    window.__CABUGI_E2E__.solveWorkspace = {
      setLanguage,
      setSourceCode: (nextSourceCode) => {
        setSourceByLanguage((current) => ({
          ...current,
          [language]: nextSourceCode,
        }));
      },
    };

    return () => {
      if (window.__CABUGI_E2E__) {
        delete window.__CABUGI_E2E__.solveWorkspace;
      }
    };
  }, [auth.mode, language]);

  const sourceCode = sourceByLanguage[language];

  const createSubmissionMutation = useMutation({
    mutationFn: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error("Missing submission token");
      }

      return createSubmission(
        {
          problemSlug,
          language,
          sourceCode,
        },
        token,
      );
    },
    onSuccess: (submission) => {
      const nextLatestSubmissionId = submission.id;
      setLatestSubmissionId(nextLatestSubmissionId);
      writeSolveWorkspaceState(problemSlug, {
        language,
        latestSubmissionId: nextLatestSubmissionId,
        sources: sourceByLanguage,
      });
      router.push(`/submissions/${submission.id}`);
    },
  });

  if (!isLoaded) {
    return (
      <section className="workspace-panel" role="status" aria-live="polite">
        <div className="workspace-panel__header">
          <div>
            <h2 className="workspace-panel__title">Solve Workspace</h2>
            <p className="workspace-panel__subtitle">Loading authentication state.</p>
          </div>
        </div>
      </section>
    );
  }

  if (!isSignedIn) {
    return (
      <section className="workspace-panel">
        <div className="workspace-panel__header">
          <div>
            <h2 className="workspace-panel__title">Solve Workspace</h2>
            <p className="workspace-panel__subtitle">Sign in to submit solutions and follow verdict changes on the live submission page.</p>
          </div>
        </div>

        <div className="workspace-auth-gate">
          <p className="workspace-auth-gate__text">Browsing stays open, but submissions require an authenticated Clerk session.</p>
          <SignInButton mode="modal">
            <button className="workspace-button" type="button">
              Sign in to solve
            </button>
          </SignInButton>
        </div>
      </section>
    );
  }

  return (
    <section className="workspace-panel">
      <div className="workspace-panel__header">
        <div>
          <h2 className="workspace-panel__title">Solve Workspace</h2>
          <p className="workspace-panel__subtitle">Keep separate buffers per language, submit once, and jump straight to the live submission stream.</p>
        </div>

        <label className="workspace-language-picker">
          <span>Language</span>
          <select value={language} onChange={(event) => setLanguage(event.target.value as SubmissionLanguage)}>
            <option value="cpp17">C++17</option>
            <option value="python">Python</option>
          </select>
        </label>
      </div>

      <div className="meta-strip" aria-label="Solve workspace status">
        <span className="meta-chip mono">problem {problemSlug}</span>
        <span className="meta-chip mono">buffer {formatSubmissionLanguage(language)}</span>
        <span className="meta-chip mono">per-language state preserved</span>
        {latestSubmissionId ? <span className="meta-chip mono">latest {latestSubmissionId}</span> : null}
      </div>

      <p className="detail-meta">Switching languages keeps each buffer intact on this browser. Successful submissions redirect to `/submissions/[id]` automatically.</p>

      <div className="history-actions">
        <Link className="table-link" href="/submissions">
          Open submission history
        </Link>
        {latestSubmissionId ? (
          <Link className="table-link" href={`/submissions/${latestSubmissionId}`}>
            Reopen latest submission
          </Link>
        ) : null}
      </div>

      <div className="workspace-editor-shell">
        <Editor
          defaultLanguage={language === "cpp17" ? "cpp" : "python"}
          height="360px"
          language={language === "cpp17" ? "cpp" : "python"}
          theme="vs-dark"
          value={sourceCode}
          onChange={(value) => {
            setSourceByLanguage((current) => ({
              ...current,
              [language]: value ?? "",
            }));
          }}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            scrollBeyondLastLine: false,
            wordWrap: "on",
          }}
        />
      </div>

      <div className="workspace-actions">
        <button
          className="workspace-button"
          type="button"
          onClick={() => createSubmissionMutation.mutate()}
          disabled={createSubmissionMutation.isPending || sourceCode.trim() === ""}
        >
          {createSubmissionMutation.isPending ? "Submitting..." : "Submit solution"}
        </button>
      </div>

      {createSubmissionMutation.error ? (
        <p className="workspace-error" role="alert">
          {formatSubmissionCreateError(createSubmissionMutation.error)}
        </p>
      ) : null}
    </section>
  );
}

function createDefaultSources(): Record<SubmissionLanguage, string> {
  return {
    cpp17: starterCodeTemplates.cpp17,
    python: starterCodeTemplates.python,
  };
}

function readSolveWorkspaceState(problemSlug: string): StoredSolveWorkspaceState {
  const defaults: StoredSolveWorkspaceState = {
    language: "cpp17",
    latestSubmissionId: null,
    sources: createDefaultSources(),
  };

  const storage = browserStorage();
  if (!storage) {
    return defaults;
  }

  try {
    const rawValue = storage.getItem(storageKey(problemSlug));
    if (!rawValue) {
      return defaults;
    }

    const parsed = JSON.parse(rawValue) as {
      language?: unknown;
      latestSubmissionId?: unknown;
      sources?: { cpp17?: unknown; python?: unknown };
    };

    return {
      language: isSubmissionLanguage(parsed.language) ? parsed.language : defaults.language,
      latestSubmissionId: typeof parsed.latestSubmissionId === "string" && parsed.latestSubmissionId.trim() !== "" ? parsed.latestSubmissionId : null,
      sources: {
        cpp17: typeof parsed.sources?.cpp17 === "string" ? parsed.sources.cpp17 : defaults.sources.cpp17,
        python: typeof parsed.sources?.python === "string" ? parsed.sources.python : defaults.sources.python,
      },
    };
  } catch {
    return defaults;
  }
}

function writeSolveWorkspaceState(problemSlug: string, state: StoredSolveWorkspaceState) {
  const storage = browserStorage();
  if (!storage) {
    return;
  }

  storage.setItem(storageKey(problemSlug), JSON.stringify(state));
}

function storageKey(problemSlug: string) {
  return `${solveWorkspaceStorageKeyPrefix}${problemSlug}`;
}

function isSubmissionLanguage(value: unknown): value is SubmissionLanguage {
  return value === "cpp17" || value === "python";
}

function browserStorage() {
  if (typeof window === "undefined") {
    return null;
  }

  const storage = window.localStorage;
  if (!storage || typeof storage.getItem !== "function" || typeof storage.setItem !== "function") {
    return null;
  }

  return storage;
}

function formatSubmissionCreateError(error: unknown) {
  if (error instanceof ApiError) {
    if (error.status === 400 && error.code === "invalid_language") {
      return "Only C++17 and Python are supported in this workspace right now.";
    }
    if (error.status === 400) {
      return "Check the selected language and source buffer, then submit again.";
    }
    if (error.status === 401) {
      return "Your session token is missing or expired. Sign in again and retry the submission.";
    }
    if (error.status === 404) {
      return "This published problem is no longer available for new submissions.";
    }
    if (error.status === 503) {
      return "The submission pipeline is unavailable right now. Try again in a moment.";
    }
  }

  return "Unable to create a submission right now. Check your connection and try again.";
}
