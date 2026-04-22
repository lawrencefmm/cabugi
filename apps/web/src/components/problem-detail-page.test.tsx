import { render, screen } from "@testing-library/react";

import { ProblemDetailPage } from "./problem-detail-page";

function mockFetch(response: Partial<Response> & { json?: () => Promise<unknown> }) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        slug: "two-sum",
        title: "Two Sum",
        statementMarkdown: "Solve it with math $a + b$.",
        inputMarkdown: "Two integers.",
        outputMarkdown: "Their sum.",
        constraintsMarkdown: "`1 <= n <= 10^5`",
        notesMarkdown: "No extra notes.",
        timeLimitMs: 1000,
        memoryLimitMb: 256,
      }),
      ...response,
    }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("ProblemDetailPage", () => {
  it("renders the full published problem detail", async () => {
    mockFetch({});

    render(<ProblemDetailPage slug="two-sum" />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading problem");
    expect(await screen.findByRole("heading", { name: "Two Sum" })).toBeInTheDocument();
    expect(screen.getByText("Statement")).toBeInTheDocument();
    expect(screen.getByText("Constraints")).toBeInTheDocument();
    expect(screen.getByText("1000 ms")).toBeInTheDocument();
    expect(screen.getByText("256 MB")).toBeInTheDocument();
  });

  it("renders a not found message for a missing problem", async () => {
    mockFetch({ ok: false, status: 404 });

    render(<ProblemDetailPage slug="missing-problem" />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Problem unavailable");
    expect(screen.getByText("Problem not found.")).toBeInTheDocument();
  });
});
