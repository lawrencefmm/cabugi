import { render, screen, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";

import { AuthProvider, useAuth } from "./auth";

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
    document.cookie = "cabugi_local_test_auth_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/";
  });

  it("loads a signed-in local test session from a browser cookie", async () => {
    document.cookie = "cabugi_local_test_auth_token=test-token; path=/";

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
});
