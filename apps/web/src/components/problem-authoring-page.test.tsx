import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { ProblemAuthoringPage } from "./problem-authoring-page";

let mockAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
};

const pushMock = vi.fn();

vi.mock("./auth", () => ({
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockAuthState,
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
}));

function renderAuthoring(ui: ReactNode) {
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

describe("ProblemAuthoringPage", () => {
  it("renders a sign-in prompt when the user is signed out", () => {
    mockAuthState = {
      getToken: async () => null,
      isLoaded: true,
      isSignedIn: false,
    };

    renderAuthoring(<ProblemAuthoringPage authEnabled />);

    expect(screen.getByRole("button", { name: "Sign in to continue" })).toBeInTheDocument();
  });

  it("renders live preview and readiness cues while editing a new draft", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    renderAuthoring(<ProblemAuthoringPage authEnabled />);

    expect(screen.getByRole("heading", { name: "Live Preview" })).toBeInTheDocument();
    expect(screen.getByText("Start writing the statement to preview it here.")).toBeInTheDocument();
    expect(screen.getByText("1 of 7 checks complete. Create the draft first, then come back to submit it for review once the checklist is green.")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Slug"), { target: { value: "two-sum-user" } });
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Two Sum User" } });
    fireEvent.change(screen.getByLabelText("Statement Markdown"), { target: { value: "# Two Sum" } });
    fireEvent.change(screen.getByLabelText("Input Markdown"), { target: { value: "Two integers." } });
    fireEvent.change(screen.getByLabelText("Output Markdown"), { target: { value: "Their sum." } });
    fireEvent.change(screen.getByLabelText("Constraints Markdown"), { target: { value: "1 <= n <= 10^5" } });

    expect(await screen.findByRole("heading", { name: "Two Sum" })).toBeInTheDocument();
    expect(screen.getAllByText("Two integers.").length).toBeGreaterThan(0);
    expect(screen.getByText("6 of 7 checks complete. Create the draft first, then come back to submit it for review once the checklist is green.")).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Draft readiness checks" })).toBeInTheDocument();
  });

  it("creates a draft and redirects to the edit page", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({
        slug: "two-sum-user",
        versionNumber: 1,
        lifecycleStatus: "draft",
        title: "Two Sum User",
        statementMarkdown: "Solve it",
        inputMarkdown: "Input",
        outputMarkdown: "Output",
        constraintsMarkdown: "Constraints",
        notesMarkdown: "Notes",
        timeLimitMs: 1000,
        memoryLimitMb: 256,
        hiddenTestBundleKey: "",
        hiddenTestBundleSha256: "",
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    renderAuthoring(<ProblemAuthoringPage authEnabled />);

    fireEvent.change(screen.getByLabelText("Slug"), { target: { value: "two-sum-user" } });
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Two Sum User" } });
    fireEvent.change(screen.getByLabelText("Statement Markdown"), { target: { value: "Solve it" } });
    fireEvent.change(screen.getByLabelText("Input Markdown"), { target: { value: "Input" } });
    fireEvent.change(screen.getByLabelText("Output Markdown"), { target: { value: "Output" } });
    fireEvent.change(screen.getByLabelText("Constraints Markdown"), { target: { value: "Constraints" } });
    fireEvent.change(screen.getByLabelText("Notes Markdown"), { target: { value: "Notes" } });
    fireEvent.click(screen.getByRole("button", { name: "Create draft" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalled();
      expect(pushMock).toHaveBeenCalledWith("/drafts/two-sum-user");
    });

    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/problem-drafts");
    expect(options.method).toBe("POST");
  });

  it("uploads a hidden bundle and includes the returned metadata when creating a draft", async () => {
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
        json: async () => ({
          hiddenTestBundleKey: "problem-drafts/user-id/uploaded-bundle.json",
          hiddenTestBundleSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => ({
          slug: "two-sum-user",
          versionNumber: 1,
          lifecycleStatus: "draft",
          title: "Two Sum User",
          statementMarkdown: "Solve it",
          inputMarkdown: "Input",
          outputMarkdown: "Output",
          constraintsMarkdown: "Constraints",
          notesMarkdown: "Notes",
          timeLimitMs: 1000,
          memoryLimitMb: 256,
          hiddenTestBundleKey: "problem-drafts/user-id/uploaded-bundle.json",
          hiddenTestBundleSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        }),
      });
    vi.stubGlobal("fetch", fetchMock);

    renderAuthoring(<ProblemAuthoringPage authEnabled />);

    fireEvent.change(screen.getByLabelText("Slug"), { target: { value: "two-sum-user" } });
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Two Sum User" } });
    fireEvent.change(screen.getByLabelText("Statement Markdown"), { target: { value: "Solve it" } });
    fireEvent.change(screen.getByLabelText("Input Markdown"), { target: { value: "Input" } });
    fireEvent.change(screen.getByLabelText("Output Markdown"), { target: { value: "Output" } });
    fireEvent.change(screen.getByLabelText("Constraints Markdown"), { target: { value: "Constraints" } });
    fireEvent.change(screen.getByLabelText("Notes Markdown"), { target: { value: "Notes" } });

    const bundleFile = new File([`{"cases":[{"input":"1\n","expectedOutput":"2\n"}]}`], "bundle.json", {
      type: "application/json",
    });
    fireEvent.change(screen.getByLabelText("Hidden test bundle file"), {
      target: { files: [bundleFile] },
    });
    fireEvent.click(screen.getByText("Upload bundle"));

    expect(await screen.findByDisplayValue("problem-drafts/user-id/uploaded-bundle.json")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Create draft" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    const [uploadUrl, uploadOptions] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(uploadUrl).toBe("/api/v1/problem-drafts/hidden-test-bundles");
    expect(uploadOptions.method).toBe("POST");
    expect(uploadOptions.headers).toMatchObject({
      Authorization: "Bearer session-token",
    });
    expect(uploadOptions.body).toBeInstanceOf(FormData);

    const [createUrl, createOptions] = fetchMock.mock.calls[1] as [string, RequestInit];
    expect(createUrl).toBe("/api/v1/problem-drafts");
    expect(JSON.parse(String(createOptions.body))).toMatchObject({
      hiddenTestBundleKey: "problem-drafts/user-id/uploaded-bundle.json",
      hiddenTestBundleSha256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    });
  });

  it("loads and saves an existing draft", async () => {
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
          slug: "two-sum-user",
          versionNumber: 1,
          lifecycleStatus: "draft",
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
          lifecycleStatus: "draft",
          title: "Updated Title",
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

    renderAuthoring(<ProblemAuthoringPage authEnabled slug="two-sum-user" />);

    expect(await screen.findByDisplayValue("Two Sum User")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Updated Title" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    const [url, options] = fetchMock.mock.calls[1] as [string, RequestInit];
    expect(url).toBe("/api/v1/problem-drafts/two-sum-user");
    expect(options.method).toBe("PATCH");
  });

  it("submits a draft for review from the edit page", async () => {
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
          slug: "two-sum-user",
          versionNumber: 1,
          lifecycleStatus: "draft",
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

    renderAuthoring(<ProblemAuthoringPage authEnabled slug="two-sum-user" />);

    expect(await screen.findByRole("button", { name: "Submit for review" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Submit for review" }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2);
    });

    const [url, options] = fetchMock.mock.calls[1] as [string, RequestInit];
    expect(url).toBe("/api/v1/problem-drafts/two-sum-user/submit-for-review");
    expect(options.method).toBe("POST");
  });

  it("keeps submit for review disabled until readiness checks pass", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi.fn().mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        slug: "two-sum-user",
        versionNumber: 1,
        lifecycleStatus: "draft",
        title: "Two Sum User",
        statementMarkdown: "Solve it",
        inputMarkdown: "Input",
        outputMarkdown: "Output",
        constraintsMarkdown: "",
        notesMarkdown: "Notes",
        timeLimitMs: 1000,
        memoryLimitMb: 256,
        hiddenTestBundleKey: "",
        hiddenTestBundleSha256: "",
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    renderAuthoring(<ProblemAuthoringPage authEnabled slug="two-sum-user" />);

    expect(await screen.findByRole("button", { name: "Submit for review" })).toBeDisabled();
    expect(screen.getAllByText("Missing").length).toBeGreaterThan(0);
    expect(screen.getByText("Hidden bundle")).toBeInTheDocument();
  });

  it("shows upload validation errors returned by the API", async () => {
    mockAuthState = {
      getToken: async () => "session-token",
      isLoaded: true,
      isSignedIn: true,
    };

    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: "invalid_hidden_test_bundle_upload" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    renderAuthoring(<ProblemAuthoringPage authEnabled />);

    const bundleFile = new File([`{"cases":[]}`], "bundle.json", {
      type: "application/json",
    });
    fireEvent.change(screen.getByLabelText("Hidden test bundle file"), {
      target: { files: [bundleFile] },
    });
    fireEvent.click(screen.getByText("Upload bundle"));

    expect(await screen.findByRole("alert")).toHaveTextContent("Hidden test bundle files must be valid JSON with at least one test case.");
  });
});
