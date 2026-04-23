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

export type ProblemDraftLifecycleStatus = "draft" | "in_review" | "published" | "archived";

export type ProblemDraft = {
  slug: string;
  versionNumber: number;
  lifecycleStatus: ProblemDraftLifecycleStatus;
  title: string;
  statementMarkdown: string;
  inputMarkdown: string;
  outputMarkdown: string;
  constraintsMarkdown: string;
  notesMarkdown: string;
  timeLimitMs: number;
  memoryLimitMb: number;
  hiddenTestBundleKey: string;
  hiddenTestBundleSha256: string;
};

export type ModerationQueueItem = {
  slug: string;
  versionNumber: number;
  title: string;
  submittedForReviewAt: string;
};

export type SubmissionStatus =
  | "queued"
  | "running"
  | "accepted"
  | "wrong_answer"
  | "compile_error"
  | "runtime_error"
  | "time_limit_exceeded"
  | "judge_failed";

export type Submission = {
  id: string;
  problemSlug: string;
  language: "cpp17" | "python";
  status: SubmissionStatus;
  queuedAt: string;
};

export type SubmissionResult = {
  testIndex: number;
  verdict: SubmissionStatus;
  executionTimeMs: number;
  memoryBytes: number;
  stdoutExcerpt: string;
  stderrExcerpt: string;
};

export type SubmissionDetail = Submission & {
  totalTests: number;
  passedTests: number;
  results: SubmissionResult[];
};

type PublishedProblemsResponse = {
  problems: PublishedProblemSummary[];
};

type ModerationQueueResponse = {
  drafts: ModerationQueueItem[];
};

type SubmissionsResponse = {
  submissions: Submission[];
};

export type ProblemDraftCreateInput = {
  slug: string;
  title: string;
  statementMarkdown: string;
  inputMarkdown: string;
  outputMarkdown: string;
  constraintsMarkdown: string;
  notesMarkdown: string;
  timeLimitMs: number;
  memoryLimitMb: number;
  hiddenTestBundleKey?: string;
  hiddenTestBundleSha256?: string;
};

export type ProblemDraftUpdateInput = Omit<ProblemDraftCreateInput, "slug">;

export type ModerationDecision = "approve" | "reject" | "request_changes";

type CreateSubmissionInput = {
  problemSlug: string;
  language: "cpp17" | "python";
  sourceCode: string;
};

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

const defaultApiBaseUrl = "http://127.0.0.1:8080";

function apiBaseUrl() {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? defaultApiBaseUrl;
}

type FetchJSONOptions = {
  signal?: AbortSignal;
  method?: "GET" | "POST" | "PATCH";
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
    let code: string | undefined;
    try {
      const payload = (await response.json()) as { error?: string };
      if (typeof payload.error === "string") {
        code = payload.error;
      }
    } catch {
      code = undefined;
    }

    throw new ApiError(response.status, code ?? `API request failed for ${path}`, code);
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

export async function createProblemDraft(input: ProblemDraftCreateInput, token: string) {
  return fetchJSON<ProblemDraft>("/v1/problem-drafts", {
    method: "POST",
    token,
    body: input,
  });
}

export async function fetchProblemDraft(slug: string, token: string) {
  return fetchJSON<ProblemDraft>(`/v1/problem-drafts/${slug}`, { token });
}

export async function updateProblemDraft(slug: string, input: ProblemDraftUpdateInput, token: string) {
  return fetchJSON<ProblemDraft>(`/v1/problem-drafts/${slug}`, {
    method: "PATCH",
    token,
    body: input,
  });
}

export async function submitProblemDraftForReview(slug: string, token: string) {
  return fetchJSON<ProblemDraft>(`/v1/problem-drafts/${slug}/submit-for-review`, {
    method: "POST",
    token,
  });
}

export async function fetchModerationProblemDrafts(token: string) {
  const response = await fetchJSON<ModerationQueueResponse>("/v1/moderation/problem-drafts", {
    token,
  });

  return response.drafts;
}

export async function applyModerationDecision(
  slug: string,
  input: { decision: ModerationDecision; moderationNotes: string },
  token: string,
) {
  return fetchJSON<ProblemDraft>(`/v1/moderation/problem-drafts/${slug}/decision`, {
    method: "POST",
    token,
    body: input,
  });
}

export async function createSubmission(input: CreateSubmissionInput, token: string) {
  return fetchJSON<Submission>("/v1/submissions", {
    method: "POST",
    token,
    body: input,
  });
}

export async function fetchSubmission(id: string, token: string) {
  return fetchJSON<SubmissionDetail>(`/v1/submissions/${id}`, {
    token,
  });
}

export async function fetchSubmissions(token: string) {
  const response = await fetchJSON<SubmissionsResponse>("/v1/submissions", {
    token,
  });

  return response.submissions;
}

export function formatProblemFetchError(error: unknown, missingMessage: string) {
  if (error instanceof ApiError && error.status === 404) {
    return missingMessage;
  }

  return "Unable to load data from the Cabugi API right now.";
}

export function formatDraftError(error: unknown, missingMessage: string) {
  if (error instanceof ApiError) {
    if (error.status === 404) {
      return missingMessage;
    }
    if (error.code === "problem_slug_taken") {
      return "A problem draft with that slug already exists. Choose a different slug.";
    }
    if (error.code === "invalid_hidden_test_bundle") {
      return "Hidden test bundle metadata is invalid or does not match the stored object.";
    }
    if (error.code === "problem_draft_not_ready_for_review") {
      return "Add valid hidden test bundle metadata before submitting this draft for review.";
    }
    if (error.code === "invalid_problem_lifecycle_transition") {
      return "This draft cannot be transitioned from its current state.";
    }
    if (error.status === 400) {
      return "Check the required fields, limits, and bundle metadata, then try again.";
    }
  }

  return "Unable to save draft changes right now.";
}

export function formatModerationError(error: unknown, forbiddenMessage: string) {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return forbiddenMessage;
    }
    if (error.status === 404) {
      return "The selected draft is no longer available.";
    }
  }

  return "Unable to load moderation data from the Cabugi API right now.";
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
    case "judge_failed":
      return "Judge Failed";
  }
}

export function formatSubmissionLanguage(language: Submission["language"]) {
  return language === "cpp17" ? "C++17" : "Python";
}

export function isTerminalSubmissionStatus(status: SubmissionStatus) {
  return status !== "queued" && status !== "running";
}

export function formatQueuedAt(queuedAt: string) {
  const date = new Date(queuedAt);
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}
