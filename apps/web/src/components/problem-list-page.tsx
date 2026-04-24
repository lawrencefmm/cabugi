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
        <span className="page-kicker">Problemset</span>
        <h1 className="page-title">Published problems</h1>
        <p className="page-subtitle">Browse the current set, inspect the limits, and open the solve workspace for any published problem.</p>
      </section>

      {state.kind === "loading" ? (
        <section className="empty-state" role="status" aria-live="polite">
          <h2 className="empty-state__title">Loading published problems</h2>
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
        <section className="table-shell">
          <div className="panel-toolbar">
            <span className="panel-toolbar__meta mono">{state.problems.length} published</span>
          </div>

          <div className="table-wrap">
            <table className="data-table" aria-label="Published problems">
              <thead>
                <tr>
                  <th scope="col">Slug</th>
                  <th scope="col">Title</th>
                  <th scope="col">Time</th>
                  <th scope="col">Memory</th>
                </tr>
              </thead>
              <tbody>
                {state.problems.map((problem) => (
                  <tr key={problem.slug}>
                    <td className="table-code">{problem.slug}</td>
                    <td>
                      <a className="table-link" href={`/problems/${problem.slug}`}>
                        {problem.title}
                      </a>
                    </td>
                    <td className="table-code">{formatTimeLimit(problem.timeLimitMs)}</td>
                    <td className="table-code">{formatMemoryLimit(problem.memoryLimitMb)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}
    </main>
  );
}
