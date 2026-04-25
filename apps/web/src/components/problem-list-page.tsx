"use client";

import Link from "next/link";
import { startTransition, useEffect, useEffectEvent, useState } from "react";

import { fetchPublishedProblems, formatMemoryLimit, formatProblemFetchError, formatTimeLimit, type PublishedProblemSummary } from "../lib/api";

const supportedLanguagesLabel = "C++17 / Python";

const topicProgress = [
  ["Arrays & Hashing", 78],
  ["Binary Search", 64],
  ["Graphs", 55],
  ["Dynamic Programming", 43],
  ["Trees", 37],
  ["Greedy", 29],
  ["Math", 21],
] as const;

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
    <main className="app-main app-main--wide problemset-page">
      <div className="problemset-layout">
        <ProblemsetSidebar />

        <section className="problemset-content" aria-label="Problem set content">
          <section className="page-hero problemset-hero">
            <span className="page-kicker">Browse</span>
            <h1 className="page-title">Problem Set</h1>
            <p className="page-subtitle">Explore problems, track progress, and sharpen your skills in a dense competitive-programming cockpit.</p>
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
        </section>

        <ProblemsetAside />
      </div>
    </main>
  );
}

function ProblemListResults({ problems }: { problems: PublishedProblemSummary[] }) {
  const solvedCount = Math.min(problems.length, Math.max(0, Math.round(problems.length * 0.25)));
  const attemptedCount = Math.min(problems.length, Math.max(solvedCount, Math.round(problems.length * 0.45)));

  return (
    <section className="problemset-results">
      <div className="problemset-metrics" aria-label="Problem set overview">
        <MetricCard label="Total Problems" value={problems.length.toLocaleString()} tone="red" />
        <MetricCard label="Solved" value={solvedCount.toLocaleString()} tone="green" />
        <MetricCard label="Attempted" value={attemptedCount.toLocaleString()} tone="blue" />
        <MetricCard label="Favorite Tags" value="14" tone="gold" />
      </div>

      <div className="problemset-filters" aria-label="Problem filters">
        <label className="problemset-search">
          <span className="sr-only">Search problems</span>
          <input type="search" placeholder="Search problems..." />
        </label>
        <select aria-label="Difficulty filter" defaultValue="">
          <option value="">Difficulty</option>
          <option>Easy</option>
          <option>Medium</option>
          <option>Hard</option>
        </select>
        <select aria-label="Tags filter" defaultValue="">
          <option value="">Tags</option>
          <option>Array</option>
          <option>Graph</option>
          <option>Dynamic Programming</option>
        </select>
        <select aria-label="Status filter" defaultValue="">
          <option value="">Status</option>
          <option>Solved</option>
          <option>Attempted</option>
          <option>To-do</option>
        </select>
        <select aria-label="Language filter" defaultValue="">
          <option value="">Language</option>
          <option>C++17</option>
          <option>Python</option>
        </select>
        <button className="icon-button" type="button" aria-label="Open advanced filters">
          F
        </button>
      </div>

      <section className="table-shell problemset-table-shell" id="problemset-table">
        <div className="table-wrap responsive-table">
          <table className="data-table data-table--problemset" aria-label="Published problems">
            <thead>
              <tr>
                <th scope="col">Status</th>
                <th scope="col">Problem</th>
                <th scope="col">Difficulty</th>
                <th scope="col">Tags</th>
                <th scope="col">Acceptance</th>
                <th scope="col">Last Updated</th>
                <th scope="col">Save</th>
              </tr>
            </thead>
            <tbody>
              {problems.map((problem, index) => {
                const difficulty = difficultyFor(index);
                const tags = tagsFor(index);
                const acceptance = acceptanceFor(index);

                return (
                  <tr key={problem.slug}>
                    <td>
                      <span className={`problem-status problem-status--${statusFor(index)}`} aria-label={statusLabelFor(index)} />
                    </td>
                    <td>
                      <div className="problem-list-entry">
                        <Link className="table-link problem-list-entry__title" href={`/problems/${problem.slug}`}>
                          {problem.title}
                        </Link>
                        <p className="detail-meta problem-list-entry__slug">{problem.slug} · {formatTimeLimit(problem.timeLimitMs)} · {formatMemoryLimit(problem.memoryLimitMb)} · {supportedLanguagesLabel}</p>
                      </div>
                    </td>
                    <td>
                      <span className={`difficulty-pill difficulty-pill--${difficulty.toLowerCase()}`}>{difficulty}</span>
                    </td>
                    <td>
                      <div className="tag-row">
                        {tags.map((tag) => (
                          <span className="tag-chip" key={`${problem.slug}-${tag}`}>{tag}</span>
                        ))}
                      </div>
                    </td>
                    <td>
                      <div className="acceptance-cell">
                        <span>{acceptance}%</span>
                        <span className="acceptance-meter" aria-hidden="true">
                          <span style={{ width: `${acceptance}%` }} />
                        </span>
                      </div>
                    </td>
                    <td className="table-code">{lastUpdatedFor(index)}</td>
                    <td>
                      <span className="bookmark-icon" aria-label={`Bookmark ${problem.title}`} />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        <div className="mobile-card-list" aria-label="Published problems cards">
          {problems.map((problem, index) => {
            const difficulty = difficultyFor(index);

            return (
              <article className="history-card problem-mobile-card" key={`${problem.slug}-card`}>
                <div className="history-card__header">
                  <div>
                    <h2 className="workspace-panel__title">{problem.title}</h2>
                    <p className="detail-meta">{problem.slug}</p>
                  </div>

                  <span className={`difficulty-pill difficulty-pill--${difficulty.toLowerCase()}`}>{difficulty}</span>
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

                  <article className="submission-stat">
                    <p className="submission-stat__label">Languages</p>
                    <p className="submission-stat__value">{supportedLanguagesLabel}</p>
                  </article>
                </div>

                <div className="history-card__footer">
                  <Link aria-label={`Open ${problem.title}`} className="table-link" href={`/problems/${problem.slug}`}>
                    Open problem
                  </Link>
                </div>
              </article>
            );
          })}
        </div>
      </section>
    </section>
  );
}

function ProblemsetSidebar() {
  return (
    <aside className="problemset-sidebar" aria-label="Problem set navigation">
      <p className="sidebar-label">Browse</p>
      <a className="sidebar-link sidebar-link--active" href="#problemset-table">
        <span aria-hidden="true">[]</span> All Problems
      </a>
      <a className="sidebar-link" href="#topics">
        <span aria-hidden="true">#</span> By Tags
      </a>
      <a className="sidebar-link" href="#topics">
        <span aria-hidden="true">@</span> By Topics
      </a>
      <a className="sidebar-link" href="#difficulty">
        <span aria-hidden="true">%</span> By Difficulty
      </a>

      <div className="sidebar-divider" />
      <p className="sidebar-label">My Lists</p>
      <SidebarCount label="Solved" value="312" />
      <SidebarCount label="Attempted" value="567" />
      <SidebarCount label="Bookmarked" value="23" />
      <SidebarCount label="To-Do" value="17" />

      <div className="sidebar-divider" />
      <p className="sidebar-label" id="topics">Topics</p>
      {topicProgress.map(([topic, value]) => (
        <ProgressLine key={topic} label={topic} value={value} />
      ))}

      <div className="study-plan-card">
        <span aria-hidden="true">M</span>
        <strong>Study Plan</strong>
        <p>Build a personalized plan to level up faster.</p>
      </div>
    </aside>
  );
}

function ProblemsetAside() {
  return (
    <aside className="problemset-aside" aria-label="Practice progress">
      <section className="side-card streak-card">
        <h2>Daily Streak</h2>
        <p className="streak-card__value"><span aria-hidden="true">F</span> 12 <small>days</small></p>
        <p className="detail-meta">Best: 24 days</p>
        <div className="streak-days" aria-hidden="true">
          {Array.from({ length: 7 }, (_, index) => <span key={index}>{index === 6 ? "S" : "F"}</span>)}
        </div>
      </section>

      <section className="side-card">
        <div className="side-card__header">
          <h2>Topic Progress</h2>
          <a href="#topics">View all</a>
        </div>
        {topicProgress.map(([topic, value]) => (
          <ProgressLine key={`aside-${topic}`} label={topic} value={value} />
        ))}
      </section>

      <section className="side-card recommendations-card">
        <h2>Recommended for you</h2>
        <Recommendation title="Sliding Window Maximum" difficulty="Hard" score="48%" />
        <Recommendation title="Monotonic Stack" difficulty="Medium" score="62%" />
        <Recommendation title="Word Ladder" difficulty="Medium" score="58%" />
        <a className="side-card__cta" href="#problemset-table">View more recommendations</a>
      </section>
    </aside>
  );
}

function MetricCard({ label, tone, value }: { label: string; tone: "red" | "green" | "blue" | "gold"; value: string }) {
  return (
    <article className={`metric-card metric-card--${tone}`}>
      <span className="metric-card__icon" aria-hidden="true" />
      <div>
        <p className="metric-card__value">{value}</p>
        <p className="metric-card__label">{label}</p>
      </div>
    </article>
  );
}

function SidebarCount({ label, value }: { label: string; value: string }) {
  return (
    <div className="sidebar-count">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function ProgressLine({ label, value }: { label: string; value: number }) {
  return (
    <div className="progress-line">
      <div>
        <span>{label}</span>
        <strong>{value}%</strong>
      </div>
      <span className="progress-line__bar" aria-hidden="true">
        <span style={{ width: `${value}%` }} />
      </span>
    </div>
  );
}

function Recommendation({ difficulty, score, title }: { difficulty: "Hard" | "Medium"; score: string; title: string }) {
  return (
    <article className="recommendation">
      <div>
        <h3>{title}</h3>
        <p><span>{difficulty}</span> {score}</p>
      </div>
    </article>
  );
}

function difficultyFor(index: number) {
  if (index % 5 === 4) {
    return "Hard";
  }

  if (index % 3 === 2) {
    return "Medium";
  }

  return "Easy";
}

function tagsFor(index: number) {
  const tagSets = [
    ["Array", "Hash Table"],
    ["Binary Search", "Array"],
    ["Graph", "DFS"],
    ["Segment Tree", "Data Structure"],
    ["DP", "Greedy"],
  ];

  return tagSets[index % tagSets.length];
}

function acceptanceFor(index: number) {
  return [87, 91, 62, 58, 46, 54, 71, 49, 63, 35][index % 10];
}

function statusFor(index: number) {
  return ["solved", "solved", "attempted", "solved", "failed", "attempted", "solved", "todo"][index % 8];
}

function statusLabelFor(index: number) {
  const status = statusFor(index);
  if (status === "solved") {
    return "Solved";
  }
  if (status === "attempted") {
    return "Attempted";
  }
  if (status === "failed") {
    return "Needs retry";
  }
  return "To-do";
}

function lastUpdatedFor(index: number) {
  return ["2 days ago", "3 days ago", "5 days ago", "1 week ago", "2 weeks ago", "3 weeks ago", "1 month ago"][index % 7];
}
