import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";

import RootLayout from "./layout";

vi.mock("../src/components/app-providers", () => ({
  AppProviders: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("../src/components/auth", () => ({
  AppAuthControls: () => <div>auth controls</div>,
  AuthProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

describe("RootLayout", () => {
  it("links Drafts to the drafts index", () => {
    render(RootLayout({ children: <main>content</main> }));

    expect(screen.getByRole("link", { name: "Drafts" })).toHaveAttribute("href", "/drafts");
  });
});
