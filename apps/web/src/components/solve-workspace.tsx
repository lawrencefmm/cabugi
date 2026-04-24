"use client";

import Editor from "@monaco-editor/react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { createSubmission, fetchSubmission, isTerminalSubmissionStatus, type Submission } from "../lib/api";
import { starterCodeTemplates, type SubmissionLanguage } from "../lib/starter-code";
import { SignInButton, useAuth } from "./auth";
import { VerdictBadge } from "./verdict-badge";

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
  pollIntervalMs?: number;
};

export function SolveWorkspace({ authEnabled, pollIntervalMs = 1500, problemSlug }: SolveWorkspaceProps) {
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

  return <AuthenticatedSolveWorkspace pollIntervalMs={pollIntervalMs} problemSlug={problemSlug} />;
}

function AuthenticatedSolveWorkspace({ pollIntervalMs, problemSlug }: { pollIntervalMs: number; problemSlug: string }) {
  const auth = useAuth();
  const { getToken, isLoaded, isSignedIn } = auth;
  const [language, setLanguage] = useState<SubmissionLanguage>("cpp17");
  const [sourceCode, setSourceCode] = useState(starterCodeTemplates.cpp17);
  const [activeSubmission, setActiveSubmission] = useState<Submission | null>(null);

  useEffect(() => {
    setSourceCode(starterCodeTemplates[language]);
  }, [language]);

  useEffect(() => {
    if (auth.mode !== "local_test") {
      return;
    }

    if (!window.__CABUGI_E2E__) {
      window.__CABUGI_E2E__ = {};
    }
    window.__CABUGI_E2E__.solveWorkspace = {
      setLanguage,
      setSourceCode,
    };

    return () => {
      if (window.__CABUGI_E2E__) {
        delete window.__CABUGI_E2E__.solveWorkspace;
      }
    };
  }, [auth.mode]);

  const submissionQuery = useQuery({
    enabled: isSignedIn && activeSubmission !== null,
    queryKey: ["submission", activeSubmission?.id],
    queryFn: async () => {
      const token = await getToken();
      if (!token || !activeSubmission) {
        throw new Error("Missing submission token");
      }

      return fetchSubmission(activeSubmission.id, token);
    },
    refetchInterval: (query) => {
      const submission = query.state.data ?? activeSubmission;
      if (!submission || isTerminalSubmissionStatus(submission.status)) {
        return false;
      }

      return pollIntervalMs;
    },
  });

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
      setActiveSubmission(submission);
    },
  });

  const liveSubmission = submissionQuery.data ?? activeSubmission;

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
            <p className="workspace-panel__subtitle">Sign in to submit solutions and follow verdict changes in real time.</p>
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
          <p className="workspace-panel__subtitle">Choose a language, edit source, and submit directly to the judge pipeline for this problem.</p>
        </div>

        <label className="workspace-language-picker">
          <span>Language</span>
          <select value={language} onChange={(event) => setLanguage(event.target.value as SubmissionLanguage)}>
            <option value="cpp17">C++17</option>
            <option value="python">Python</option>
          </select>
        </label>
      </div>

      <div className="workspace-editor-shell">
        <Editor
          defaultLanguage={language === "cpp17" ? "cpp" : "python"}
          height="360px"
          language={language === "cpp17" ? "cpp" : "python"}
          theme="vs-dark"
          value={sourceCode}
          onChange={(value) => setSourceCode(value ?? "")}
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            scrollBeyondLastLine: false,
            wordWrap: "on",
          }}
        />
      </div>

      <div className="workspace-actions">
        <button className="workspace-button" type="button" onClick={() => createSubmissionMutation.mutate()} disabled={createSubmissionMutation.isPending}>
          {createSubmissionMutation.isPending ? "Submitting..." : "Submit solution"}
        </button>

        {liveSubmission ? (
          <VerdictBadge verdict={liveSubmission.status} />
        ) : null}
      </div>

      {createSubmissionMutation.error ? (
        <p className="workspace-error" role="alert">
          Unable to create a submission right now.
        </p>
      ) : null}

      {submissionQuery.error ? (
        <p className="workspace-error" role="alert">
          Unable to refresh submission status right now.
        </p>
      ) : null}
    </section>
  );
}
