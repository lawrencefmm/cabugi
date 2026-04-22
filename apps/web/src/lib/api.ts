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

export type SubmissionStatus =
  | "queued"
  | "running"
  | "accepted"
  | "wrong_answer"
  | "compile_error"
  | "runtime_error"
  | "time_limit_exceeded";

export type Submission = {
  id: string;
  problemSlug: string;
  language: "cpp17" | "python";
  status: SubmissionStatus;
  queuedAt: string;
};

type PublishedProblemsResponse = {
  problems: PublishedProblemSummary[];
};

type CreateSubmissionInput = {
  problemSlug: string;
  language: "cpp17" | "python";
  sourceCode: string;
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

type FetchJSONOptions = {
  signal?: AbortSignal;
  method?: "GET" | "POST";
  token?: string;
  body?: object;
};

async function fetchJSON<T>(path: string, options: FetchJSONOptions = {}): Promise<T> {
  const { signal, method = "GET", token, body } = options;
  const response = await fetch(`${apiBaseUrl()}${path}`, {
    method,
    headers: {
      Accept: "application/json",
      ...(body ? { "Content-Type": "application/json" } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    signal,
  });

  if (!response.ok) {
    throw new ApiError(response.status, `API request failed for ${path}`);
  }

  return response.json() as Promise<T>;
}

export async function fetchPublishedProblems(signal?: AbortSignal) {
  const response = await fetchJSON<PublishedProblemsResponse>("/v1/problems", { signal });
  return response.problems;
}

export async function fetchPublishedProblem(slug: string, signal?: AbortSignal) {
  return fetchJSON<PublishedProblemDetail>(`/v1/problems/${slug}`, { signal });
}

export async function createSubmission(input: CreateSubmissionInput, token: string) {
  return fetchJSON<Submission>("/v1/submissions", {
    method: "POST",
    token,
    body: input,
  });
}

export async function fetchSubmission(id: string, token: string) {
  return fetchJSON<Submission>(`/v1/submissions/${id}`, {
    token,
  });
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

export function formatSubmissionStatus(status: SubmissionStatus) {
  switch (status) {
    case "queued":
      return "Queued";
    case "running":
      return "Running";
    case "accepted":
      return "Accepted";
    case "wrong_answer":
      return "Wrong Answer";
    case "compile_error":
      return "Compile Error";
    case "runtime_error":
      return "Runtime Error";
    case "time_limit_exceeded":
      return "Time Limit Exceeded";
  }
}

export function isTerminalSubmissionStatus(status: SubmissionStatus) {
  return status !== "queued" && status !== "running";
}
