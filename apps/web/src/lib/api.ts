export type PublishedProblemSummary = {
  slug: string;
  title: string;
  timeLimitMs: number;
  memoryLimitMb: number;
};

export type PublishedProblemDetail = PublishedProblemSummary & {
  statementMarkdown: string;
  inputMarkdown: string;
  outputMarkdown: string;
  constraintsMarkdown: string;
  notesMarkdown: string;
};

type PublishedProblemsResponse = {
  problems: PublishedProblemSummary[];
};

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const defaultApiBaseUrl = "http://127.0.0.1:8080";

function apiBaseUrl() {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? defaultApiBaseUrl;
}

async function fetchJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${apiBaseUrl()}${path}`, {
    headers: {
      Accept: "application/json",
    },
    signal,
  });

  if (!response.ok) {
    throw new ApiError(response.status, `API request failed for ${path}`);
  }

  return response.json() as Promise<T>;
}

export async function fetchPublishedProblems(signal?: AbortSignal) {
  const response = await fetchJSON<PublishedProblemsResponse>("/v1/problems", signal);
  return response.problems;
}

export async function fetchPublishedProblem(slug: string, signal?: AbortSignal) {
  return fetchJSON<PublishedProblemDetail>(`/v1/problems/${slug}`, signal);
}

export function formatProblemFetchError(error: unknown, missingMessage: string) {
  if (error instanceof ApiError && error.status === 404) {
    return missingMessage;
  }

  return "Unable to load data from the Cabugi API right now.";
}

export function formatTimeLimit(timeLimitMs: number) {
  return `${timeLimitMs} ms`;
}

export function formatMemoryLimit(memoryLimitMb: number) {
  return `${memoryLimitMb} MB`;
}
