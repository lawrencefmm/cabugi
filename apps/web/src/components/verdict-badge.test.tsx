import { render, screen } from "@testing-library/react";

import { VerdictBadge } from "./verdict-badge";

describe("VerdictBadge", () => {
  it("renders the human-readable verdict label", () => {
    render(<VerdictBadge verdict="time_limit_exceeded" />);

    expect(screen.getByText("Time Limit Exceeded")).toBeInTheDocument();
  });
});
