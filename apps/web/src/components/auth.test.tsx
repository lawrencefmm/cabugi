import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { AuthProvider, AppAuthControls, LocalTestAuthControls, SignInButton, useAuth } from "./auth";
import { localTestAuthCookieName, localTestAuthProfileCookieName } from "../lib/local-test-auth";

let mockClerkAuthState: {
  getToken: () => Promise<string | null>;
  isLoaded: boolean;
  isSignedIn: boolean;
} = {
  getToken: async () => null,
  isLoaded: true,
  isSignedIn: false,
};

let mockClerkUserState: {
  user: {
    firstName: string | null;
    lastName: string | null;
    username: string | null;
    primaryEmailAddress: { emailAddress: string } | null;
  } | null;
} = {
  user: null,
};

const signOutMock = vi.fn(async () => undefined);

vi.mock("@clerk/nextjs", () => ({
  ClerkProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  SignInButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  SignUpButton: ({ children }: { children: ReactNode }) => <>{children}</>,
  useAuth: () => mockClerkAuthState,
  useClerk: () => ({ signOut: signOutMock }),
  useUser: () => mockClerkUserState,
}));

function AuthStateProbe() {
  const auth = useAuth();

  return <p>{!auth.isLoaded ? "loading" : auth.isSignedIn ? "signed-in" : "signed-out"}</p>;
}

describe("AuthProvider", () => {
  afterEach(() => {
    document.cookie = `${localTestAuthCookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
    document.cookie = `${localTestAuthProfileCookieName}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
    mockClerkAuthState = {
      getToken: async () => null,
      isLoaded: true,
      isSignedIn: false,
    };
    mockClerkUserState = { user: null };
    signOutMock.mockReset();
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

  it("shows sign in and create account in the header for signed-out Clerk mode", () => {
    render(
      <AuthProvider publishableKey="pk_test_example">
        <AppAuthControls />
      </AuthProvider>,
    );

    expect(screen.getByText("Account")).toBeInTheDocument();
    expect(screen.getByText("Signed out")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create account" })).toBeInTheDocument();
  });

  it("shows signed-in Clerk account state and supports sign out", async () => {
    mockClerkAuthState = {
      getToken: async () => "clerk-token",
      isLoaded: true,
      isSignedIn: true,
    };
    mockClerkUserState = {
      user: {
        firstName: "Dev",
        lastName: "User",
        username: "devuser",
        primaryEmailAddress: { emailAddress: "dev@example.com" },
      },
    };

    render(
      <AuthProvider publishableKey="pk_test_example">
        <AppAuthControls />
      </AuthProvider>,
    );

    expect(screen.getByText("Dev User")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));

    await waitFor(() => {
      expect(signOutMock).toHaveBeenCalledTimes(1);
    });
  });
});
