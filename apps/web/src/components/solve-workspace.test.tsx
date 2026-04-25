import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { SolveWorkspace } from "./solve-workspace";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
  mode: "disabled" | "clerk" | "local_test";
};

const pushMock = vi.fn();

vi.mock("./auth", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
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
      mutations: {
        retry: false,
      },
    },
  });

  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  pushMock.mockReset();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("SolveWorkspace", () => {
  it("renders a sign-in prompt when the user is not authenticated", () => {
    mockAuthState = {
      getToken: async () => null,
      isLoaded: true,
      isSignedIn: false,
      mode: "clerk",
    };

    renderWorkspace(<SolveWorkspace authEnabled problemSlug="two-sum" />);

    expect(screen.getByRole("button", { name: "Sign in to solve" })).toBeInTheDocument();
  });

  it("creates a submission and redirects to the live submission page", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
      mode: "clerk",
    };

    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 201,
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

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/submissions/submission-1");
    });
  });
});
