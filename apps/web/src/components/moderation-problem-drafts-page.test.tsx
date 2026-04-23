import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { ModerationProblemDraftsPage } from "./moderation-problem-drafts-page";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

vi.mock("@clerk/nextjs", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

function renderModeration(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
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

describe("ModerationProblemDraftsPage", () => {
  it("shows a forbidden state for non-moderators", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({ error: "forbidden" }),
      }),
    );

    renderModeration(<ModerationProblemDraftsPage authEnabled />);

    expect(await screen.findByText("Moderator access required")).toBeInTheDocument();
  });

  it("renders the moderation queue and selected draft detail", async () => {
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
        json: async () => ({ drafts: [{ slug: "two-sum-user", versionNumber: 1, title: "Two Sum User", submittedForReviewAt: new Date().toISOString() }] }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          slug: "two-sum-user",
          versionNumber: 1,
          lifecycleStatus: "in_review",
          title: "Two Sum User",
          statementMarkdown: "Solve it",
          inputMarkdown: "Input",
          outputMarkdown: "Output",
          constraintsMarkdown: "Constraints",
          notesMarkdown: "Notes",
          timeLimitMs: 1000,
          memoryLimitMb: 256,
          hiddenTestBundleKey: "bundles/two-sum.json",
          hiddenTestBundleSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderModeration(<ModerationProblemDraftsPage authEnabled />);

    expect(await screen.findByRole("button", { name: /Two Sum User/i })).toBeInTheDocument();
    expect(await screen.findByText("Hidden bundle key")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Request changes" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reject draft" })).toBeInTheDocument();
  });

  it("applies a moderation decision from the queue", async () => {
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
        json: async () => ({ drafts: [{ slug: "two-sum-user", versionNumber: 1, title: "Two Sum User", submittedForReviewAt: new Date().toISOString() }] }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          slug: "two-sum-user",
          versionNumber: 1,
          lifecycleStatus: "in_review",
          title: "Two Sum User",
          statementMarkdown: "Solve it",
          inputMarkdown: "Input",
          outputMarkdown: "Output",
          constraintsMarkdown: "Constraints",
          notesMarkdown: "Notes",
          timeLimitMs: 1000,
          memoryLimitMb: 256,
          hiddenTestBundleKey: "bundles/two-sum.json",
          hiddenTestBundleSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          slug: "two-sum-user",
          versionNumber: 1,
          lifecycleStatus: "published",
          title: "Two Sum User",
          statementMarkdown: "Solve it",
          inputMarkdown: "Input",
          outputMarkdown: "Output",
          constraintsMarkdown: "Constraints",
          notesMarkdown: "Notes",
          timeLimitMs: 1000,
          memoryLimitMb: 256,
          hiddenTestBundleKey: "bundles/two-sum.json",
          hiddenTestBundleSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ drafts: [] }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderModeration(<ModerationProblemDraftsPage authEnabled />);

    expect(await screen.findByRole("button", { name: "Approve" })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Moderation notes"), { target: { value: "Looks good" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(4);
    });

    const [url, options] = fetchMock.mock.calls[2] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:8080/v1/moderation/problem-drafts/two-sum-user/decision");
    expect(options.method).toBe("POST");
    expect(JSON.parse(String(options.body))).toMatchObject({ decision: "approve", moderationNotes: "Looks good" });
  });
});
