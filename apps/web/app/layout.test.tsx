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

vi.mock("next/navigation", () => ({
  usePathname: () => "/drafts",
}));

describe("RootLayout", () => {
  it("links Drafts to the drafts index", () => {
    render(RootLayout({ children: <main>content</main> }));

    expect(screen.getByRole("link", { name: "Drafts" })).toHaveAttribute("href", "/drafts");
  });

  it("includes the redesigned primary navigation", () => {
    render(RootLayout({ children: <main>content</main> }));

    expect(screen.getByRole("link", { name: "Problems" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Submissions" })).toHaveAttribute("href", "/submissions");
    expect(screen.getByRole("link", { name: "Moderation" })).toHaveAttribute("href", "/moderation/problem-drafts");
    expect(screen.getByRole("link", { name: "Docs" })).toHaveAttribute("href", "/api/openapi/v1.yaml");
  });
});
