"use client";

import { startTransition, useEffect, useEffectEvent, useState } from "react";

import { fetchPublishedProblems, formatMemoryLimit, formatProblemFetchError, formatTimeLimit, type PublishedProblemSummary } from "../lib/api";

type ProblemListState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; problems: PublishedProblemSummary[] };

export function ProblemListPage() {
  const [state, setState] = useState<ProblemListState>({ kind: "loading" });

  const loadProblems = useEffectEvent(async (signal: AbortSignal) => {
    try {
      const problems = await fetchPublishedProblems(signal);
      if (signal.aborted) {
        return;
      }

      startTransition(() => {
        setState({ kind: "ready", problems });
      });
    } catch (error) {
      if (signal.aborted) {
        return;
      }

      startTransition(() => {
        setState({
          kind: "error",
          message: formatProblemFetchError(error, "Published problems could not be found."),
        });
      });
    }
  });

  useEffect(() => {
    const controller = new AbortController();
    startTransition(() => {
      setState({ kind: "loading" });
    });

    void loadProblems(controller.signal);

    return () => {
      controller.abort();
    };
  }, []);

  return (
    <main className="app-main">
      <section className="page-hero">
        <span className="page-kicker">Published Problems</span>
        <h1 className="page-title">Train on problems that feel contest-ready.</h1>
        <p className="page-subtitle">
          Cabugi focuses on a clean practice flow: browse a curated problem set, read complete statements, and step straight into solving.
        </p>
      </section>

      {state.kind === "loading" ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading published problems</h2>
          <p className="empty-state__text">Fetching the current problem set from the API.</p>
        </section>
      ) : null}

      {state.kind === "error" ? (
        <section className="error-state" role="alert">
          <h2 className="error-state__title">Problem list unavailable</h2>
          <p className="error-state__text">{state.message}</p>
        </section>
      ) : null}

      {state.kind === "ready" && state.problems.length === 0 ? (
        <section className="empty-state">
          <h2 className="empty-state__title">No published problems yet</h2>
          <p className="empty-state__text">Once problems are published, they will appear here for solving.</p>
        </section>
      ) : null}

      {state.kind === "ready" && state.problems.length > 0 ? (
        <section className="problem-grid" aria-label="Published problems">
          {state.problems.map((problem) => (
            <a className="problem-card" href={`/problems/${problem.slug}`} key={problem.slug}>
              <div className="problem-card__slug">{problem.slug}</div>
              <h2 className="problem-card__title">{problem.title}</h2>
              <div className="problem-card__meta">
                <span>{formatTimeLimit(problem.timeLimitMs)}</span>
                <span>{formatMemoryLimit(problem.memoryLimitMb)}</span>
              </div>
            </a>
          ))}
        </section>
      ) : null}
    </main>
  );
}
