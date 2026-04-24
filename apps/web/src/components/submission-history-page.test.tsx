import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { type ReactNode } from "react";

import { SubmissionHistoryPage } from "./submission-history-page";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

vi.mock("./auth", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

function renderHistory(ui: ReactNode) {
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

describe("SubmissionHistoryPage", () => {
  it("renders a sign-in prompt when the user is signed out", () => {
    mockAuthState = {
      getToken: async () => null,
      isLoaded: true,
      isSignedIn: false,
    };

    renderHistory(<SubmissionHistoryPage authEnabled />);

    expect(screen.getByRole("button", { name: "Sign in to continue" })).toBeInTheDocument();
  });

  it("renders submission history rows with links", async () => {
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
          submissions: [
            { id: "submission-2", problemSlug: "two-sum", language: "cpp17", status: "accepted", queuedAt: new Date().toISOString() },
          ],
        }),
      }),
    );

    renderHistory(<SubmissionHistoryPage authEnabled />);

    expect(await screen.findByRole("link", { name: "two-sum" })).toHaveAttribute("href", "/problems/two-sum");
    expect(screen.getByRole("link", { name: "View submission" })).toHaveAttribute("href", "/submissions/submission-2");
  });

  it("renders an empty state when there are no submissions", async () => {
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
        json: async () => ({ submissions: [] }),
      }),
    );

    renderHistory(<SubmissionHistoryPage authEnabled />);

    expect(await screen.findByText("No submissions yet")).toBeInTheDocument();
  });
});
