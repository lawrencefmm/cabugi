"use client";

import Link from "next/link";
import { startTransition, useEffect, useEffectEvent, useState } from "react";

import { fetchPublishedProblem, formatMemoryLimit, formatProblemFetchError, formatTimeLimit, type PublishedProblemDetail } from "../lib/api";
import { ProblemMarkdown } from "./problem-markdown";
import { SolveWorkspace } from "./solve-workspace";

type ProblemDetailState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; problem: PublishedProblemDetail };

type ProblemDetailPageProps = {
  authEnabled: boolean;
  slug: string;
};

export function ProblemDetailPage({ authEnabled, slug }: ProblemDetailPageProps) {
  const [state, setState] = useState<ProblemDetailState>({ kind: "loading" });
  const [reloadToken, setReloadToken] = useState(0);

  const loadProblem = useEffectEvent(async (signal: AbortSignal) => {
    try {
      const problem = await fetchPublishedProblem(slug, signal);
      if (signal.aborted) {
        return;
      }

      startTransition(() => {
        setState({ kind: "ready", problem });
      });
    } catch (error) {
      if (signal.aborted) {
        return;
      }

      startTransition(() => {
        setState({
          kind: "error",
          message: formatProblemFetchError(error, "Problem not found."),
        });
      });
    }
  });

  useEffect(() => {
    const controller = new AbortController();
    startTransition(() => {
      setState({ kind: "loading" });
    });

    void loadProblem(controller.signal);

    return () => {
      controller.abort();
    };
  }, [reloadToken, slug]);

  function retryProblem() {
    setReloadToken((current) => current + 1);
  }

  if (state.kind === "loading") {
    return (
      <main className="app-main">
        <section className="empty-state" role="status" aria-live="polite">
          <h1 className="empty-state__title">Loading problem</h1>
        </section>
      </main>
    );
  }

  if (state.kind === "error") {
    return (
      <main className="app-main">
        <section className="error-state" role="alert">
          <h1 className="error-state__title">Problem unavailable</h1>
          <p className="error-state__text">{state.message}</p>
          <div className="history-actions">
            <button className="workspace-button workspace-button--secondary" onClick={retryProblem} type="button">
              Retry problem
            </button>
            <Link className="table-link" href="/">
              Back to problems
            </Link>
          </div>
        </section>
      </main>
    );
  }

  const { problem } = state;

  return (
    <main className="app-main app-main--full solve-page">
      <section className="solve-shell">
        <article className="problem-panel problem-panel--statement">
          <div className="problem-panel__topbar">
            <Link className="status-note status-note--ghost" href="/">
              Back to problems
            </Link>
            <div className="problem-panel__icons" aria-hidden="true">
              <span />
              <span />
            </div>
          </div>

          <header className="problem-reader__header">
            <h1 className="page-title">{problem.title}</h1>
            <div className="meta-strip problem-reader__tags" aria-label="Problem metadata overview">
              <span className="difficulty-pill difficulty-pill--easy">Easy</span>
              <span className="tag-chip">Arrays</span>
              <span className="tag-chip">Hash Table</span>
              <span className="tag-chip">Implementation</span>
              <span className="points-chip">100 pts</span>
            </div>
            <div className="problem-limits" aria-label="Problem limits">
              <span>Time Limit: {formatTimeLimit(problem.timeLimitMs)}</span>
              <span>Memory Limit: {formatMemoryLimit(problem.memoryLimitMb)}</span>
              <span>Languages: C++17 / Python</span>
            </div>
          </header>

          <section className="problem-section">
            <h2 className="problem-section__heading">Statement</h2>
            <ProblemMarkdown content={problem.statementMarkdown} />
          </section>

          <section className="problem-section">
            <h2 className="problem-section__heading">Input</h2>
            <ProblemMarkdown content={problem.inputMarkdown} />
          </section>

          <section className="problem-section">
            <h2 className="problem-section__heading">Output</h2>
            <ProblemMarkdown content={problem.outputMarkdown} />
          </section>

          <section className="problem-section">
            <h2 className="problem-section__heading">Constraints</h2>
            <ProblemMarkdown content={problem.constraintsMarkdown} />
          </section>

          <section className="problem-section">
            <h2 className="problem-section__heading">Notes</h2>
            <ProblemMarkdown content={problem.notesMarkdown} />
          </section>
        </article>

        <section className="workspace-section workspace-section--split">
          <SolveWorkspace authEnabled={authEnabled} problemSlug={problem.slug} />
        </section>
      </section>
    </main>
  );
}
