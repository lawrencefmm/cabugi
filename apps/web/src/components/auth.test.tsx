import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { AuthProvider, LocalTestAuthControls, SignInButton, useAuth } from "./auth";
import { localTestAuthCookieName, localTestAuthProfileCookieName } from "../lib/local-test-auth";

vi.mock("@clerk/nextjs", () => ({
  ClerkProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => ({
    getToken: async () => null,
    isLoaded: true,
    isSignedIn: false,
  }),
}));

function AuthStateProbe() {
  const auth = useAuth();

  return <p>{!auth.isLoaded ? "loading" : auth.isSignedIn ? "signed-in" : "signed-out"}</p>;
}

describe("AuthProvider", () => {
  afterEach(() => {
    document.cookie = `${localTestAuthCookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
    document.cookie = `${localTestAuthProfileCookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
    vi.restoreAllMocks();
  });

  it("loads a signed-in local test session from a browser cookie", async () => {
    document.cookie = `${localTestAuthCookieName}=test-token; path=/`;

    render(
      <AuthProvider localTestAuthEnabled>
        <AuthStateProbe />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText("signed-in")).toBeInTheDocument();
    });
  });

  it("treats local test auth as signed out when the cookie is missing", async () => {
    render(
      <AuthProvider localTestAuthEnabled>
        <AuthStateProbe />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText("signed-out")).toBeInTheDocument();
    });
  });

  it("uses the local test sign-in button to create an author session", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        const payload = JSON.parse(String(init?.body ?? "{}")) as { profile?: string };
        document.cookie = `${localTestAuthCookieName}=token-${payload.profile}; path=/`;
        document.cookie = `${localTestAuthProfileCookieName}=${payload.profile}; path=/`;
        return new Response(JSON.stringify({ profile: payload.profile }), { status: 200 });
      }),
    );

    render(
      <AuthProvider localTestAuthEnabled>
        <SignInButton mode="modal">
          <button type="button">Sign in locally</button>
        </SignInButton>
        <AuthStateProbe />
      </AuthProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Sign in locally" }));

    await waitFor(() => {
      expect(screen.getByText("signed-in")).toBeInTheDocument();
    });
    expect(document.cookie).toContain(`${localTestAuthProfileCookieName}=author`);
  });

  it("switches the local test session to moderator from the header controls", async () => {
    document.cookie = `${localTestAuthCookieName}=token-author; path=/`;
    document.cookie = `${localTestAuthProfileCookieName}=author; path=/`;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        const payload = JSON.parse(String(init?.body ?? "{}")) as { profile?: string };
        document.cookie = `${localTestAuthCookieName}=token-${payload.profile}; path=/`;
        document.cookie = `${localTestAuthProfileCookieName}=${payload.profile}; path=/`;
        return new Response(JSON.stringify({ profile: payload.profile }), { status: 200 });
      }),
    );

    render(
      <AuthProvider localTestAuthEnabled>
        <LocalTestAuthControls />
      </AuthProvider>,
    );

    await waitFor(() => {
      expect(screen.getByText("Author session")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Switch to moderator" }));

    await waitFor(() => {
      expect(screen.getByText("Moderator session")).toBeInTheDocument();
    });
    expect(document.cookie).toContain(`${localTestAuthProfileCookieName}=moderator`);
  });
});
