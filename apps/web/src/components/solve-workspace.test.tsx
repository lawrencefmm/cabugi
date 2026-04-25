import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { starterCodeTemplates } from "../lib/starter-code";
import { SolveWorkspace } from "./solve-workspace";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
  mode: "disabled" | "clerk" | "local_test";
};

const pushMock = vi.fn();

function createStorageMock() {
  const store = new Map<string, string>();

  return {
    clear: vi.fn(() => {
      store.clear();
    }),
    getItem: vi.fn((key: string) => store.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store.set(key, value);
    }),
  };
}

let storageMock = createStorageMock();

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

beforeEach(() => {
  storageMock = createStorageMock();
  vi.stubGlobal("localStorage", storageMock);
  Object.defineProperty(window, "localStorage", {
    configurable: true,
    value: storageMock,
  });
});

afterEach(() => {
  pushMock.mockReset();
  storageMock.clear();
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

  it("creates a submission, persists the latest run, and redirects to the live submission page", async () => {
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
    expect(screen.getByRole("link", { name: "Reopen latest submission" })).toHaveAttribute("href", "/submissions/submission-1");
  });

  it("preserves edited buffers when switching languages", () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
      mode: "clerk",
    };

    renderWorkspace(<SolveWorkspace authEnabled problemSlug="two-sum" />);

    const languagePicker = screen.getByLabelText("Language");

    fireEvent.change(screen.getByTestId("monaco-editor"), { target: { value: "int main() { return 42; }" } });
    fireEvent.change(languagePicker, { target: { value: "python" } });

    expect(screen.getByTestId("monaco-editor")).toHaveValue(starterCodeTemplates.python);

    fireEvent.change(screen.getByTestId("monaco-editor"), { target: { value: "print(42)\n" } });
    fireEvent.change(languagePicker, { target: { value: "cpp17" } });

    expect(screen.getByTestId("monaco-editor")).toHaveValue("int main() { return 42; }");

    fireEvent.change(languagePicker, { target: { value: "python" } });

    expect(screen.getByTestId("monaco-editor")).toHaveValue("print(42)\n");
  });

  it("hydrates the latest submission shortcut from local storage", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
      mode: "clerk",
    };

    storageMock.setItem(
      "cabugi_solve_workspace:two-sum",
      JSON.stringify({
        language: "python",
        latestSubmissionId: "submission-9",
        sources: {
          cpp17: starterCodeTemplates.cpp17,
          python: "print('cached')\n",
        },
      }),
    );

    renderWorkspace(<SolveWorkspace authEnabled problemSlug="two-sum" />);

    await waitFor(() => {
      expect(screen.getByRole("link", { name: "Reopen latest submission" })).toHaveAttribute("href", "/submissions/submission-9");
    });
    expect(screen.getByLabelText("Language")).toHaveValue("python");
    expect(screen.getByTestId("monaco-editor")).toHaveValue("print('cached')\n");
  });

  it("renders actionable submission failure messages", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
      mode: "clerk",
    };

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        json: async () => ({ error: "not_found" }),
      }),
    );

    renderWorkspace(<SolveWorkspace authEnabled problemSlug="two-sum" />);

    fireEvent.click(screen.getByRole("button", { name: "Submit solution" }));

    expect(await screen.findByText("This published problem is no longer available for new submissions.")).toBeInTheDocument();
  });
});
