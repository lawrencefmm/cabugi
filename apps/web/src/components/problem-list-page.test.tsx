import { render, screen } from "@testing-library/react";

import { ProblemListPage } from "./problem-list-page";

function mockFetch(response: Partial<Response> & { json?: () => Promise<unknown> }) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ problems: [] }),
      ...response,
    }),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("ProblemListPage", () => {
  it("renders loading first and then the published problems", async () => {
    mockFetch({
      json: async () => ({
        problems: [{ slug: "two-sum", title: "Two Sum", timeLimitMs: 1000, memoryLimitMb: 256 }],
      }),
    });

    render(<ProblemListPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading published problems");
    expect(await screen.findByRole("link", { name: "Two Sum" })).toHaveAttribute("href", "/problems/two-sum");
    expect(screen.getByRole("table", { name: "Published problems" })).toBeInTheDocument();
    expect(screen.getByText("two-sum")).toBeInTheDocument();
  });

  it("renders an error state when the API fails", async () => {
    mockFetch({ ok: false, status: 500 });

    render(<ProblemListPage />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Problem list unavailable");
  });
});
