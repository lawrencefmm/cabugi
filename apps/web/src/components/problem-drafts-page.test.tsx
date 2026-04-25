import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { type ReactNode } from "react";

import { ProblemDraftsPage } from "./problem-drafts-page";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

vi.mock("./auth", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

function renderDrafts(ui: ReactNode) {
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

describe("ProblemDraftsPage", () => {
  it("renders a sign-in prompt when the user is signed out", () => {
    mockAuthState = {
      getToken: async () => null,
      isLoaded: true,
      isSignedIn: false,
    };

    renderDrafts(<ProblemDraftsPage authEnabled />);

    expect(screen.getByRole("button", { name: "Sign in to continue" })).toBeInTheDocument();
  });

  it("renders an empty state with a new draft entry point", async () => {
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
        json: async () => ({ drafts: [] }),
      }),
    );

    renderDrafts(<ProblemDraftsPage authEnabled />);

    expect(await screen.findByText("No drafts yet")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Create your first draft" })).toHaveAttribute("href", "/drafts/new");
  });

  it("renders existing drafts with reopen links", async () => {
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
          drafts: [
            {
              slug: "two-sum-user",
              versionNumber: 1,
              lifecycleStatus: "draft",
              title: "Two Sum User",
              updatedAt: "2026-04-25T10:00:00Z",
              submittedForReviewAt: null,
            },
            {
              slug: "a-plus-b-user",
              versionNumber: 2,
              lifecycleStatus: "in_review",
              title: "A + B User",
              updatedAt: "2026-04-24T10:00:00Z",
              submittedForReviewAt: "2026-04-24T09:00:00Z",
            },
          ],
        }),
      }),
    );

    renderDrafts(<ProblemDraftsPage authEnabled />);

    expect(await screen.findByRole("link", { name: "Two Sum User" })).toHaveAttribute("href", "/drafts/two-sum-user");
    expect(screen.getByRole("link", { name: "Resume editing" })).toHaveAttribute("href", "/drafts/two-sum-user");
    expect(screen.getByRole("link", { name: "Open draft" })).toHaveAttribute("href", "/drafts/a-plus-b-user");
    expect(screen.getByText("In Review")).toBeInTheDocument();
  });
});
