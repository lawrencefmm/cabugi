"use client";

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
  }, [slug]);

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
        </section>
      </main>
    );
  }

  const { problem } = state;

  return (
    <main className="app-main">
      <section className="page-hero">
        <a className="status-note" href="/">
          Problems / {problem.slug}
        </a>
        <span className="page-kicker">Problem</span>
        <h1 className="page-title">{problem.title}</h1>
        <p className="page-subtitle">Read the statement, inspect the limits for this published version, then move directly into the solve workspace.</p>
        <div className="meta-strip" aria-label="Problem metadata overview">
          <span className="meta-chip mono">{problem.slug}</span>
          <span className="meta-chip mono">{formatTimeLimit(problem.timeLimitMs)}</span>
          <span className="meta-chip mono">{formatMemoryLimit(problem.memoryLimitMb)}</span>
          <span className="meta-chip mono">C++17 / Python</span>
        </div>
      </section>

      <section className="problem-layout">
        <article className="problem-panel">
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

        <aside className="problem-sidebar">
          <div className="problem-sidebar__section">
            <h2 className="problem-sidebar__heading">Metadata</h2>
            <div className="problem-sidebar__list">
              <div>
                <span className="problem-sidebar__meta-label">Slug</span>
                <span className="problem-sidebar__meta-value">{problem.slug}</span>
              </div>
              <div>
                <span className="problem-sidebar__meta-label">Time limit</span>
                <span className="problem-sidebar__meta-value">{formatTimeLimit(problem.timeLimitMs)}</span>
              </div>
              <div>
                <span className="problem-sidebar__meta-label">Memory limit</span>
                <span className="problem-sidebar__meta-value">{formatMemoryLimit(problem.memoryLimitMb)}</span>
              </div>
            </div>
          </div>

          <div className="problem-sidebar__section">
            <h2 className="problem-sidebar__heading">Languages</h2>
            <div className="problem-sidebar__list">
              <span className="problem-sidebar__meta-value">C++17</span>
              <span className="problem-sidebar__meta-value">Python</span>
            </div>
          </div>
        </aside>
      </section>

      <section className="workspace-section">
        <SolveWorkspace authEnabled={authEnabled} problemSlug={problem.slug} />
      </section>
    </main>
  );
}
