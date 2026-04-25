"use client";

import Link from "next/link";
import { startTransition, useEffect, useEffectEvent, useState } from "react";

import { fetchPublishedProblems, formatMemoryLimit, formatProblemFetchError, formatTimeLimit, type PublishedProblemSummary } from "../lib/api";

type ProblemListState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "ready"; problems: PublishedProblemSummary[] };

export function ProblemListPage() {
  const [state, setState] = useState<ProblemListState>({ kind: "loading" });
  const [reloadToken, setReloadToken] = useState(0);

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
  }, [reloadToken]);

  function retryProblems() {
    setReloadToken((current) => current + 1);
  }

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
          <div className="history-actions">
            <button className="workspace-button workspace-button--secondary" onClick={retryProblems} type="button">
              Retry problems
            </button>
          </div>
        </section>
      ) : null}

      {state.kind === "ready" && state.problems.length === 0 ? (
        <section className="empty-state">
          <h2 className="empty-state__title">No published problems yet</h2>
          <p className="empty-state__text">Once problems are published, they will appear here for solving.</p>
          <div className="history-actions">
            <button className="workspace-button workspace-button--secondary" onClick={retryProblems} type="button">
              Refresh problemset
            </button>
          </div>
        </section>
      ) : null}

      {state.kind === "ready" && state.problems.length > 0 ? <ProblemListResults problems={state.problems} /> : null}
    </main>
  );
}

function ProblemListResults({ problems }: { problems: PublishedProblemSummary[] }) {
  return (
    <section className="table-shell">
      <div className="panel-toolbar">
        <span className="panel-toolbar__meta mono">{problems.length} published</span>
      </div>

      <div className="table-wrap responsive-table">
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
            {problems.map((problem) => (
              <tr key={problem.slug}>
                <td className="table-code">{problem.slug}</td>
                <td>
                  <Link className="table-link" href={`/problems/${problem.slug}`}>
                    {problem.title}
                  </Link>
                </td>
                <td className="table-code">{formatTimeLimit(problem.timeLimitMs)}</td>
                <td className="table-code">{formatMemoryLimit(problem.memoryLimitMb)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mobile-card-list" aria-label="Published problems cards">
        {problems.map((problem) => (
          <article className="history-card" key={`${problem.slug}-card`}>
            <div className="history-card__header">
              <div>
                <h2 className="workspace-panel__title">{problem.title}</h2>
                <p className="detail-meta mono">{problem.slug}</p>
              </div>

              <Link aria-label={`Open ${problem.title}`} className="table-link" href={`/problems/${problem.slug}`}>
                Open problem
              </Link>
            </div>

            <div className="history-card__body submission-stats" aria-label={`${problem.title} limits`}>
              <article className="submission-stat">
                <p className="submission-stat__label">Time limit</p>
                <p className="submission-stat__value">{formatTimeLimit(problem.timeLimitMs)}</p>
              </article>

              <article className="submission-stat">
                <p className="submission-stat__label">Memory limit</p>
                <p className="submission-stat__value">{formatMemoryLimit(problem.memoryLimitMb)}</p>
              </article>
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}
