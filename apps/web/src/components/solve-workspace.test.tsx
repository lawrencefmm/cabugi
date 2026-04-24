import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { SolveWorkspace } from "./solve-workspace";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

vi.mock("./auth", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

vi.mock("@monaco-editor/react", () => ({
  default: ({ onChange, value }: { onChange: (value: string) => void; value: string }) => (
    <textarea data-testid="monaco-editor" value={value} onChange={(event) => onChange(event.target.value)} />
  ),
}));

function renderWorkspace(ui: ReactNode) {
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

describe("SolveWorkspace", () => {
  it("renders a sign-in prompt when the user is not authenticated", () => {
    mockAuthState = {
      getToken: async () => null,
      isLoaded: true,
      isSignedIn: false,
    };

    renderWorkspace(<SolveWorkspace authEnabled problemSlug="two-sum" />);

    expect(screen.getByRole("button", { name: "Sign in to solve" })).toBeInTheDocument();
  });

  it("creates a submission with the expected payload", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "queued", queuedAt: new Date().toISOString() }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "queued", queuedAt: new Date().toISOString() }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderWorkspace(<SolveWorkspace authEnabled problemSlug="two-sum" />);

    fireEvent.click(screen.getByRole("button", { name: "Submit solution" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalled();
    });

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:8080/v1/submissions");
    expect(options.method).toBe("POST");
    expect(options.headers).toMatchObject({
      Authorization: "Bearer session-token",
      "Content-Type": "application/json",
    });
    expect(JSON.parse(String(options.body))).toMatchObject({
      language: "cpp17",
      problemSlug: "two-sum",
    });
  });

  it("polls submission status until a terminal verdict is reached", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "queued", queuedAt: new Date().toISOString() }),
    })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "running", queuedAt: new Date().toISOString() }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "accepted", queuedAt: new Date().toISOString() }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderWorkspace(<SolveWorkspace authEnabled pollIntervalMs={10} problemSlug="two-sum" />);

    fireEvent.click(screen.getByRole("button", { name: "Submit solution" }));

    expect(await screen.findByText("Running")).toBeInTheDocument();
    expect(await screen.findByText("Accepted")).toBeInTheDocument();
  });

  it("treats judge failures as terminal submission states", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "queued", queuedAt: new Date().toISOString() }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "running", queuedAt: new Date().toISOString() }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({ id: "submission-1", problemSlug: "two-sum", language: "cpp17", status: "judge_failed", queuedAt: new Date().toISOString(), totalTests: 0, passedTests: 0, results: [] }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderWorkspace(<SolveWorkspace authEnabled pollIntervalMs={10} problemSlug="two-sum" />);

    fireEvent.click(screen.getByRole("button", { name: "Submit solution" }));

    expect(await screen.findByText("Judge Failed")).toBeInTheDocument();
  });
});
