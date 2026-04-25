import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { SubmissionSummaryPage } from "./submission-summary-page";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

vi.mock("./auth", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

function renderSubmission(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("SubmissionSummaryPage", () => {
  it("fetches the submission detail endpoint and renders verdict breakdown", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        id: "submission-1",
        problemSlug: "two-sum",
        language: "cpp17",
        status: "wrong_answer",
        queuedAt: new Date().toISOString(),
        totalTests: 3,
        passedTests: 2,
        compileOutputExcerpt: "",
        results: [
          { testIndex: 0, verdict: "accepted", executionTimeMs: 9, memoryBytes: 0, stdoutExcerpt: "42\n", stderrExcerpt: "" },
          { testIndex: 2, verdict: "wrong_answer", executionTimeMs: 14, memoryBytes: 0, stdoutExcerpt: "41\n", stderrExcerpt: "" },
        ],
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    renderSubmission(<SubmissionSummaryPage authEnabled submissionId="submission-1" />);

    expect(await screen.findByText("Final verdict")).toBeInTheDocument();
    expect(screen.getByText("Passed tests")).toBeInTheDocument();
    expect(screen.getByText("Total tests")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("Test 3")).toBeInTheDocument();
    expect(screen.getAllByText("Stdout excerpt")).toHaveLength(2);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "/api/v1/submissions/submission-1",
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: "Bearer session-token",
          }),
        }),
      );
    });
  });

  it("polls a live submission until a terminal verdict arrives", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          id: "submission-live",
          problemSlug: "two-sum",
          language: "cpp17",
          status: "queued",
          queuedAt: new Date().toISOString(),
          totalTests: 0,
          passedTests: 0,
          compileOutputExcerpt: "",
          results: [],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          id: "submission-live",
          problemSlug: "two-sum",
          language: "cpp17",
          status: "running",
          queuedAt: new Date().toISOString(),
          totalTests: 1,
          passedTests: 0,
          compileOutputExcerpt: "",
          results: [],
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          id: "submission-live",
          problemSlug: "two-sum",
          language: "cpp17",
          status: "accepted",
          queuedAt: new Date().toISOString(),
          totalTests: 1,
          passedTests: 1,
          compileOutputExcerpt: "",
          results: [{ testIndex: 0, verdict: "accepted", executionTimeMs: 6, memoryBytes: 0, stdoutExcerpt: "4\n", stderrExcerpt: "" }],
        }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderSubmission(<SubmissionSummaryPage authEnabled pollIntervalMs={10} submissionId="submission-live" />);

    expect(await screen.findByText("Live status stream")).toBeInTheDocument();
    expect((await screen.findAllByText("Accepted")).length).toBeGreaterThan(0);
    expect(await screen.findByText("Test 1")).toBeInTheDocument();
    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(3);
    });
  });

  it("handles compile errors without per-test result rows gracefully", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          id: "submission-2",
          problemSlug: "broken-solution",
          language: "cpp17",
          status: "compile_error",
          queuedAt: new Date().toISOString(),
          totalTests: 0,
          passedTests: 0,
          compileOutputExcerpt: "main.cpp:1: error: expected ';'",
          results: [],
        }),
      }),
    );

    renderSubmission(<SubmissionSummaryPage authEnabled submissionId="submission-2" />);

    expect(await screen.findByText("Per-test results unavailable")).toBeInTheDocument();
    expect(screen.getByText("No per-test results were recorded because compilation failed before execution started.")).toBeInTheDocument();
    expect(screen.getByText("Compile output")).toBeInTheDocument();
    expect(screen.getByText("main.cpp:1: error: expected ';'")) .toBeInTheDocument();
  });
});
